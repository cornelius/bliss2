// Package ui provides interactive terminal UI components for bliss.
package ui

import (
	"bliss/internal/list"
	"bliss/internal/store"
	"bliss/internal/todo"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CheckItem holds a row in the check view — a list header, section separator, or todo.
type CheckItem struct {
	IsSectionHeader bool   // true if this is a section separator row
	IsListHeader    bool   // true if this is a list-name header row
	SectionHeader   string // section name; empty for unnamed sections
	SectionName     string // actual stored name (same as SectionHeader, kept for clarity)
	SectionIdx      int    // index into the list's Sections slice
	ListName        string // list this todo belongs to (used for section insertion)
	ListContextUUID string // context UUID for that list (empty for personal lists)
	Todo            *todo.Todo
}

// CheckModel is the bubbletea model for the check command.
type CheckModel struct {
	store          *store.Store
	contextUUID    string
	listName       string // non-empty when viewing a single named list
	items          []CheckItem
	cursor         int
	editing        bool // editing a todo title
	editingSection bool // editing a section name
	textInput      textinput.Model
	dirty          bool
	quitting       bool
	err            error
}

// NewCheckModel creates a new CheckModel. listName is non-empty when viewing a single named list.
func NewCheckModel(s *store.Store, contextUUID string, items []CheckItem, listName string) CheckModel {
	ti := textinput.New()
	cursor := 0
	for i, item := range items {
		if item.Todo != nil {
			cursor = i
			break
		}
	}
	return CheckModel{
		store:       s,
		contextUUID: contextUUID,
		listName:    listName,
		items:       items,
		cursor:      cursor,
		textInput:   ti,
	}
}

func (m CheckModel) Init() tea.Cmd {
	return nil
}

func (m CheckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editingSection {
		return m.updateEditingSection(msg)
	}
	if m.editing {
		return m.updateEditing(msg)
	}
	return m.updateNormal(msg)
}

func (m CheckModel) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			if m.dirty {
				_ = m.store.Commit("check session")
			}
			return m, tea.Quit

		case "up", "k":
			m.cursor = m.prevItem()

		case "down", "j":
			m.cursor = m.nextItem()

		case "enter":
			if m.cursor < 0 || m.cursor >= len(m.items) {
				break
			}
			item := &m.items[m.cursor]
			if item.IsSectionHeader && m.listName != "" {
				m.textInput.SetValue(item.SectionName)
				m.textInput.Placeholder = "Section name..."
				m.textInput.Focus()
				m.editingSection = true
				return m, textinput.Blink
			}
			if item.Todo != nil {
				m.textInput.SetValue(item.Todo.Title)
				m.textInput.Placeholder = "Edit title..."
				m.textInput.Focus()
				m.editing = true
				return m, textinput.Blink
			}

		case "s":
			if m.currentTodoItem() != nil {
				newM, err := m.insertSection()
				if err != nil {
					m.err = err
					return m, nil
				}
				return newM, nil
			}

		case " ", "d":
			if item := m.currentTodoItem(); item != nil {
				uuid := item.Todo.UUID
				if err := m.store.DeleteTodo(m.contextUUID, uuid); err != nil {
					m.err = err
					return m, nil
				}
				if err := m.store.RemoveFromAllLists(m.contextUUID, uuid); err != nil {
					m.err = err
					return m, nil
				}
				m.dirty = true
				newItems := make([]CheckItem, 0, len(m.items))
				for _, it := range m.items {
					if it.Todo == nil || it.Todo.UUID != uuid {
						newItems = append(newItems, it)
					}
				}
				m.items = newItems
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		}
	}
	return m, nil
}

func (m CheckModel) updateEditing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item := m.currentTodoItem(); item != nil {
				newTitle := strings.TrimSpace(m.textInput.Value())
				if newTitle != "" {
					item.Todo.Title = newTitle
					t := *item.Todo
					if err := m.store.WriteTodo(m.contextUUID, t); err != nil {
						m.err = err
					} else {
						m.dirty = true
					}
				}
			}
			m.textInput.Blur()
			m.editing = false
			return m, nil

		case "esc":
			m.textInput.Blur()
			m.editing = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m CheckModel) updateEditingSection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.textInput.Value())
			m.textInput.Blur()
			m.editingSection = false
			item := &m.items[m.cursor]
			newM, err := m.renameSection(item.SectionIdx, name)
			if err != nil {
				m.err = err
				return m, nil
			}
			return newM, nil

		case "esc":
			m.textInput.Blur()
			m.editingSection = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// insertSection inserts an unnamed section separator after the current todo.
func (m CheckModel) insertSection() (CheckModel, error) {
	item := m.currentTodoItem()
	if item == nil {
		return m, nil
	}

	listName := m.listName
	listCtx := m.contextUUID
	if listName == "" {
		listName = item.ListName
		listCtx = item.ListContextUUID
	}
	if listName == "" {
		return m, nil
	}

	uuid := item.Todo.UUID
	l, err := m.store.ReadList(listCtx, listName)
	if err != nil {
		return m, err
	}

	inserted := false
	newSectionIdx := 0
	for si := range l.Sections {
		for pi, id := range l.Sections[si].Items {
			if id != uuid {
				continue
			}
			before := append([]string(nil), l.Sections[si].Items[:pi+1]...)
			after := append([]string(nil), l.Sections[si].Items[pi+1:]...)
			l.Sections[si].Items = before
			tail := append([]list.Section(nil), l.Sections[si+1:]...)
			l.Sections = append(l.Sections[:si+1], append([]list.Section{{Items: after}}, tail...)...)
			newSectionIdx = si + 1
			inserted = true
			break
		}
		if inserted {
			break
		}
	}
	if !inserted {
		return m, nil
	}

	if err := m.store.WriteList(listCtx, listName, l); err != nil {
		return m, err
	}
	m.dirty = true
	prevCursor := m.cursor

	if m.listName != "" {
		m.items = itemsFromList(l, m.store, m.contextUUID)
	} else {
		newItem := CheckItem{IsSectionHeader: true, SectionIdx: newSectionIdx}
		tail := append([]CheckItem(nil), m.items[prevCursor+1:]...)
		m.items = append(m.items[:prevCursor+1], append([]CheckItem{newItem}, tail...)...)
	}

	m.cursor = prevCursor + 1
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	return m, nil
}

// renameSection updates the name of the section at sectionIdx.
func (m CheckModel) renameSection(sectionIdx int, name string) (CheckModel, error) {
	l, err := m.store.ReadList(m.contextUUID, m.listName)
	if err != nil {
		return m, err
	}
	if sectionIdx >= 0 && sectionIdx < len(l.Sections) {
		l.Sections[sectionIdx].Name = name
	}
	if err := m.store.WriteList(m.contextUUID, m.listName, l); err != nil {
		return m, err
	}
	m.dirty = true
	m.items = itemsFromList(l, m.store, m.contextUUID)
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	return m, nil
}

// itemsFromList builds CheckItems from a list, preserving section separators.
func itemsFromList(l list.List, s *store.Store, contextUUID string) []CheckItem {
	var items []CheckItem
	for si, section := range l.Sections {
		if si > 0 {
			items = append(items, CheckItem{
				IsSectionHeader: true,
				SectionHeader:   section.Name,
				SectionName:     section.Name,
				SectionIdx:      si,
			})
		}
		for _, uuid := range section.Items {
			t, err := s.ReadTodo(contextUUID, uuid)
			if err != nil {
				continue
			}
			tc := t
			items = append(items, CheckItem{Todo: &tc})
		}
	}
	return items
}

func (m CheckModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(styleHeader.Render("bliss check") + styleMuted.Render("  ↑↓ navigate  enter edit  space/d complete  s section  q quit") + "\n\n")

	if m.err != nil {
		sb.WriteString(fmt.Sprintf("error: %v\n", m.err))
	}

	for i, item := range m.items {
		if item.IsListHeader {
			prefix := "\n"
			if i == 0 {
				prefix = ""
			}
			sb.WriteString(prefix + styleListHeader.Render("["+item.SectionHeader+"]") + "\n")
			continue
		}
		if item.IsSectionHeader {
			if item.SectionHeader == "" {
				if m.cursor == i {
					sb.WriteString(styleCursor.Render("> ──") + "\n")
				} else {
					sb.WriteString(styleSectionHead.Render("  ──") + "\n")
				}
			} else if m.editingSection && i == m.cursor {
				sb.WriteString(styleEditing.Render("> ") + m.textInput.View() + "\n")
			} else if i == m.cursor {
				sb.WriteString(styleCursor.Render("> ") + styleSectionHead.Render("── "+item.SectionHeader) + "\n")
			} else {
				sb.WriteString(styleSectionHead.Render("  ── "+item.SectionHeader) + "\n")
			}
			continue
		}
		if item.Todo == nil {
			continue
		}

		if m.editing && i == m.cursor {
			sb.WriteString(styleEditing.Render("> ") + m.textInput.View() + "\n")
		} else if i == m.cursor {
			sb.WriteString(styleCursor.Render("> ") + lipgloss.NewStyle().Bold(true).Render(item.Todo.Title) + "\n")
		} else {
			sb.WriteString("  " + item.Todo.Title + "\n")
		}
	}

	if len(m.todoOnlyItems()) == 0 {
		sb.WriteString(styleMuted.Render("  (no todos)") + "\n")
	}

	return sb.String()
}

// currentTodoItem returns the CheckItem at cursor if it's a todo.
func (m *CheckModel) currentTodoItem() *CheckItem {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	item := &m.items[m.cursor]
	if item.Todo == nil {
		return nil
	}
	return item
}

// todoOnlyItems returns only the todo items.
func (m CheckModel) todoOnlyItems() []CheckItem {
	var result []CheckItem
	for _, item := range m.items {
		if item.Todo != nil {
			result = append(result, item)
		}
	}
	return result
}

// nextItem returns the index of the next navigable item.
func (m CheckModel) nextItem() int {
	if m.cursor+1 < len(m.items) {
		return m.cursor + 1
	}
	return m.cursor
}

// prevItem returns the index of the previous navigable item.
func (m CheckModel) prevItem() int {
	if m.cursor-1 >= 0 {
		return m.cursor - 1
	}
	return m.cursor
}
