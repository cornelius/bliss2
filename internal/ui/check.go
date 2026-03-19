// Package ui provides interactive terminal UI components for bliss.
package ui

import (
	"bliss/internal/store"
	"bliss/internal/todo"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// CheckItem holds a todo to display in the check view.
type CheckItem struct {
	SectionHeader string // non-empty if this is a section header row
	Todo          *todo.Todo
}

// CheckModel is the bubbletea model for the check command.
type CheckModel struct {
	store       *store.Store
	contextUUID string
	items       []CheckItem
	cursor      int
	editing     bool
	textInput   textinput.Model
	dirty       bool
	quitting    bool
	err         error
}

// NewCheckModel creates a new CheckModel with the given todos.
func NewCheckModel(s *store.Store, contextUUID string, items []CheckItem) CheckModel {
	ti := textinput.New()
	ti.Placeholder = "Edit title..."
	// Set cursor to first todo item
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
		items:       items,
		cursor:      cursor,
		textInput:   ti,
	}
}

func (m CheckModel) Init() tea.Cmd {
	return nil
}

func (m CheckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.cursor = m.prevTodo()

		case "down", "j":
			m.cursor = m.nextTodo()

		case "enter":
			if item := m.currentTodoItem(); item != nil {
				m.textInput.SetValue(item.Todo.Title)
				m.textInput.Focus()
				m.editing = true
				return m, textinput.Blink
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
				// Remove from items
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

func (m CheckModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("bliss check — arrows to navigate, enter to edit, space/d to complete, q to quit\n\n")

	if m.err != nil {
		sb.WriteString(fmt.Sprintf("error: %v\n", m.err))
	}

	for i, item := range m.items {
		if item.SectionHeader != "" {
			sb.WriteString(fmt.Sprintf("\n[%s]\n", item.SectionHeader))
			continue
		}
		if item.Todo == nil {
			continue
		}

		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		if m.editing && i == m.cursor {
			sb.WriteString(cursor + m.textInput.View() + "\n")
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", cursor, item.Todo.Title))
		}
	}

	if len(m.todoOnlyItems()) == 0 {
		sb.WriteString("  (no todos)\n")
	}

	return sb.String()
}

// currentTodoItem returns the CheckItem at cursor if it's a todo (not header).
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

// todoOnlyItems returns only the non-header items.
func (m CheckModel) todoOnlyItems() []CheckItem {
	var result []CheckItem
	for _, item := range m.items {
		if item.Todo != nil {
			result = append(result, item)
		}
	}
	return result
}

// nextTodo returns the index of the next todo item (skipping headers).
func (m CheckModel) nextTodo() int {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].Todo != nil {
			return i
		}
	}
	return m.cursor
}

// prevTodo returns the index of the previous todo item (skipping headers).
func (m CheckModel) prevTodo() int {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].Todo != nil {
			return i
		}
	}
	return m.cursor
}
