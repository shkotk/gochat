package models

import (
	"regexp"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type hostKeys struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

func (k hostKeys) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter}
}

type hostFocus int

const (
	hostInput hostFocus = iota
	loginButton
	registerButton
)

type Host struct {
	keys hostKeys

	width     int
	height    int
	formWidth int

	input textinput.Model
	err   string
	focus hostFocus
	help  help.Model
}

func NewHost(width, height int) tea.Model {
	m := Host{
		keys: hostKeys{
			Up: key.NewBinding(
				key.WithKeys("up", "shift+tab"),
				key.WithHelp("↑/shift+tab", "move up"),
			),
			Down: key.NewBinding(
				key.WithKeys("down", "tab"),
				key.WithHelp("↓/tab", "move down"),
			),
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "confirm"),
			),
		},

		input: textinput.New(),
		focus: hostInput,
		help:  help.New(),
	}

	m.setSize(width, height)

	m.input.Prompt = "> "
	m.input.PromptStyle = m.input.PlaceholderStyle.Copy()
	m.input.Placeholder = "server host"
	m.input.Focus()

	return m
}

func (m Host) Init() tea.Cmd {
	return textinput.Blink
}

func (m Host) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if m.focus == hostInput {
			m.err = "" // reset validation message on user input
		}

		switch {
		case key.Matches(msg, m.keys.Up):
			cmd := m.focusUp()
			return m, cmd

		case m.focus == hostInput &&
			(key.Matches(msg, m.keys.Down) || key.Matches(msg, m.keys.Enter)):
			hostIsValid := hostRegexp.MatchString(m.input.Value())
			if !hostIsValid {
				m.err = "It doesn't look like a valid hostname!"
				return m, nil
			}

			cmd := m.focusDown()
			return m, cmd

		case key.Matches(msg, m.keys.Enter):
			switch m.focus {
			case loginButton:
				return m, func() tea.Msg {
					return HostSetMsg{m.input.Value(), LoginAction}
				}
			case registerButton:
				return m, func() tea.Msg {
					return HostSetMsg{m.input.Value(), RegisterAction}
				}
			}

		case key.Matches(msg, m.keys.Down):
			cmd := m.focusDown()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Host) View() string {
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
				infoView(m.err, m.formWidth, 4, m.err != ""),
				buttonView("Login", m.formWidth, m.focus == loginButton),
				buttonView("Register", m.formWidth, m.focus == registerButton),
			),
		),
		m.help.ShortHelpView(m.keys.Bindings()),
	)
}

func (m *Host) setSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width

	if width < maxFormWidth {
		m.formWidth = width
	} else {
		m.formWidth = maxFormWidth
	}

	m.input.Width = m.formWidth - 3
}

func (m *Host) focusUp() tea.Cmd {
	if m.focus == hostInput {
		return nil
	}

	m.focus -= 1
	return m.updateFocus()
}

func (m *Host) focusDown() tea.Cmd {
	if m.focus == registerButton {
		return nil
	}

	m.focus += 1
	return m.updateFocus()
}

func (m *Host) updateFocus() tea.Cmd {
	if m.focus == hostInput && !m.input.Focused() {
		return m.input.Focus()
	} else if m.focus != hostInput && m.input.Focused() {
		m.input.Blur()
	}

	return nil
}

var hostRegexp = regexp.MustCompile(`^(?i)[a-z0-9-]+(\.[a-z0-9-]+)*(:[0-9]+)?$`)

type HostSetAction int

const (
	LoginAction HostSetAction = iota
	RegisterAction
)

type HostSetMsg struct {
	Host   string
	Action HostSetAction
}
