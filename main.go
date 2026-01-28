package main

// These imports will be used later on the tutorial. If you save the file
// now, Go might complain they are unused, but that's fine.
// You may also need to run `go mod tidy` to download bubbletea and its
// dependencies.
import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	dirName  = ".tui-do"
	fileName = ".tui-do.json"
)

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type listItem struct {
	Item      string `json:"item"`
	Completed bool   `json:"completed"`
}

type model struct {
	items     []listItem
	cursor    int
	textInput textinput.Model
}

func initialModel() model {
	items := loadItems()

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	m := model{
		items:     items,
		cursor:    len(items),
		textInput: ti,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if msg.String() == "q" && m.cursor == len(m.items) {
				break
			}
			m.SaveItems()
			return m, tea.Quit
		case "tab", "shift+tab", "up", "down":
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.cursor--
			} else {
				m.cursor++
			}

			if m.cursor > len(m.items) {
				m.cursor = 0
			} else if m.cursor < 0 {
				m.cursor = len(m.items)
			}

			if len(m.items) == m.cursor {
				cmd = m.textInput.Focus()
				m.textInput.PromptStyle = focusedStyle
				m.textInput.TextStyle = focusedStyle

			} else {
				m.textInput.Blur()
				m.textInput.PromptStyle = noStyle
				m.textInput.TextStyle = noStyle
			}

			return m, cmd

		case "enter":
			if m.cursor != len(m.items) {
				completed := m.items[m.cursor].Completed
				if completed {
					m.items[m.cursor].Completed = false
				} else {
					m.items[m.cursor].Completed = true
				}
			} else {
				text := m.textInput.Value()
				m.items = append(m.items, listItem{
					Item:      text,
					Completed: false,
				})

				m.cursor++
				m.textInput.SetValue("")

				return m, textinput.Blink
			}
		case " ":
			if m.cursor != len(m.items) {
				completed := m.items[m.cursor].Completed
				if completed {
					m.items[m.cursor].Completed = false
				} else {
					m.items[m.cursor].Completed = true
				}
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := "Your Tui-Dos\n\n"

	// Iterate over our choices
	for i, item := range m.items {

		// Is the cursor pointing at this item?
		c := " " // no cursor
		if m.cursor == i {
			c = ">" // cursor!
		}

		// Is this item selected?
		checked := " " // not selected
		if m.items[i].Completed {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", c, checked, item.Item)
	}

	s += "\n" + m.textInput.View()

	// The footer
	s += "\nPress q or ctrl+c to quit.\n"

	// Send the UI for rendering
	return s
}

func (m model) SaveItems() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	dataDir := filepath.Join(homeDir, dirName)
	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	// Marshal the full slice as a single JSON array
	asStr, err := json.Marshal(m.items)
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(dataDir, fileName), asStr, 0o644); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
	}
}

func (m model) ToggleItem() {

}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func loadItems() []listItem {
	items := make([]listItem, 0)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	filePath := filepath.Join(homeDir, dirName, fileName)
	// Read the whole file at once; the file contains a single JSON array
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No file yet: start with an empty list
			return items
		}
		log.Fatal(err)
	}

	// Allow empty files to be treated as empty lists
	if len(data) == 0 {
		return items
	}

	if err := json.Unmarshal(data, &items); err != nil {
		log.Fatal(err)
	}

	return items
}
