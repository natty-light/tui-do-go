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

	tea "github.com/charmbracelet/bubbletea"
)

const (
	dirName  = ".tui-do"
	fileName = ".tui-do.json"
)

type listItem struct {
	Item      string `json:"item"`
	Completed bool   `json:"completed"`
}

type model struct {
	items    []listItem
	cursor   int
	selected map[int]struct{}
}

func initialModel() model {
	m := model{
		items:    make([]listItem, 0),
		cursor:   0,
		selected: make(map[int]struct{}),
	}
	m.LoadItems()

	return m
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			m.SaveItems()
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	s := "Your Tui-Dos\n\n"

	// Iterate over our choices
	for i, item := range m.items {

		// Is the cursor pointing at this item?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this item selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, item.Item)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
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

func (m model) LoadItems() {
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
			m.items = items
			return
		}
		log.Fatal(err)
	}

	// Allow empty files to be treated as empty lists
	if len(data) == 0 {
		m.items = items
		return
	}

	if err := json.Unmarshal(data, &items); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d items\n", len(items))
	m.items = items
}
