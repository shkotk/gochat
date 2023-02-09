package services

import (
	"fmt"
	"sync"

	"github.com/shkotk/gochat/server/interfaces"
	"github.com/sirupsen/logrus"
)

type ChatManager struct {
	chats     map[string]*Chat
	chatsLock sync.RWMutex

	eventsPreProcessor interfaces.EventPreProcessor
	logger             *logrus.Logger
}

func NewChatManager(
	logger *logrus.Logger,
	eventsPreProcessor interfaces.EventPreProcessor,
) *ChatManager {
	return &ChatManager{
		chats:              make(map[string]*Chat),
		eventsPreProcessor: eventsPreProcessor,
		logger:             logger,
	}
}

func (m *ChatManager) Create(chatName string) error {
	m.chatsLock.Lock()
	defer m.chatsLock.Unlock()

	if _, ok := m.chats[chatName]; ok {
		return fmt.Errorf("chat with name '%v' already exists", chatName)
	}

	chat := NewChat(chatName, m.eventsPreProcessor, m.logger)
	m.chats[chatName] = chat
	go chat.Run()

	return nil
}

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

func (m *ChatManager) AddClient(client interfaces.Client, chatName string) error {
	m.chatsLock.RLock()
	defer m.chatsLock.RUnlock()

	chat, ok := m.chats[chatName]
	if !ok {
		return fmt.Errorf("chat '%v' does not exist", chatName)
	}

	if err := chat.AddClient(client); err != nil {
		return err
	}

	return nil
}
