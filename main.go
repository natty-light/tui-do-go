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
	focusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	noStyle       = lipgloss.NewStyle()
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	buttonStyle   = lipgloss.NewStyle().Background(lipgloss.Color("15")).Foreground(lipgloss.Color("0"))
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
	lists            map[string][]listItem
	keys             []string
	selectedList     int
	cursor           int
	textInput        textinput.Model
	newListTextInput textinput.Model
	listStartOffset  int
}

type screenItemKind int

const (
	kindKey screenItemKind = iota
	kindItem
	kindNewItem
	kindNewList
)

type screenItem struct {
	kind    screenItemKind
	keyIdx  int // for keys row
	itemIdx int // for list items
}

func (m *model) screenItems() []screenItem {
	items := make([]screenItem, 0)
	for i := range m.keys {
		items = append(items, screenItem{kind: kindKey, keyIdx: i})
	}

	if len(m.keys) > 0 {
		lst := m.lists[m.keys[m.selectedList]]
		for i := range lst {
			items = append(items, screenItem{kind: kindItem, itemIdx: i})
		}
	}

	if len(m.keys) > 0 {
		items = append(items, screenItem{kind: kindNewItem})
	}
	items = append(items, screenItem{kind: kindNewList})
	return items
}

func initialModel() *model {
	lists, keys := loadItems()
	selectedList := 0

	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 20

	newListTi := textinput.New()
	newListTi.CharLimit = 156
	newListTi.Width = 20

	listStartOffset := len(keys)

	cursor := 0
	if len(keys) > 0 {
		cursor = len(lists[keys[selectedList]]) + listStartOffset
	}

	m := model{
		lists:            lists,
		selectedList:     selectedList,
		keys:             keys,
		cursor:           cursor,
		textInput:        ti,
		listStartOffset:  listStartOffset,
		newListTextInput: newListTi,
	}

	// Set initial focus/styles based on whether we have any lists
	if len(keys) > 0 {
		// Focus new-item input
		m.focusTextInput()
		m.blurListTextInput()
	} else {
		// No lists yet: focus new-list input
		m.focusListTextInput()
		m.blurTextInput()
	}

	return &m
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var newListCmd tea.Cmd

	numKeys := len(m.keys)

	items := make([]listItem, 0)
	if numKeys > 0 {
		items = m.lists[m.keys[m.selectedList]]
	}
	numItems := len(items)

	inputsCount := 1
	if numKeys > 0 {
		inputsCount = 2
	}

	total := numKeys + numItems + inputsCount
	newListIdx := total - 1
	newItemIdx := -1
	if inputsCount == 2 {
		newItemIdx = total - 2
	}

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
			// Deleting a list key or an item depending on cursor position
			if m.cursor < numKeys {
				// Delete the key at cursor (including if it's the selected list)
				deleteKeyIdx := m.cursor
				deleteKey := m.keys[deleteKeyIdx]

				// Build new keys slice without the deleted key
				newKeys := make([]string, 0, numKeys-1)
				for i, key := range m.keys {
					if i != deleteKeyIdx {
						newKeys = append(newKeys, key)
					}
				}
				// Remove the list from the map to avoid orphans
				delete(m.lists, deleteKey)

				// Update selection based on which key was deleted
				if len(newKeys) == 0 {
					m.keys = newKeys
					m.selectedList = 0
					m.listStartOffset = 0
					m.cursor = 0
					// Focus new-list input when there are no lists
					m.blurTextInput()
					newListCmd = m.focusListTextInput()
					return m, tea.Batch(cmd, newListCmd, textinput.Blink)
				}

				// There are still lists
				// Adjust selectedList considering index shift
				if deleteKeyIdx == m.selectedList {
					// Move to previous list (or stay at 0 if first was deleted)
					m.selectedList = deleteKeyIdx - 1
					if m.selectedList < 0 {
						m.selectedList = 0
					}
				} else if deleteKeyIdx < m.selectedList {
					// Shift left because indices after deletion move
					m.selectedList--
				}

				m.keys = newKeys
				numKeys = len(newKeys)
				m.listStartOffset = numKeys
				// Keep cursor on the (new) selected key
				m.cursor = m.selectedList
				// Blur inputs when on keys
				m.blurTextInput()
				m.blurListTextInput()
				return m, cmd
			}

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

			if newItemIdx >= 0 && m.cursor == newItemIdx {
				cmd = tea.Batch(m.focusTextInput(), textinput.Blink)
				m.blurListTextInput()
			} else if m.cursor == newListIdx {
				cmd = tea.Batch(m.focusListTextInput(), textinput.Blink)
				m.blurTextInput()
			} else {
				m.blurListTextInput()
				m.blurTextInput()
			}

			// Wrap across all on-screen elements: keys + items + input
			if m.cursor >= total {
				m.cursor = 0
			} else if m.cursor < 0 {
				m.cursor = total - 1
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
			// Add a new item only when lists exist and cursor is at the new-item input
			if newItemIdx >= 0 && m.cursor == newItemIdx {
				text := strings.TrimSpace(m.textInput.Value())
				if text == "" {
					break
				}
				m.lists[m.keys[m.selectedList]] = append(items, listItem{
					Item:      text,
					Completed: false,
				})
				// Stay on the new-item input for rapid entry
				m.cursor++
				m.textInput.SetValue("")
				return m, textinput.Blink
			}

			// Add a new list when cursor is at the new-list input
			if m.cursor == newListIdx {
				text := strings.TrimSpace(m.newListTextInput.Value())
				if text == "" {
					break
				}
				m.keys = append(m.keys, text)
				m.lists[text] = make([]listItem, 0)
				// Select the newly created list
				m.selectedList = len(m.keys) - 1
				m.listStartOffset = len(m.keys)
				// Place cursor onto the new-item input for the new list
				m.cursor = m.listStartOffset // equals newItemIdx after recompute
				m.newListTextInput.SetValue("")
				// Focus swap: new-item gets focus now
				cmd = m.textInput.Focus()
				m.textInput.PromptStyle = focusedStyle
				m.textInput.TextStyle = focusedStyle
				m.newListTextInput.Blur()
				m.newListTextInput.PromptStyle = noStyle
				m.newListTextInput.TextStyle = noStyle
				return m, tea.Batch(cmd, textinput.Blink)
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
	m.newListTextInput, newListCmd = m.newListTextInput.Update(msg)
	return m, tea.Batch(cmd, newListCmd)
}

func (m *model) View() string {
	s := "Your Tui-Dos\n\n"

	items := m.screenItems()

	s += fmt.Sprintf("items: %d  cursor: %d\n\n", len(items), m.cursor)
	for i, si := range items {
		switch si.kind {
		case kindKey:
			key := m.keys[si.keyIdx]
			if m.cursor == i {
				s += selectedStyle.Render(key) + "\t"
			} else if m.selectedList == si.keyIdx {
				s += buttonStyle.Render(key) + "\t"
			} else {
				s += fmt.Sprintf("%s\t", key)
			}
			if i+1 < len(items) && items[i+1].kind != kindKey {
				s += "\n\n"
			}
		case kindItem:
			item := m.lists[m.keys[m.selectedList]][si.itemIdx]
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
		case kindNewItem:
			s += "\n\n" + m.textInput.View()
		case kindNewList:
			s += "\n\nAdd a new list: " + m.newListTextInput.View()
		}
	}

	s += footer

	return s
}

func (m *model) SaveItems() {
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

func (m *model) setCursor(new int) {
	m.cursor = new + m.listStartOffset
}

func (m *model) getItemCursor() int {
	return m.cursor - m.listStartOffset
}

func (m *model) blurTextInput() {
	m.textInput.Blur()
	m.textInput.TextStyle = noStyle
	m.textInput.PromptStyle = noStyle
}

func (m *model) blurListTextInput() {
	m.newListTextInput.Blur()
	m.newListTextInput.TextStyle = noStyle
	m.newListTextInput.PromptStyle = noStyle
}

func (m *model) focusTextInput() tea.Cmd {
	m.textInput.PromptStyle = focusedStyle
	m.textInput.TextStyle = focusedStyle
	return m.textInput.Focus()
}

func (m *model) focusListTextInput() tea.Cmd {
	m.newListTextInput.TextStyle = focusedStyle
	m.newListTextInput.PromptStyle = focusedStyle
	return m.newListTextInput.Focus()
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
