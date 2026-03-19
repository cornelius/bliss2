package ui

import "github.com/charmbracelet/lipgloss"

var (
	styleHeader      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleListHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleCursor      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	styleSectionHead = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleMuted       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleEditing     = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
)
