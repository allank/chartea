package main

import (
	"log"
	"os"

	"play/clob"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// mainModel represents the state of our TUI application.
type mainModel struct {
	clob   clob.Model
	width  int
	height int
}

// InitialModel creates the initial state of the application model.
func InitialModel() mainModel {
	m := mainModel{
		clob: clob.New(),
	}
	m.clob.Asks = []clob.Order{
		{Price: 100, Volume: 5},
		{Price: 101, Volume: 10},
		{Price: 102, Volume: 20},
	}
	m.clob.Bids = []clob.Order{
		{Price: 99, Volume: 1},
		{Price: 98, Volume: 20},
		{Price: 97, Volume: 40},
	}

	// Override default styles
	m.clob.StyleOnBid = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("64"))
	m.clob.StyleOnAsk = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("164"))

	return m
}

// Init is the first command that is run when the program starts.
func (m mainModel) Init() tea.Cmd {
	return nil
}

// Update handles all incoming messages and updates the model accordingly.
func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.clob, cmd = m.clob.Update(msg)
	return m, cmd
}

// View renders the UI based on the current model state.
func (m mainModel) View() string {
	return m.clob.ViewWithOptions(clob.ViewOptions{Width: m.width, Height: m.height})
}

func main() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}