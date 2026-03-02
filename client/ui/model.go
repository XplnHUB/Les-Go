package ui

import (
	"github.com/XplnHUB/Les-Go/client/crypto"
	"github.com/XplnHUB/Les-Go/client/network"
	tea "github.com/charmbracelet/bubbletea"
)

type SessionState int

const (
	StateLogin SessionState = iota
	StateChat
)

type Model struct {
	state SessionState

	// Sub-models
	loginModel tea.Model
	chatModel  tea.Model

	// Global dimensions
	width  int
	height int

	client *network.APIClient
}

func InitialModel(client *network.APIClient) Model {
	return Model{
		state:      StateLogin,
		loginModel: InitialLoginModel(client),
		chatModel:  InitialChatModel(client),
		client:     client,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loginModel.Init(), m.chatModel.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Propagate resize to sub-models
		var loginModel, chatModel tea.Model
		loginModel, cmd = m.loginModel.Update(msg)
		m.loginModel = loginModel
		cmds = append(cmds, cmd)

		chatModel, cmd = m.chatModel.Update(msg)
		m.chatModel = chatModel
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case StateChangeMsg:
		m.state = msg.NewState
		var cmd tea.Cmd
		if msg.NewState == StateChat {
			cm, ok := m.chatModel.(ChatModel)
			if ok {
				m.chatModel = cm.SetUsernameAndKeys(msg.Username, msg.Keys)
				cmd = cm.listenForMessages()
			}
		}
		return m, cmd
	}

	// Route updating to the active sub-model
	switch m.state {
	case StateLogin:
		m.loginModel, cmd = m.loginModel.Update(msg)
	case StateChat:
		m.chatModel, cmd = m.chatModel.Update(msg)
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.state {
	case StateLogin:
		return m.loginModel.View()
	case StateChat:
		return m.chatModel.View()
	default:
		return "Unknown state"
	}
}

// StateChangeMsg is used to trigger a view change
type StateChangeMsg struct {
	NewState SessionState
	Username string
	Keys     *crypto.KeyPair
}
