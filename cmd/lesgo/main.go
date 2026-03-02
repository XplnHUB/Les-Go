package main

import (
	"fmt"
	"log"
	"os"

	"github.com/XplnHUB/Les-Go/client/network"
	"github.com/XplnHUB/Les-Go/client/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	apiClient := network.NewAPIClient("http://localhost:8080")
	m := ui.InitialModel(apiClient)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		log.Fatal(err)
	}
}
