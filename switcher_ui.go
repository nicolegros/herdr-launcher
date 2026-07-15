package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// switcherModel is the Bubble Tea model for the workspace history switcher overlay.
type switcherModel struct {
	entries  []switcherEntry
	cursor   int
	chosen   *switcherEntry
	width    int
	height   int
}

type switcherEntry struct {
	WorkspaceID string
	Label       string
	Current     bool // true for the workspace the user is currently on
}

func newSwitcherModel(entries []switcherEntry, startCursor int) switcherModel {
	cursor := startCursor
	if cursor >= len(entries) {
		cursor = 0
	}
	return switcherModel{
		entries: entries,
		cursor:  cursor,
	}
}

func (m switcherModel) Init() tea.Cmd {
	return nil
}

func (m switcherModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.entries) > 0 && m.cursor < len(m.entries) {
				e := m.entries[m.cursor]
				m.chosen = &e
			}
			return m, tea.Quit
		case "up", "k", "shift+tab":
			if len(m.entries) > 0 {
				m.cursor = ((m.cursor - 1) + len(m.entries)) % len(m.entries)
			}
			return m, nil
		case "down", "j", "tab":
			if len(m.entries) > 0 {
				m.cursor = (m.cursor + 1) % len(m.entries)
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	return m, nil
}

func (m switcherModel) View() string {
	if len(m.entries) == 0 {
		return descStyle.Render("  no workspace history")
	}

	var b strings.Builder

	// Title
	b.WriteString(promptStyle.Render("switch workspace"))
	b.WriteString("\n\n")

	// Limit visible rows
	maxRows := m.height - 3
	if maxRows < 1 {
		maxRows = 10
	}

	// Scroll window around cursor
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.entries) {
		end = len(m.entries)
	}

	for i := start; i < end; i++ {
		e := m.entries[i]
		selected := i == m.cursor

		if selected {
			b.WriteString(barStyle.Render("▌ "))
		} else {
			b.WriteString("  ")
		}

		if e.Current {
			b.WriteString(activeStyle.Render("● "))
		} else {
			b.WriteString("  ")
		}

		label := e.Label
		if selected {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render(label))
		} else {
			b.WriteString(nameStyle.Render(label))
		}

		b.WriteString("\n")
	}

	return b.String()
}
