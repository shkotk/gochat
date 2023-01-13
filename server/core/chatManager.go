package core

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type ChatManager struct {
	chats     map[string]*Chat
	chatsLock sync.RWMutex

	logger *logrus.Logger
}

func NewChatManager(logger *logrus.Logger) *ChatManager {
	return &ChatManager{
		chats:  make(map[string]*Chat),
		logger: logger,
	}
}

// Creates new Chat with specified chat name.
func (m *ChatManager) Create(chatName string) error {
	m.chatsLock.Lock()
	defer m.chatsLock.Unlock()

	if _, ok := m.chats[chatName]; ok {
		return fmt.Errorf("chat with name '%v' already exists", chatName)
	}

	chat := NewChat(chatName, m.logger)
	m.chats[chatName] = chat
	go chat.Run()

	return nil
}

// Lists all existing chats.
func (m *ChatManager) List() ([]string, error) {
	m.chatsLock.RLock()
	defer m.chatsLock.RUnlock()

	chatNames := make([]string, len(m.chats))
	i := 0
	for key := range m.chats {
		chatNames[i] = key
		i++
	}

	return chatNames, nil
}

// Processes join chat request using provided connection and identifiers.
func (m *ChatManager) Join(username string, conn *websocket.Conn, chatName string) error {
	m.chatsLock.RLock()
	defer m.chatsLock.RUnlock()

	chat, ok := m.chats[chatName]
	if !ok {
		return fmt.Errorf("chat '%v' does not exist", chatName)
	}

	if err := chat.Join(username, conn); err != nil {
		return err
	}

	return nil
}
