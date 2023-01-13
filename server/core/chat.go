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
	events  chan any

	logger *logrus.Logger
}

func NewChat(chatName string, logger *logrus.Logger) *Chat {
	return &Chat{
		Name:    chatName,
		members: make(map[string]*Client),
		events:  make(chan any),
		logger:  logger,
	}
}

func (c *Chat) Join(username string, conn *websocket.Conn) error {
	err := make(chan error)
	c.events <- joinEvent{
		Producer: username,
		Conn:     conn,
		Err:      err,
	}

	return <-err // returns as soon as join event is processed by main loop
}

func (c *Chat) Run() {
	for {
		switch event := (<-c.events).(type) {

		case joinEvent:
			if _, ok := c.members[event.Producer]; ok {
				event.Err <- fmt.Errorf("client '%v' is already in chat '%v'", event.Producer, c.Name)
				continue
			}

			client := NewClient(event.Producer, event.Conn, c.logger)
			c.members[event.Producer] = client
			go client.Run(c.events)
			event.Err <- nil

			c.broadcast(events.SystemMessage{
				Text: fmt.Sprintf("%v joined chat", event.Producer),
				Time: time.Now(),
			})

		case events.NewMessage:
			c.broadcast(event)

		case leaveEvent:
			if _, ok := c.members[event.Producer]; !ok {
				// TODO log
				continue
			}
			delete(c.members, event.Producer)

			c.broadcast(events.SystemMessage{
				Text: fmt.Sprintf("%v left chat", event.Producer),
				Time: time.Now(),
			})

		default:
			// TODO log
		}
	}
}

func (c *Chat) broadcast(event any) {
	for _, client := range c.members {
		go client.Write(event)
	}
}

type joinEvent struct {
	Producer string
	Conn     *websocket.Conn
	Err      chan error
}

type leaveEvent struct {
	Producer string
}
