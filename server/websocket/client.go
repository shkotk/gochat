package websocket

import (
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
	maxMessageSize = 1024

	// Give up on sending event to writeQueue after this period.
	writeEnqueueTimeout = time.Minute
)

var expectedCloseCodes = []int{
	websocket.CloseGoingAway,
	websocket.CloseAbnormalClosure,
}

type Client struct {
	username string
	conn     *websocket.Conn

	in   chan any
	out  chan any
	done chan struct{}

	logger *logrus.Logger
}

func NewClient(username string, conn *websocket.Conn, logger *logrus.Logger) *Client {
	client := &Client{
		username: username,
		conn:     conn,
		in:       make(chan any),
		out:      make(chan any),
		done:     make(chan struct{}),
		logger:   logger,
	}

	return client
}

func (c *Client) ID() string {
	return c.username
}

func (c *Client) In() <-chan any {
	return c.in
}

func (c *Client) Out() chan<- any {
	return c.out
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Launches read and write loops in separate goroutines and waits for them to complete.
func (c *Client) Run() {
	defer close(c.done)

	doneReading := c.startReading()
	cancelWriting := make(chan struct{})
	doneWriting := c.startWriting(cancelWriting)

	<-doneReading
	close(cancelWriting)
	<-doneWriting
}

// Starts goroutine which reads WebSocket messages and writes them to in channel.
func (c *Client) startReading() <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

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
				if websocket.IsUnexpectedCloseError(err, expectedCloseCodes...) {
					c.logger.WithError(err).Warnf("client: failed to read message from %s", c.username)
				}
				return
			}

			if mt != websocket.TextMessage {
				c.logger.Warnf("client: got message of unexpected type '%v' from %s", mt, c.username)
				continue
			}

			event, err := events.Parse(message)
			if err != nil {
				c.logger.WithError(err).Warnf(
					"client: failed to parse message from %s; message: '%s'", c.username, message)
				continue
			}

			c.in <- event
		}
	}()

	return done
}

// Starts goroutine which reads events from in channel and writes them as WebSocket messages.
func (c *Client) startWriting(cancel <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			c.conn.Close()
			close(done)
		}()

		for {
			select {
			case event := <-c.out:
				message, err := events.Serialize(event)
				if err != nil {
					c.logger.WithError(err).Errorf(
						"client: failed to serialize event of type %T, value: '%v'", event, event)
					continue
				}

				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err = c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
					c.logger.WithError(err).Warnf("client: error writing message to %s", c.username)
					return
				}

			case <-ticker.C:
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.logger.WithError(err).Errorf("client: error sending ping to %s", c.username)
					return
				}

			case <-cancel:
				return
			}
		}
	}()

	return done
}
