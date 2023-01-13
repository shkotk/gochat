package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/shkotk/gochat/client/apiclient"
	"github.com/shkotk/gochat/client/models"
	"github.com/shkotk/gochat/common/validation"
)

type Model struct {
	width  int
	height int

	subModel tea.Model

	apiClient  *apiclient.ApiClient
	validate   *validator.Validate
	translator ut.Translator
}

func initialModel() Model {
	m := Model{
		subModel: models.NewHost(30, 30), // size will be updated with initial WindowSizeMsg
		validate: validator.New(),
	}

	// TODO move validator configuration to common and use in server
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(m.validate, trans)
	m.translator = trans
	m.validate.RegisterTranslation("name", trans, func(ut ut.Translator) error {
		return ut.Add("name", "{0} should consist of alphanumeric characters with optional separators [._-]", false)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("name", fe.Field())
		return t
	})

	m.validate.SetTagName("binding") // use same tag as gin does to avoid duplicating rules
	m.validate.RegisterValidation("name", validation.IsValidName)

	return m
}

func (m Model) Init() tea.Cmd {
	return m.subModel.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case models.HostSetMsg:
		m.apiClient = apiclient.New(msg.Host)
		switch msg.Action {
		case models.LoginAction:
			m.subModel = models.NewLogin(
				m.width, m.height, m.apiClient, m.validate, m.translator)
		case models.RegisterAction:
			// TODO
		}
		return m, m.subModel.Init()

	case models.LoginMsg:
		m.subModel = models.NewHub(m.width, m.height, m.apiClient)
		return m, tea.Batch(
			m.subModel.Init(),
			refreshTokenCmd(msg.TokenExpiresAt, m.apiClient),
		)

	case tokenRefreshedMsg:
		return m, refreshTokenCmd(msg.ExpiresAt, m.apiClient)

	case models.BackToHubMsg:
		m.subModel = models.NewHub(m.width, m.height, m.apiClient)
		return m, m.subModel.Init()

	case models.ChatJoinedMsg:
		m.subModel = models.NewChat(m.width, m.height, m.apiClient)
		return m, m.subModel.Init()
	}

	var cmd tea.Cmd
	m.subModel, cmd = m.subModel.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.subModel.View()
}

func main() {
	_, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run()
	if err != nil {
		panic(err)
	}
}

func refreshTokenCmd(expiresAt time.Time, client *apiclient.ApiClient) tea.Cmd {
	return func() tea.Msg {
		<-time.After(time.Until(expiresAt) - 30*time.Second)

		newExpiresAt, err := client.RefreshToken()
		if err != nil {
			return models.ErrorMsg(err.Error())
		}

		return tokenRefreshedMsg{newExpiresAt}
	}
}

type tokenRefreshedMsg struct {
	ExpiresAt time.Time
}
