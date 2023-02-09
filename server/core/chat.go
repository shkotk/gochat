package core

import (
	"fmt"
	"time"

	"github.com/shkotk/gochat/common/apimodels/events"
	"github.com/shkotk/gochat/server/interfaces"
	"github.com/sirupsen/logrus"
)

type Chat struct {
	Name string

	members map[string]interfaces.Client

	events        chan any
	joinRequests  chan joinRequest
	leaveRequests chan leaveRequest

	eventsPreProcessor interfaces.EventPreProcessor
	logger             *logrus.Logger
}

func NewChat(
	chatName string,
	eventsPreProcessor interfaces.EventPreProcessor,
	logger *logrus.Logger,
) *Chat {
	return &Chat{
		Name:               chatName,
		members:            make(map[string]interfaces.Client),
		events:             make(chan any),
		joinRequests:       make(chan joinRequest),
		leaveRequests:      make(chan leaveRequest),
		eventsPreProcessor: eventsPreProcessor,
		logger:             logger,
	}
}

func (c *Chat) AddClient(client interfaces.Client) error {
	err := make(chan error)
	c.joinRequests <- joinRequest{
		Client: client,
		Err:    err,
	}

	return <-err
}

// Loop processing join and leave requests, events from clients in chat.
func (c *Chat) Run() {
	for {
		select {
		case request := <-c.joinRequests:
			c.processJoinRequest(request)
		case request := <-c.leaveRequests:
			c.processLeaveRequest(request)
		case event := <-c.events:
			c.broadcast(event)
		}
	}
}

func (c *Chat) processJoinRequest(request joinRequest) {
	client := request.Client
	if _, ok := c.members[client.ID()]; ok {
		request.Err <- fmt.Errorf(
			"client '%s' is already in chat '%s'", client.ID(), c.Name)
		return
	}

	request.Err <- nil
	c.broadcast(&events.SystemMessage{
		Text: fmt.Sprintf("%s joined chat", client.ID()),
		Time: time.Now(),
	})
	c.members[client.ID()] = client

	go c.pumpMessages(client)
}

func (c *Chat) processLeaveRequest(request leaveRequest) {
	if _, ok := c.members[request.ClientID]; !ok {
		c.logger.Warnf(
			"chat: can't process leave request, user '%s' is not in chat", request.ClientID)
		return
	}

	delete(c.members, request.ClientID)
	c.broadcast(&events.SystemMessage{
		Text: fmt.Sprintf("%s left chat", request.ClientID),
		Time: time.Now(),
	})
}

// Reads incoming events from client and pumps them to chat events channel.
func (c *Chat) pumpMessages(client interfaces.Client) {
	for {
		select {
		case event := <-client.In():
			err := c.eventsPreProcessor.PreProcess(event, client)
			if err != nil {
				c.logger.WithError(err).Warnf(
					"chat: error pre-processing event from '%s'", client.ID())
				continue
			}
			c.events <- event

		case <-client.Done():
			c.leaveRequests <- leaveRequest{client.ID()}
			return
		}
	}
}

func (c *Chat) broadcast(event any) {
	for _, client := range c.members {
		go send(event, client)
	}
}

func send(event any, client interfaces.Client) {
	select {
	case client.Out() <- event:
	case <-client.Done():
	}
}

type joinRequest struct {
	Client interfaces.Client
	Err    chan error
}

type leaveRequest struct {
	ClientID string
}
