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

	// Give up on sending event to writeQueue after this period.
	writeEnqueueTimeout = time.Minute
)

type Client struct {
	Username   string
	conn       *websocket.Conn
	writeQueue chan any

	chat   *Chat
	logger *logrus.Logger
}

func NewClient(username string, conn *websocket.Conn, chat *Chat, logger *logrus.Logger) *Client {
	client := &Client{
		Username:   username,
		conn:       conn,
		writeQueue: make(chan any),
		chat:       chat,
		logger:     logger,
	}

	return client
}

// Launches read and write loops in separate goroutines.
func (c *Client) Start() {
	go c.readLoop()
	go c.writeLoop()
}

// Reads WebSocket messages and writes them to provided channel.
func (c *Client) readLoop() {
	defer func() {
		c.chat.Leave(c.Username) // signal chat
		close(c.writeQueue)      // signal write loop
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(
		func(string) error {
			c.conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

	for {
		mt, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Warnf("client: failed to read message from %s", c.Username)
			}
			return
		}

		if mt != websocket.TextMessage {
			c.logger.Warnf("client: got message of unexpected type '%v' from %s", mt, c.Username)
			continue
		}

		event, err := events.Parse(message)
		if err != nil {
			c.logger.WithError(err).Warnf(
				"client: failed to parse message from %s; message: '%s'", c.Username, message)
			continue
		}

		switch e := event.(type) {

		case events.NewMessage:
			e.Producer = c.Username
			e.Time = time.Now()
			event = e

		default:
			c.logger.Warnf("client: got unexpected event type '%T'", event)
		}

		c.chat.Send(event)
	}
}

// Populates internal write queue with provided event to be written as WebScoket message.
func (c *Client) Write(event any) error {
	select {
	case c.writeQueue <- event:
		return nil
	case <-time.After(writeEnqueueTimeout):
		return fmt.Errorf("client: timed out writing message to %s", c.Username)
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
				c.logger.WithError(err).Errorf(
					"client: failed to serialize event of type %T, value: '%v'", event, event)
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err = c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.WithError(err).Warnf("client: error writing message to %s", c.Username)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithError(err).Errorf("client: error sending ping to %s", c.Username)
				return
			}
		}
	}
}
