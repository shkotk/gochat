package models

import (
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/shkotk/gochat/client/apiclient"
	"github.com/shkotk/gochat/common/apimodels/requests"
)

type loginKeys struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

func (k loginKeys) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter}
}

type loginFocus int

const (
	usernameInput loginFocus = iota
	passwordInput
)

type Login struct {
	keys loginKeys

	width     int
	height    int
	formWidth int

	inputs []textinput.Model
	focus  loginFocus
	err    string
	help   help.Model

	client     *apiclient.ApiClient
	validate   *validator.Validate
	translator ut.Translator
}

func NewLogin(
	width, height int,
	client *apiclient.ApiClient,
	validate *validator.Validate,
	translator ut.Translator,
) tea.Model {
	m := Login{
		keys: loginKeys{
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

		inputs: make([]textinput.Model, 2),

		focus: usernameInput,
		help:  help.New(),

		client:     client,
		validate:   validate,
		translator: translator,
	}

	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].Prompt = "> "
		m.inputs[i].PromptStyle = m.inputs[i].PlaceholderStyle.Copy()
	}

	m.inputs[usernameInput].Placeholder = "username"
	m.inputs[usernameInput].Focus()

	m.inputs[passwordInput].Placeholder = "password"
	m.inputs[passwordInput].EchoMode = textinput.EchoPassword
	m.inputs[passwordInput].EchoCharacter = '•'

	m.setSize(width, height)

	return m
}

func (m Login) Init() tea.Cmd {
	return textinput.Blink
}

func (m Login) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		// validate focused textinput on user navigation actions
		if key.Matches(msg, m.keys.Up, m.keys.Down, m.keys.Enter) {
			m.validateFocused()
		}

		// don't let user skip textinput or make login request
		// until we have a valid value for current field
		if m.err != "" && key.Matches(msg, m.keys.Down, m.keys.Enter) {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Up):
			cmd := m.focusUp()
			return m, cmd

		case key.Matches(msg, m.keys.Down):
			cmd := m.focusDown()
			return m, cmd

		case key.Matches(msg, m.keys.Enter):
			if m.focus == usernameInput {
				cmd := m.focusDown()
				return m, cmd
			}

			// TODO run spinner or smthng
			return m, func() tea.Msg {
				expiresAt, err := m.client.Login(requests.Auth{
					Username: m.inputs[usernameInput].Value(),
					Password: m.inputs[passwordInput].Value(),
				})
				if err != nil {
					return ErrorMsg(err.Error())
				}

				return LoginMsg{expiresAt}
			}
		}

		// reset error if key press doesn't match any navigation binding
		m.err = ""

	case ErrorMsg:
		m.err = string(msg)
		return m, nil
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m Login) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.Place(
			m.width,
			m.height-1, // one line for help
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.JoinVertical(
				lipgloss.Left,
				itemStyle.Render(m.inputs[usernameInput].View()),
				itemStyle.Render(m.inputs[passwordInput].View()),
				infoView(m.err, m.formWidth, 3, m.err != ""),
			),
		),
		m.help.ShortHelpView(m.keys.Bindings()),
	)
}

func (m *Login) setSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width

	if width < maxFormWidth {
		m.formWidth = width
	} else {
		m.formWidth = maxFormWidth
	}

	for _, input := range m.inputs {
		input.Width = m.formWidth - 3
	}
}

func (m *Login) focusUp() tea.Cmd {
	if m.focus == usernameInput {
		return nil
	}

	m.focus -= 1
	return m.updateFocus()
}

func (m *Login) focusDown() tea.Cmd {
	if m.focus == passwordInput {
		return nil
	}

	m.focus += 1
	return m.updateFocus()
}

func (m *Login) updateFocus() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.inputs {
		if int(m.focus) == i && !m.inputs[i].Focused() {
			cmd = m.inputs[i].Focus()
		} else if int(m.focus) != i && m.inputs[i].Focused() {
			m.inputs[i].Blur()
		}
	}

	return cmd
}

// Validates focused input and populates m.err with the resulting error message
func (m *Login) validateFocused() {
	err := m.validate.Struct(requests.Auth{
		Username: m.inputs[usernameInput].Value(),
		Password: m.inputs[passwordInput].Value(),
	})
	if err == nil {
		m.err = ""
		return
	}

	var fieldName string
	switch m.focus {
	case usernameInput:
		fieldName = "Username"
	case passwordInput:
		fieldName = "Password"
	default:
		log.Panicf("unexpected focus %v", m.focus)
	}

	builder := strings.Builder{}
	for _, err := range err.(validator.ValidationErrors) {
		if err.StructField() == fieldName {
			if builder.Len() > 0 {
				builder.WriteString("; ")
			}
			builder.WriteString(err.Translate(m.translator))
		}
	}

	m.err = builder.String()
}

type LoginMsg struct {
	TokenExpiresAt time.Time
}
