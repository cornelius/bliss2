package ui

import (
	"bliss/internal/list"
	"bliss/internal/store"
	"bliss/internal/todo"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DefaultKanbanOrder defines the default list order for grooming.
var DefaultKanbanOrder = []string{"inbox", "today", "this-week", "next-week", "later"}

// ListKeyMap maps keyboard shortcuts to list names.
var ListKeyMap = map[string]string{
	"i": "inbox",
	"t": "today",
	"w": "this-week",
	"n": "next-week",
	"l": "later",
}

// GroomItem represents a display item in the groom view (section header or todo).
type GroomItem struct {
	SectionIdx    int
	SectionHeader string // non-empty if this is a section header
	Todo          *todo.Todo
}

// GroomModel is the bubbletea model for the groom command.
type GroomModel struct {
	store       *store.Store
	contextUUID string

	// kanban list names in order
	listOrder []string
	// current list index in listOrder
	listIdx int

	// items for the current list
	items  []GroomItem
	cursor int

	// touched UUIDs (acted on in this session)
	touched map[string]bool

	// pending list key for two-key sequences (e.g. "t3" = today section 3)
	pendingListKey string

	dirty    bool
	quitting bool
	err      error
}

// NewGroomModel creates a new GroomModel.
func NewGroomModel(s *store.Store, contextUUID string, listOrder []string, startList string) GroomModel {
	if len(listOrder) == 0 {
		listOrder = DefaultKanbanOrder
	}

	startIdx := 0
	for i, name := range listOrder {
		if name == startList {
			startIdx = i
			break
		}
	}

	m := GroomModel{
		store:       s,
		contextUUID: contextUUID,
		listOrder:   listOrder,
		listIdx:     startIdx,
		touched:     make(map[string]bool),
	}
	return m
}

func (m GroomModel) Init() tea.Cmd {
	return loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)
}

type listLoadedMsg struct {
	items []GroomItem
}

func loadListCmd(s *store.Store, contextUUID, listName string, touched map[string]bool) tea.Cmd {
	return func() tea.Msg {
		items, err := buildGroomItems(s, contextUUID, listName, touched)
		if err != nil {
			return groomErrMsg{err}
		}
		return listLoadedMsg{items: items}
	}
}

type groomErrMsg struct{ err error }

func buildGroomItems(s *store.Store, contextUUID, listName string, touched map[string]bool) ([]GroomItem, error) {
	var items []GroomItem

	if listName == "inbox" {
		todos, err := s.ListTodos(contextUUID)
		if err != nil {
			return nil, err
		}
		listNames, err := s.ListNames(contextUUID)
		if err != nil {
			return nil, err
		}
		listedUUIDs := make(map[string]bool)
		for _, name := range listNames {
			l, err := s.ReadList(contextUUID, name)
			if err != nil {
				continue
			}
			for _, uuid := range list.AllUUIDs(l) {
				listedUUIDs[uuid] = true
			}
		}
		for i := range todos {
			t := &todos[i]
			if !listedUUIDs[t.UUID] && !touched[t.UUID] {
				tc := *t
				items = append(items, GroomItem{Todo: &tc})
			}
		}
		return items, nil
	}

	// Try context list first; fall back to personal list.
	// Personal lists may reference todos from any context, so use FindTodo.
	contextListNames, _ := s.ListNames(contextUUID)
	isContextList := false
	for _, name := range contextListNames {
		if name == listName {
			isContextList = true
			break
		}
	}

	var listContextUUID string
	if isContextList {
		listContextUUID = contextUUID
	}

	l, err := s.ReadList(listContextUUID, listName)
	if err != nil {
		return nil, err
	}

	for si, section := range l.Sections {
		if si > 0 || section.Name != "" {
			header := fmt.Sprintf("section %d", si+1)
			if section.Name != "" {
				header = section.Name
			}
			items = append(items, GroomItem{SectionIdx: si, SectionHeader: header})
		}
		for _, uuid := range section.Items {
			if touched[uuid] {
				continue
			}
			var t todo.Todo
			if isContextList {
				t, err = s.ReadTodo(contextUUID, uuid)
			} else {
				t, err = s.FindTodo(uuid)
			}
			if err != nil {
				continue
			}
			tc := t
			items = append(items, GroomItem{SectionIdx: si, Todo: &tc})
		}
	}
	return items, nil
}

func (m GroomModel) currentListName() string {
	if m.listIdx < 0 || m.listIdx >= len(m.listOrder) {
		return ""
	}
	return m.listOrder[m.listIdx]
}

func (m GroomModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listLoadedMsg:
		m.items = msg.items
		// Set cursor to first todo
		m.cursor = 0
		for i, item := range m.items {
			if item.Todo != nil {
				m.cursor = i
				break
			}
		}
		return m, nil

	case groomErrMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m GroomModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// If we have a pending list key, expect a section number
	if m.pendingListKey != "" {
		listName := ListKeyMap[m.pendingListKey]
		m.pendingListKey = ""
		if key >= "1" && key <= "9" {
			sectionIdx := int(key[0]-'0') - 1
			return m.moveTodoToListSection(listName, sectionIdx)
		}
		// No number followed — just move to end of list
		return m.moveTodoToList(listName)
	}

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		if m.dirty {
			_ = m.store.Commit("groom session")
		}
		return m, tea.Quit

	case "tab":
		m.listIdx = (m.listIdx + 1) % len(m.listOrder)
		return m, loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)

	case "shift+tab":
		m.listIdx = (m.listIdx - 1 + len(m.listOrder)) % len(m.listOrder)
		return m, loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)

	case "up", "k":
		m.cursor = m.groomPrevTodo()

	case "down", "j":
		m.cursor = m.groomNextTodo()

	case " ", "d":
		return m.completeTodo()

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		n := int(key[0]-'0') - 1
		if n < len(m.listOrder) {
			m.listIdx = n
			return m, loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)
		}

	default:
		// Check for list shortcut keys
		if listName, ok := ListKeyMap[key]; ok {
			if m.currentTodo() != nil {
				// Check if already on that list — if so, set pending for section
				if listName == m.currentListName() {
					m.pendingListKey = key
					return m, nil
				}
				return m.moveTodoToList(listName)
			} else {
				// Navigate to that list
				for i, name := range m.listOrder {
					if name == listName {
						m.listIdx = i
						return m, loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)
					}
				}
			}
		}
	}

	return m, nil
}

func (m GroomModel) currentTodo() *todo.Todo {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return m.items[m.cursor].Todo
}

func (m GroomModel) completeTodo() (GroomModel, tea.Cmd) {
	t := m.currentTodo()
	if t == nil {
		return m, nil
	}
	uuid := t.UUID
	if err := m.store.DeleteTodo(m.contextUUID, uuid); err != nil {
		m.err = err
		return m, nil
	}
	if err := m.store.RemoveFromAllLists(m.contextUUID, uuid); err != nil {
		m.err = err
		return m, nil
	}
	m.dirty = true
	m.touched[uuid] = true
	return m, loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)
}

func (m GroomModel) moveTodoToList(listName string) (GroomModel, tea.Cmd) {
	return m.moveTodoToListSection(listName, -1)
}

// moveTodoToListSection moves the current todo to the given list and section.
// sectionIdx -1 appends to end. "inbox" is the virtual inbox: moving there
// just removes from all named lists so the todo falls back into the inbox.
func (m GroomModel) moveTodoToListSection(listName string, sectionIdx int) (GroomModel, tea.Cmd) {
	t := m.currentTodo()
	if t == nil {
		return m, nil
	}
	uuid := t.UUID

	if err := m.store.RemoveFromAllLists(m.contextUUID, uuid); err != nil {
		m.err = err
		return m, nil
	}

	if listName != "inbox" {
		l, err := m.store.ReadList(m.contextUUID, listName)
		if err != nil {
			m.err = err
			return m, nil
		}

		if sectionIdx < 0 || sectionIdx >= len(l.Sections) {
			list.Add(&l, uuid, false)
		} else {
			l.Sections[sectionIdx].Items = append(l.Sections[sectionIdx].Items, uuid)
		}

		if err := m.store.WriteList(m.contextUUID, listName, l); err != nil {
			m.err = err
			return m, nil
		}
	}

	m.dirty = true
	m.touched[uuid] = true
	return m, loadListCmd(m.store, m.contextUUID, m.currentListName(), m.touched)
}

func (m GroomModel) groomNextTodo() int {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].Todo != nil {
			return i
		}
	}
	return m.cursor
}

func (m GroomModel) groomPrevTodo() int {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].Todo != nil {
			return i
		}
	}
	return m.cursor
}

func (m GroomModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	listName := m.currentListName()
	sb.WriteString(styleHeader.Render("bliss groom") + "  " + styleHeader.Render("["+listName+"]") + "\n")

	sb.WriteString(styleMuted.Render("tab/shift-tab switch  d/space complete  q quit") + "\n")
	for i, name := range m.listOrder {
		if i == m.listIdx {
			sb.WriteString(styleCursor.Render(fmt.Sprintf(" %d:%s", i+1, name)))
		} else {
			sb.WriteString(styleMuted.Render(fmt.Sprintf(" %d:%s", i+1, name)))
		}
	}
	sb.WriteString("\n\n")

	if m.err != nil {
		sb.WriteString(fmt.Sprintf("error: %v\n", m.err))
	}

	todoCount := 0
	for i, item := range m.items {
		if item.SectionHeader != "" {
			sb.WriteString("\n" + styleSectionHead.Render("── "+item.SectionHeader) + "\n")
			continue
		}
		if item.Todo == nil {
			continue
		}
		todoCount++
		if i == m.cursor {
			sb.WriteString(styleCursor.Render("> ") + lipgloss.NewStyle().Bold(true).Render(item.Todo.Title) + "\n")
		} else {
			sb.WriteString("  " + item.Todo.Title + "\n")
		}
	}

	if todoCount == 0 {
		sb.WriteString(styleMuted.Render("  (no todos)") + "\n")
	}

	if m.pendingListKey != "" {
		sb.WriteString("\n" + styleMuted.Render("move to section: press 1-9") + "\n")
	}

	return sb.String()
}
