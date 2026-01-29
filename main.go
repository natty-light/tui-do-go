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
	"strings"

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
	lists           map[string][]listItem
	keys            []string
	selectedList    int
	cursor          int
	textInput       textinput.Model
	listStartOffset int
}

type screenItemKind int

const (
	kindKey screenItemKind = iota
	kindItem
	kindInput
)

type screenItem struct {
	kind      screenItemKind
	keyIndex  int // for keys row
	itemIndex int // for list items
}

func (m model) screenItems() []screenItem {
	items := make([]screenItem, 0)
	for i := range m.keys {
		items = append(items, screenItem{kind: kindKey, keyIndex: i})
	}

	if len(m.keys) > 0 {
		lst := m.lists[m.keys[m.selectedList]]
		for i := range lst {
			items = append(items, screenItem{kind: kindItem, itemIndex: i})
		}
	}

	items = append(items, screenItem{kind: kindInput})
	return items
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

	listStartOffset := len(keys)

	cursor := 0
	if len(keys) > 0 {
		cursor = len(lists[keys[selectedList]]) + listStartOffset
	}

	m := model{
		lists:           lists,
		selectedList:    selectedList,
		keys:            keys,
		cursor:          cursor,
		textInput:       ti,
		listStartOffset: listStartOffset,
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

	total := numKeys + numItems + 1
	itemCursor := m.getItemCursor()

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if msg.String() == "q" && itemCursor == numItems && numKeys > 0 {
				break
			}
			m.SaveItems()
			return m, tea.Quit
		case "d":
			// Only delete when cursor is on a valid item (not keys or input)
			if itemCursor < 0 || itemCursor >= numItems {
				break
			}
			newItems := make([]listItem, 0, len(items)-1)
			for i, item := range items {
				if i != itemCursor {
					newItems = append(newItems, item)
				}
			}
			m.setCursor(len(newItems) - 1)
			items = newItems
			m.lists[m.keys[m.selectedList]] = newItems

		case "tab", "shift+tab", "up", "down":
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.cursor--
			} else {
				m.cursor++
			}

			// Wrap across all on-screen elements: keys + items + input
			if m.cursor >= total {
				m.cursor = 0
			} else if m.cursor < 0 {
				m.cursor = total - 1
			}

			// Focus input when cursor is on it (last index)
			if m.cursor == total-1 {
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
			// If cursor is on a key, select that list
			if m.cursor < numKeys {
				m.selectedList = m.cursor
				break
			}
			// If on an item, toggle completion
			if itemCursor >= 0 && itemCursor < numItems {
				completed := items[itemCursor].Completed
				m.lists[m.keys[m.selectedList]][itemCursor].Completed = !completed
				break
			}
			// If on input, add a new item
			if itemCursor == numItems {
				text := strings.TrimSpace(m.textInput.Value())
				if text == "" {
					break
				}
				m.lists[m.keys[m.selectedList]] = append(items, listItem{
					Item:      text,
					Completed: false,
				})
				m.cursor++
				m.textInput.SetValue("")
				return m, textinput.Blink
			}
		case " ":
			// Only toggle when cursor is on a valid item
			if itemCursor >= 0 && itemCursor < numItems {
				completed := items[itemCursor].Completed
				m.lists[m.keys[m.selectedList]][itemCursor].Completed = !completed
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := "Your Tui-Dos\n\n"

	// Build a unified view by iterating over all screen items
	items := m.screenItems()
	// First line: keys row (rendered in place as we iterate)
	for i, si := range items {
		switch si.kind {
		case kindKey:
			key := m.keys[si.keyIndex]
			if m.cursor == i {
				s += selectedStyle.Render(key) + "\t"
			} else if m.selectedList == si.keyIndex {
				s += currentListStyle.Render(key) + "\t"
			} else {
				s += fmt.Sprintf("%s\t", key)
			}
			// Peek next kind to know when keys end to add spacing
			if i+1 < len(items) && items[i+1].kind != kindKey {
				s += "\n\n"
			}
		case kindItem:
			item := m.lists[m.keys[m.selectedList]][si.itemIndex]
			style := noStyle
			c := " "
			if m.cursor == i {
				c = ">"
				style = selectedStyle
			}
			checked := " "
			if item.Completed {
				checked = "x"
			}
			s += style.Render(fmt.Sprintf("%s [%s] %s", c, checked, item.Item)) + "\n"
		case kindInput:
			// After items, show input
			s += "\n" + m.textInput.View()
		}
	}

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

func (m model) setCursor(new int) {
	m.cursor = new + m.listStartOffset
}

func (m model) getItemCursor() int {
	return m.cursor - m.listStartOffset
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
