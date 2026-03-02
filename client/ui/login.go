package ui

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/XplnHUB/Les-Go/client/crypto"
	"github.com/XplnHUB/Les-Go/client/network"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LoginModel struct {
	client *network.APIClient
	inputs []textinput.Model
	focus  int
	err    error
	mode   string // "login" or "register"
}

func InitialLoginModel(client *network.APIClient) LoginModel {
	m := LoginModel{
		client: client,
		inputs: make([]textinput.Model, 2),
		mode:   "login",
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = "Username"
			t.Focus()
			t.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			t.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		case 1:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		m.inputs[i] = t
	}

	return m
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.focus--
			} else {
				m.focus++
			}

			if m.focus > len(m.inputs)-1 {
				m.focus = 0
			} else if m.focus < 0 {
				m.focus = len(m.inputs) - 1
			}

			cmds = append(cmds, m.updateFocus())

		case "ctrl+r":
			if m.mode == "login" {
				m.mode = "register"
			} else {
				m.mode = "login"
			}
			m.err = nil

		case "enter":
			username := m.inputs[0].Value()
			password := m.inputs[1].Value()

			if username == "" || password == "" {
				m.err = fmt.Errorf("Please enter username and password")
				return m, nil
			}

			// Perform action
			var keys *crypto.KeyPair
			if m.mode == "register" {
				// Generate keys
				keys, _ = crypto.GenerateKeyPair()
				pubKeyStr := base64.StdEncoding.EncodeToString(keys.PublicKey[:])

				if err := m.client.Register(username, password, pubKeyStr); err != nil {
					m.err = err
					return m, nil
				}
				// Save keys to disk
				if err := crypto.SaveKeys(username, password, keys); err != nil {
					m.err = fmt.Errorf("Registered, but failed to save keys: %v", err)
					return m, nil
				}
				m.mode = "login"
				m.err = fmt.Errorf("Registered and keys saved! Press Enter to Login")
				return m, nil
			} else {
				// Login
				if err := m.client.Login(username, password); err != nil {
					m.err = err
					return m, nil
				}

				// Load keys from disk
				keys, err := crypto.LoadKeys(username, password)
				if err != nil {
					m.err = fmt.Errorf("Login successful, but %v", err)
					return m, nil
				}

				// Generate transient keypair or let model hold user identity.
				// In a real app we'd load the private key from disk using a passphrase.
				// For this local demo, we just auto-generate an ephemeral key if we log in without registering.
				// NOTE: If they register then login, they lose the key if we don't save it!
				// For simplicity of this task, we will just generate an ephemeral one and register automatically
				// if logging in fails, or we assume they just Registered.

				// To solve this properly, let's connect WS immediately after login
				if err := m.client.ConnectWebSocket(); err != nil {
					m.err = err
					return m, nil
				}

				// Transition! We need to pass username and keys to Chat state.
				return m, func() tea.Msg {
					return StateChangeMsg{NewState: StateChat, Username: username, Keys: keys}
				}
			}
		}
	}

	// Handle character inputs
	cmd := m.updateInputs(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *LoginModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := 0; i < len(m.inputs); i++ {
		if i == m.focus {
			cmds = append(cmds, m.inputs[i].Focus())
			m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			m.inputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			continue
		}
		m.inputs[i].Blur()
		m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		m.inputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	}
	return tea.Batch(cmds...)
}

func (m *LoginModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		m.inputs[i], _ = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m LoginModel) View() string {
	b := strings.Builder{}

	title := "🔐 Login to Les'Go"
	if m.mode == "register" {
		title = "📝 Register for Les'Go"
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).PaddingBottom(1).Render(title) + "\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View() + "\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n[Tab] Next field • [Enter] Submit • [Ctrl+R] Toggle Login/Register"))

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("\n\nError: %v", m.err)))
	}

	style := lipgloss.NewStyle().
		Margin(1, 2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return style.Render(b.String())
}
