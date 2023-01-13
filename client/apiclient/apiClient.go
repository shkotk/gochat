package apiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shkotk/gochat/common/apimodels/events"
	"github.com/shkotk/gochat/common/apimodels/requests"
	"github.com/shkotk/gochat/common/apimodels/responses"
)

type ApiClient struct {
	client http.Client
	host   string
	token  token

	chattingLock sync.Mutex
	conn         *websocket.Conn
	in           chan any
	out          chan any
}

func New(host string) *ApiClient {
	return &ApiClient{
		client: http.Client{
			Timeout: 30 * time.Second,
		},
		host: host,
	}
}

func (c *ApiClient) Login(authRequest requests.Auth) (time.Time, error) {
	jsonBody, err := json.Marshal(authRequest)
	if err != nil {
		return time.Time{}, err
	}

	u := url.URL{Scheme: "https", Host: c.host, Path: "/token/get"}
	request, err := http.NewRequest(
		http.MethodGet, u.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return time.Time{}, err
	}

	response, err := c.client.Do(request)
	if err != nil {
		return time.Time{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return time.Time{}, extractError(response, "get token")
	}

	tokenResponse := &responses.Token{}
	err = json.NewDecoder(response.Body).Decode(tokenResponse)
	if err != nil {
		return time.Time{}, err
	}

	c.token.Set(tokenResponse.Token)

	return tokenResponse.ExpiresAt, nil
}

func (c *ApiClient) RefreshToken() (time.Time, error) {
	u := url.URL{Scheme: "https", Host: c.host, Path: "/token/refresh"}
	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return time.Time{}, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token.Get()))

	response, err := c.client.Do(request)
	if err != nil {
		return time.Time{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return time.Time{}, extractError(response, "refresh token")
	}

	tokenResponse := &responses.Token{}
	err = json.NewDecoder(response.Body).Decode(tokenResponse)
	if err != nil {
		return time.Time{}, err
	}

	c.token.Set(tokenResponse.Token)

	return tokenResponse.ExpiresAt, nil
}

func (c *ApiClient) GetChats() ([]string, error) {
	u := url.URL{Scheme: "https", Host: c.host, Path: "/chat/list"}
	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token.Get()))

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, extractError(response, "get chats")
	}

	chatsResponse := &responses.Chats{}
	err = json.NewDecoder(response.Body).Decode(chatsResponse)
	if err != nil {
		return nil, err
	}

	return chatsResponse.Chats, nil
}

func (c *ApiClient) Create(chatName string) error {
	u := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   "/chat/create/" + url.PathEscape(chatName),
	}
	request, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token.Get()))

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return extractError(response, "create chat")
	}

	return nil
}

func (c *ApiClient) Join(chatName string) error {
	if !c.chattingLock.TryLock() {
		return errors.New("can't join more then one chat at once")
	}

	u := url.URL{
		Scheme: "wss",
		Host:   c.host,
		Path:   "/chat/join/" + url.PathEscape(chatName),
	}
	conn, response, err := websocket.DefaultDialer.Dial(u.String(), http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", c.token.Get())},
	})
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("got join response with unexpected status code '%v'", response.Status)
	}

	c.conn = conn
	c.in = make(chan any)
	c.out = make(chan any)

	go c.readLoop()
	go c.writeLoop()

	return nil
}

func (c *ApiClient) readLoop() {
	defer func() {
		close(c.in)
		c.in = nil
		close(c.out)
	}()

	for {
		mt, message, err := c.conn.ReadMessage()
		if err != nil {
			// TODO log
			return
		}
		if mt != websocket.TextMessage {
			// TODO log
			continue
		}

		event, err := events.Parse(message)
		if err != nil {
			// TODO log
			continue
		}

		c.in <- event
	}
}

func (c *ApiClient) writeLoop() {
	defer func() {
		c.out = nil
		c.chattingLock.Unlock()
	}()

	for event := range c.out {
		message, err := events.Serialize(event)
		if err != nil {
			// TODO log
			continue
		}

		err = c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			// TODO log
			return
		}
	}

	// out channel was closed from outside
	c.conn.WriteMessage(websocket.CloseMessage, nil)
}

func (c *ApiClient) ReadEvent() (event any, more bool, err error) {
	if c.in == nil {
		return nil, false, errors.New("no active connection, can't read event")
	}

	event, ok := <-c.in
	if !ok {
		return nil, false, nil
	}

	return event, true, nil
}

func (c *ApiClient) WriteEvent(event any) error {
	if c.out == nil {
		return errors.New("no active connection, can't write event")
	}

	select {
	case c.out <- event:
		return nil
	case <-time.After(time.Minute):
		return errors.New("timed out writing event")
	}
}

func (c *ApiClient) Leave() {
	c.conn.Close()
}

func extractError(response *http.Response, action string) error {
	errorResponse := &responses.Error{}
	err := json.NewDecoder(response.Body).Decode(errorResponse)
	if err != nil {
		return fmt.Errorf("failed to %s, got response with code %d",
			action, response.StatusCode)
	}
	return fmt.Errorf("failed to %s, got response with code %d, message: '%s'",
		action, response.StatusCode, errorResponse.Error)
}

type token struct {
	value string
	lock  sync.RWMutex
}

func (t *token) Set(newValue string) {
	t.lock.Lock()
	t.value = newValue
	t.lock.Unlock()
}

func (t *token) Get() string {
	t.lock.RLock()
	v := t.value
	t.lock.RUnlock()
	return v
}
