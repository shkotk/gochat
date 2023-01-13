package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shkotk/gochat/client/apiclient"
	"github.com/shkotk/gochat/common/apimodels/events"
)

var (
	senderNameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	systemMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type chatKeys struct {
	Enter  key.Binding
	Escape key.Binding
}

func (k chatKeys) Bindings() []key.Binding {
	return []key.Binding{k.Enter, k.Escape}
}

type Chat struct {
	keys chatKeys

	width  int
	height int

	messages []string
	textarea textarea.Model
	help     help.Model

	client *apiclient.ApiClient
}

func NewChat(width, height int, client *apiclient.ApiClient) Chat {
	m := Chat{
		keys: chatKeys{
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "send"),
			),
			Escape: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "leave"),
			),
		},

		textarea: textarea.New(),
		help:     help.New(),

		client: client,
	}

	m.textarea.KeyMap.InsertNewline.SetEnabled(false)
	m.textarea.SetHeight(1)
	m.textarea.ShowLineNumbers = false
	m.textarea.Focus()

	m.setSize(width, height)

	return m
}

func (m Chat) Init() tea.Cmd {
	return tea.Batch(readEventCmd(m.client), textarea.Blink)
}

func (m Chat) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)

	case EventMsg:
		line := ""
		switch event := msg.Event.(type) {
		case events.NewMessage:
			line = fmt.Sprintf(
				"%s: %s", senderNameStyle.Render(event.Producer), event.Text)
		case events.SystemMessage:
			line = systemMessageStyle.Render(event.Text)
		}

		if line != "" {
			m.messages = append(m.messages, line)
		}

		return m, readEventCmd(m.client)

	case ChatConnClosedMsg:
		return m, func() tea.Msg { return BackToHubMsg{} }

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			message := strings.TrimSpace(m.textarea.Value())
			if message == "" {
				return m, nil
			}
			m.textarea.Reset()
			return m, func() tea.Msg {
				err := m.client.WriteEvent(events.NewMessage{Text: message})
				if err != nil {
					return ErrorMsg(err.Error())
				}
				return nil
			}
		case key.Matches(msg, m.keys.Escape):
			m.client.Leave()
			return m, func() tea.Msg { return BackToHubMsg{} }
		}
	}

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Chat) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-m.textarea.Height()-1).
			Render(strings.Join(m.messages, "\n")), // TODO find some better way to display messages; bubbles.viewport seems almost ideal but it crops long messages instead of wrapping them
		m.textarea.View(),
		m.help.ShortHelpView(m.keys.Bindings()),
	)
}

func (m *Chat) setSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width
	m.textarea.SetWidth(width)
}

type EventMsg struct {
	Event any
}

type ChatConnClosedMsg struct{}

func readEventCmd(client *apiclient.ApiClient) tea.Cmd {
	return func() tea.Msg {
		event, more, err := client.ReadEvent()
		if err != nil {
			return ErrorMsg(err.Error())
		}
		if !more {
			return ChatConnClosedMsg{}
		}
		return EventMsg{event}
	}
}
