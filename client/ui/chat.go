package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/XplnHUB/Les-Go/client/crypto"
	"github.com/XplnHUB/Les-Go/client/network"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WSMsg struct {
	From string
	Data string
}

type ChatModel struct {
	client   *network.APIClient
	messages []string
	inputs   []textinput.Model
	focus    int
	err      error
	myUser   string
	keys     *crypto.KeyPair // In a real scenario, this would persist on disk
	online   map[string]bool // Sidebar data
}

func InitialChatModel(client *network.APIClient) ChatModel {
	m := ChatModel{
		client:   client,
		messages: []string{},
		inputs:   make([]textinput.Model, 2),
		online:   make(map[string]bool),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

		switch i {
		case 0:
			t.Placeholder = "To (Username)"
			t.Focus()
			t.CharLimit = 32
			t.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			t.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		case 1:
			t.Placeholder = "Message..."
			t.CharLimit = 256
		}
		m.inputs[i] = t
	}

	return m
}

func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.listenForMessages(),
	)
}

// listenForMessages reads from the WebSocket asynchronously
func (m ChatModel) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		if m.client.WSConn == nil {
			return nil // not connected
		}
		_, p, err := m.client.WSConn.ReadMessage()
		if err != nil {
			return nil // disconnected
		}

		var event struct {
			Type    string `json:"type"`
			Payload string `json:"payload"`
			From    string `json:"from"`
		}
		json.Unmarshal(p, &event)

		if event.Type == "CHAT_MESSAGE" {
			return WSMsg{From: event.From, Data: event.Payload}
		}
		if event.Type == "USER_ONLINE" || event.Type == "USER_OFFLINE" {
			return PresenceMsg{Username: event.Payload, Online: event.Type == "USER_ONLINE"}
		}
		if event.Type == "ONLINE_USERS" {
			users := strings.Split(event.Payload, ",")
			return OnlineListMsg{Usernames: users}
		}
		return m.listenForMessages()() // Wait for next msg if not chat
	}
}

type OnlineListMsg struct {
	Usernames []string
}

type PresenceMsg struct {
	Username string
	Online   bool
}

func (m ChatModel) SetUsernameAndKeys(u string, k *crypto.KeyPair) ChatModel {
	m.myUser = u
	m.keys = k
	return m
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case WSMsg:
		// Attempt to decrypt
		var decodedMsg string
		senderPubKeyStr, err := m.client.GetPublicKey(msg.From)
		if err == nil {
			var senderPubKey [32]byte
			decoded, _ := base64.StdEncoding.DecodeString(senderPubKeyStr)
			copy(senderPubKey[:], decoded)

			decrypted, err := crypto.DecryptMessage(msg.Data, &senderPubKey, m.keys.PrivateKey)
			if err == nil {
				decodedMsg = fmt.Sprintf("[%s]: %s", msg.From, string(decrypted))
			} else {
				decodedMsg = fmt.Sprintf("[%s]: <encrypted>", msg.From)
			}
		} else {
			decodedMsg = fmt.Sprintf("[%s]: <could not fetch key>", msg.From)
		}

		m.messages = append(m.messages, decodedMsg)
		cmds = append(cmds, m.listenForMessages()) // Wait for next message

	case PresenceMsg:
		m.online[msg.Username] = msg.Online
		if !msg.Online {
			delete(m.online, msg.Username)
		}
		cmds = append(cmds, m.listenForMessages())

	case OnlineListMsg:
		for _, u := range msg.Usernames {
			if u != "" {
				m.online[u] = true
			}
		}
		cmds = append(cmds, m.listenForMessages())

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

		case "enter":
			if m.focus == 1 {
				to := m.inputs[0].Value()
				text := m.inputs[1].Value()

				if to != "" && text != "" {
					m.err = nil
					// Fetch recipient public key
					recipientKeyStr, err := m.client.GetPublicKey(to)
					if err != nil {
						m.err = fmt.Errorf("User %s not found", to)
						return m, nil
					}

					var recipientPubKey [32]byte
					decoded, _ := base64.StdEncoding.DecodeString(recipientKeyStr)
					copy(recipientPubKey[:], decoded)

					// Encrypt
					enc, err := crypto.EncryptMessage([]byte(text), &recipientPubKey, m.keys.PrivateKey)
					if err != nil {
						m.err = err
						return m, nil
					}

					// Send
					if err := m.client.SendMessage(to, enc); err != nil {
						m.err = err
					} else {
						m.messages = append(m.messages, fmt.Sprintf("[Me -> %s]: %s", to, text))
						m.inputs[1].SetValue("") // Clear input
					}
				}
			}
		}
	}

	cmd := m.updateInputs(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ChatModel) updateFocus() tea.Cmd {
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

func (m *ChatModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		m.inputs[i], _ = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m ChatModel) View() string {
	b := strings.Builder{}

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Render(fmt.Sprintf("Les'Go Chat Session [%s]", m.myUser)) + "\n\n")

	// 1. Sidebar
	sidebarStyle := lipgloss.NewStyle().
		Width(20).Height(10).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(1)

	var onlineList []string
	for user := range m.online {
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
		onlineList = append(onlineList, fmt.Sprintf("%s %s", status, user))
	}
	sidebarContent := lipgloss.NewStyle().Bold(true).Render("Online") + "\n" + strings.Join(onlineList, "\n")
	if len(onlineList) == 0 {
		sidebarContent += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true).Render("(none)")
	}

	// 2. Message history
	historyBox := lipgloss.NewStyle().
		Width(50).Height(10).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	historyStr := strings.Join(m.messages, "\n")
	if historyStr == "" {
		historyStr = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true).Render("No messages yet...")
	}

	// Join Sidebar and History
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, sidebarStyle.Render(sidebarContent), historyBox.Render(historyStr))
	b.WriteString(mainView + "\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View() + "\n")
	}

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("\nError: %v", m.err)))
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n[Tab] Next field • [Enter] inside Message to Send • [Ctrl+C] Quit"))

	style := lipgloss.NewStyle().
		Margin(1, 2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205"))

	return style.Render(b.String())
}
