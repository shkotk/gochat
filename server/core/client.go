package core

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shkotk/gochat/common/apimodels/events"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	Username   string
	conn       *websocket.Conn
	writeQueue chan any

	logger *logrus.Logger
}

func NewClient(username string, conn *websocket.Conn, logger *logrus.Logger) *Client {
	client := &Client{
		Username:   username,
		conn:       conn,
		writeQueue: make(chan any),
		logger:     logger,
	}

	return client
}

func (c *Client) Run(out chan any) {
	go c.readLoop(out)
	c.writeLoop()
}

// Reads WebSocket messages and writes them to provided channel.
func (c *Client) readLoop(eventsChan chan any) {
	defer func() {
		eventsChan <- leaveEvent{Producer: c.Username} // signal caller
		close(c.writeQueue)                            // signal write loop
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.logger.Debugf("got pong from %s", c.Username)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		mt, message, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.WithError(err).Debugf("error occurred while reading message from %s", c.Username)
			return
		}
		if mt != websocket.TextMessage {
			c.logger.Debugf("got message of unexpected type '%v' from %s", mt, c.Username)
			continue
		}

		event, err := events.Parse(message)
		if err != nil {
			c.logger.WithError(err).Debugf("error occurred while parsing message from %s: '%s'", c.Username, message)
			continue
		}

		switch e := event.(type) {

		case events.NewMessage:
			e.Producer = c.Username
			e.Time = time.Now()
			event = e

		default:
			panic(fmt.Sprintf("unexpected event type '%T'", event))
		}

		c.logger.Debugf("got event from %s: '%v'", c.Username, event)
		eventsChan <- event
	}
}

// Populates internal write queue with provided event to be written as WebScoket message.
func (c *Client) Write(event any) error {
	select {
	case c.writeQueue <- event:
		return nil
	case <-time.After(time.Minute):
		return fmt.Errorf("timed out writing message to %s", c.Username)
	}
}

// Reads events from write queue and writes them as WebSocket messages.
func (c *Client) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, more := <-c.writeQueue:
			if !more {
				return
			}

			message, err := events.Serialize(event)
			if err != nil {
				c.logger.WithError(err).Debugf("failed to serialize event '%v'")
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.logger.Debugf("sending message '%s' to %s", message, c.Username)
			if err = c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.WithError(err).Debugf("error occurred while writing message to %s", c.Username)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.logger.Debugf("sending ping to %s", c.Username)
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithError(err).Debugf("ping failed for %s", c.Username)
				return
			}
		}
	}
}
