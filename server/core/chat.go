package core

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shkotk/gochat/common/apimodels/events"
	"github.com/sirupsen/logrus"
)

type Chat struct {
	Name string

	members map[string]*Client

	events        chan any
	joinRequests  chan joinRequest
	leaveRequests chan leaveRequest

	logger *logrus.Logger
}

func NewChat(chatName string, logger *logrus.Logger) *Chat {
	return &Chat{
		Name:          chatName,
		members:       make(map[string]*Client),
		events:        make(chan any),
		joinRequests:  make(chan joinRequest),
		leaveRequests: make(chan leaveRequest),
		logger:        logger,
	}
}

func (c *Chat) Join(username string, conn *websocket.Conn) error {
	err := make(chan error)
	c.joinRequests <- joinRequest{
		Username: username,
		Conn:     conn,
		Err:      err,
	}

	return <-err // returns as soon as join event is processed by main loop
}

func (c *Chat) Leave(username string) {
	c.leaveRequests <- leaveRequest{
		Username: username,
	}
}

func (c *Chat) Send(event any) {
	c.events <- event
}

// Loop processing join and leave requests, events from clients in chat.
func (c *Chat) Run() {
	for {
		select {
		case request := <-c.joinRequests:
			if _, ok := c.members[request.Username]; ok {
				request.Err <- fmt.Errorf(
					"client '%v' is already in chat '%v'", request.Username, c.Name)
				continue
			}

			client := NewClient(request.Username, request.Conn, c, c.logger)
			c.members[request.Username] = client
			client.Start()

			request.Err <- nil

			c.broadcast(events.SystemMessage{
				Text: fmt.Sprintf("%v joined chat", request.Username),
				Time: time.Now(),
			})

		case request := <-c.leaveRequests:
			if _, ok := c.members[request.Username]; !ok {
				continue
			}
			delete(c.members, request.Username)

			c.broadcast(events.SystemMessage{
				Text: fmt.Sprintf("%v left chat", request.Username),
				Time: time.Now(),
			})

		case event := <-c.events:
			c.processEvent(event)
		}
	}
}

func (c *Chat) processEvent(event any) {
	switch event.(type) {
	case events.NewMessage:
		c.broadcast(event)

	default:
		c.logger.Errorf("chat: can't process event of type %T: '%v'", event, event)
	}
}

func (c *Chat) broadcast(event any) {
	for _, client := range c.members {
		go func(client *Client) {
			err := client.Write(event)
			if err != nil {
				c.logger.WithError(err).Warnf(
					"chat: failed writing message to %s", client.Username)
			}
		}(client)
	}
}

type joinRequest struct {
	Username string
	Conn     *websocket.Conn
	Err      chan error
}

type leaveRequest struct {
	Username string
}
