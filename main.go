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

	footer = "\nPress q or ctrl+c to quit.\n"
)

var (
	focusedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	noStyle          = lipgloss.NewStyle()
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	currentListStyle = lipgloss.NewStyle().Background(lipgloss.Color("202"))
)

type listItem struct {
	Item      string `json:"item"`
	Completed bool   `json:"completed"`
}

type jsonData struct {
	Keys  []string              `json:"keys"`
	Lists map[string][]listItem `json:"lists"`
}

type model struct {
	lists        map[string][]listItem
	keys         []string
	selectedList int
	cursor       int
	textInput    textinput.Model
}

func initialModel() model {
	lists, keys := loadItems()
	selectedList := 0

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	cursor := 0
	if len(keys) > 0 {
		cursor = len(lists[keys[selectedList]])
	}

	m := model{
		lists:        lists,
		selectedList: selectedList,
		keys:         keys,
		cursor:       cursor,
		textInput:    ti,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	numKeys := len(m.keys)

	items := make([]listItem, 0)
	if numKeys > 0 {
		items = m.lists[m.keys[m.selectedList]]
	}
	numItems := len(items)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if msg.String() == "q" && m.cursor == numItems && numKeys > 0 {
				break
			}
			m.SaveItems()
			return m, tea.Quit
		case "d":
			if m.cursor == numItems {
				break
			}
			newItems := make([]listItem, 0)
			for i, item := range items {
				if i != m.cursor {
					newItems = append(newItems, item)
				}
			}
			m.cursor = len(newItems) - 1
			items = newItems

		case "tab", "shift+tab", "up", "down":
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.cursor--
			} else {
				m.cursor++
			}

			if m.cursor > len(items) {
				m.cursor = 0
			} else if m.cursor < 0 {
				m.cursor = len(items)
			}

			if numItems == m.cursor {
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
			if m.cursor != numItems {
				completed := items[m.cursor].Completed
				if completed {
					m.lists[m.keys[m.selectedList]][m.cursor].Completed = false
				} else {
					m.lists[m.keys[m.selectedList]][m.cursor].Completed = true
				}
			} else {
				text := m.textInput.Value()
				m.lists[m.keys[m.selectedList]] = append(items, listItem{
					Item:      text,
					Completed: false,
				})

				m.cursor++
				m.textInput.SetValue("")

				return m, textinput.Blink
			}
		case " ":
			if m.cursor != numItems {
				completed := items[m.cursor].Completed
				if completed {
					m.lists[m.keys[m.selectedList]][m.cursor].Completed = false
				} else {
					m.lists[m.keys[m.selectedList]][m.cursor].Completed = true
				}
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := "Your Tui-Dos\n\n"

	if len(m.keys) == 0 {
		s += "add a new list\n\n"
		s += footer
		return s
	}

	for idx, _ := range m.keys {
		key := m.keys[idx]
		if m.selectedList == idx {
			s += currentListStyle.Render(key) + "\t"
		} else {
			s += fmt.Sprintf("%s\t", key)
		}
	}
	s += "\n\n"

	// Iterate over our choices
	for i, item := range m.lists[m.keys[m.selectedList]] {
		style := noStyle

		// Is the cursor pointing at this item?
		c := " " // no cursor
		if m.cursor == i {
			c = ">" // cursor!
			style = selectedStyle
		}

		// Is this item selected?
		checked := " " // not selected
		if item.Completed {
			checked = "x" // selected!
		}

		// Render the row
		s += style.Render(fmt.Sprintf("%s [%s] %s", c, checked, item.Item))
		s += "\n"
	}

	s += "\n" + m.textInput.View()

	// The footer
	s += footer

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

	data := jsonData{
		Keys:  m.keys,
		Lists: m.lists,
	}
	asStr, err := json.Marshal(data)
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

func loadItems() (map[string][]listItem, []string) {
	data := &jsonData{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	filePath := filepath.Join(homeDir, dirName, fileName)
	// Read the whole file at once; the file contains a single JSON array
	contents, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No file yet: start with an empty list
			return make(map[string][]listItem), []string{}
		}
		log.Fatal(err)
	}

	// Allow empty files to be treated as empty lists
	if len(contents) == 0 {
		return make(map[string][]listItem), []string{}
	}

	if err := json.Unmarshal(contents, &data); err != nil {
		log.Fatal(err)
	}
	lists := data.Lists
	keys := data.Keys

	return lists, keys
}
