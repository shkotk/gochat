package models

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shkotk/gochat/client/apiclient"
)

type hubKeys struct {
	Create  key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Escape  key.Binding
}

type hubState int

const (
	chatsList hubState = iota
	createChatMenu
)

type Hub struct {
	keys hubKeys

	width     int
	height    int
	formWidth int

	state hubState
	list  list.Model
	input textinput.Model
	err   string
	help  help.Model

	client *apiclient.ApiClient
}

func NewHub(width, height int, client *apiclient.ApiClient) tea.Model {
	m := Hub{
		keys: hubKeys{
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "confirm"),
			),
			Refresh: key.NewBinding(
				key.WithKeys("ctrl+r"),
				key.WithHelp("ctrl+r", "refresh"),
			),
			Create: key.NewBinding(
				key.WithKeys("ctrl+n"),
				key.WithHelp("ctrl+n", "create"),
			),
			Escape: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		},

		input: textinput.New(),
		state: chatsList,
		help:  help.New(),

		client: client,
	}

	listDelegate := list.NewDefaultDelegate()
	listDelegate.ShowDescription = false
	m.list = list.New([]list.Item{}, listDelegate, 0, 0)
	m.list.Title = "Chats"
	m.list.SetStatusBarItemName("chat", "chats")
	m.list.DisableQuitKeybindings()

	m.input.Prompt = "> "
	m.input.PromptStyle = m.input.PlaceholderStyle.Copy()
	m.input.Placeholder = "new chat name"

	m.setSize(width, height)

	return m
}

func (m Hub) Init() tea.Cmd {
	return fetchChatsCmd(m.client)
}

func (m Hub) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)

	case ErrorMsg:
		m.err = string(msg)

	case chatsListMsg:
		items := make([]list.Item, len(msg.Chats))
		for i, chatName := range msg.Chats {
			items[i] = item(chatName)
		}
		cmd := m.list.SetItems(items)
		return m, cmd

	case chatCreatedMsg:
		m.state = chatsList
		m.input.Reset()
		m.input.Blur()
		return m, fetchChatsCmd(m.client) // TODO join created chat

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break // let list handle all key presses while setting a filter
		}

		if key.Matches(msg, m.keys.Create, m.keys.Enter) {
			m.err = "" // reset error on changing focus
		}

		switch {
		case key.Matches(msg, m.keys.Create):
			m.state = createChatMenu
			cmd := m.input.Focus()
			return m, cmd
		case key.Matches(msg, m.keys.Refresh):
			return m, fetchChatsCmd(m.client)
		case key.Matches(msg, m.keys.Enter):
			switch m.state {
			case chatsList:
				selectedItem := m.list.SelectedItem()
				if selectedItem == nil {
					return m, nil
				}
				chatName := string(selectedItem.(item))
				return m, joinChatCmd(m.client, chatName)
			case createChatMenu:
				// TODO validate
				// TODO start spinner or smthng
				return m, createChatCmd(m.client, m.input.Value())
			default:
				log.Panicf("unexpected state %v", m.state)
			}
		case key.Matches(msg, m.keys.Escape):
			switch m.state {
			case chatsList:
				// TODO logout?
			case createChatMenu:
				m.state = chatsList
				m.input.Blur()
				m.input.Reset()
				return m, nil
			default:
				log.Panicf("unexpected state %v", m.state)
			}
		}
	}

	var cmd tea.Cmd
	switch m.state {
	case chatsList:
		m.list, cmd = m.list.Update(msg)
	case createChatMenu:
		m.input, cmd = m.input.Update(msg)
	default:
		log.Panicf("unexpected state %v", m.state)
	}

	return m, cmd
}

func (m Hub) View() string {
	switch m.state {
	case chatsList:
		// update additional keys to show relevant help
		var additionalKeys func() []key.Binding
		switch m.list.FilterState() {
		case list.Filtering:
			additionalKeys = func() []key.Binding { return []key.Binding{} }
		case list.Unfiltered, list.FilterApplied:
			joinBinding := m.keys.Enter
			joinBinding.SetHelp("enter", "join")
			additionalKeys = func() []key.Binding {
				return []key.Binding{
					m.keys.Create, joinBinding, m.keys.Refresh,
				}
			}
		}
		m.list.AdditionalFullHelpKeys = additionalKeys
		m.list.AdditionalShortHelpKeys = additionalKeys

		return m.list.View()

	case createChatMenu:
		createBinding := m.keys.Enter
		createBinding.SetHelp("enter", "create")
		return lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.Place(
				m.width,
				m.height-1, // one line for help
				lipgloss.Center,
				lipgloss.Center,
				lipgloss.JoinVertical(
					lipgloss.Left,
					itemStyle.Render(m.input.View()),
					infoView(m.err, m.formWidth, 3, m.err != ""),
				),
			),
			m.help.ShortHelpView([]key.Binding{createBinding, m.keys.Escape}),
		)

	default:
		panic(fmt.Sprintf("unexpected state %v", m.state))
	}
}

func (m *Hub) setSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
	m.help.Width = width

	if width < maxFormWidth {
		m.formWidth = width
	} else {
		m.formWidth = maxFormWidth
	}

	m.input.Width = m.formWidth - 3
}

type chatsListMsg struct {
	Chats []string
}

func fetchChatsCmd(client *apiclient.ApiClient) tea.Cmd {
	return func() tea.Msg {
		chats, err := client.GetChats()
		if err != nil {
			return ErrorMsg(err.Error())
		}

		return chatsListMsg{Chats: chats}
	}
}

type chatCreatedMsg struct{}

func createChatCmd(client *apiclient.ApiClient, chatName string) tea.Cmd {
	return func() tea.Msg {
		err := client.Create(chatName)
		if err != nil {
			return ErrorMsg(err.Error())
		}

		return chatCreatedMsg{}
	}
}

type ChatJoinedMsg struct{}

func joinChatCmd(client *apiclient.ApiClient, chatName string) tea.Cmd {
	return func() tea.Msg {
		err := client.Join(chatName)
		if err != nil {
			return ErrorMsg(err.Error())
		}

		return ChatJoinedMsg{}
	}
}

// List item implementation
type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }
