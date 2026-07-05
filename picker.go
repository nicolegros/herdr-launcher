package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// Styles
var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // cyan
	countStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
	nameStyle    = lipgloss.NewStyle()
	nameSelStyle = lipgloss.NewStyle().Bold(true)
	matchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	barStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
)

// pickerModel is the Bubble Tea model for the directory picker.
type pickerModel struct {
	entries  []DirEntry
	warnings []string

	input    textinput.Model
	filtered []scoredEntry
	cursor   int
	chosen   *DirEntry
	width    int
	height   int
}

type scoredEntry struct {
	entry   DirEntry
	matched []int
}

func newPickerModel(entries []DirEntry, warnings []string) pickerModel {
	ti := textinput.New()
	ti.Placeholder = "search projects..."
	ti.Prompt = ""
	ti.Focus()

	m := pickerModel{
		entries:  entries,
		warnings: warnings,
		input:    ti,
	}
	m.filter()
	return m
}

func (m *pickerModel) filter() {
	q := strings.TrimSpace(m.input.Value())
	m.filtered = m.filtered[:0]

	if q == "" {
		for _, e := range m.entries {
			m.filtered = append(m.filtered, scoredEntry{entry: e})
		}
		// Pin active (open) workspaces to the top
		sort.SliceStable(m.filtered, func(i, j int) bool {
			return m.filtered[i].entry.Active && !m.filtered[j].entry.Active
		})
		m.clampCursor()
		return
	}

	// Build haystacks: "name  parent_path" for matching
	haystacks := make([]string, len(m.entries))
	nameLens := make([]int, len(m.entries))
	for i, e := range m.entries {
		haystacks[i] = e.Name + "  " + e.ParentPath
		nameLens[i] = len(e.Name)
	}

	for _, mt := range fuzzy.Find(q, haystacks) {
		var inName []int
		for _, idx := range mt.MatchedIndexes {
			if idx < nameLens[mt.Index] {
				inName = append(inName, idx)
			}
		}
		m.filtered = append(m.filtered, scoredEntry{entry: m.entries[mt.Index], matched: inName})
	}
	m.clampCursor()
}

func (m *pickerModel) clampCursor() {
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				e := m.filtered[m.cursor].entry
				m.chosen = &e
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	// Pass to text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filter()
	return m, cmd
}

func (m pickerModel) View() string {
	var b strings.Builder

	// Warnings
	for _, w := range m.warnings {
		b.WriteString(warnStyle.Render("⚠ " + w))
		b.WriteString("\n")
	}

	// Prompt
	b.WriteString(promptStyle.Render("❯ "))
	b.WriteString(m.input.View())
	b.WriteString("   ")
	b.WriteString(countStyle.Render(fmt.Sprintf("%d/%d", len(m.filtered), len(m.entries))))
	b.WriteString("\n\n")

	// Results
	if len(m.filtered) == 0 {
		b.WriteString(descStyle.Render("  no matches"))
		b.WriteString("\n")
	}

	// Limit visible rows to available height
	maxRows := m.height - 3 - len(m.warnings) // prompt + count + padding
	if maxRows < 1 {
		maxRows = 10
	}

	// Scroll window around cursor
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		s := m.filtered[i]
		selected := i == m.cursor
		if selected {
			b.WriteString(barStyle.Render("▌ "))
		} else {
			b.WriteString("  ")
		}
		if s.entry.Active {
			b.WriteString(activeStyle.Render("● "))
		}
		b.WriteString(highlightName(s.entry.Name, s.matched, selected))
		b.WriteString("  ")
		b.WriteString(descStyle.Render(s.entry.ParentPath))
		b.WriteString("\n")
	}

	return b.String()
}

// highlightName renders a name with fuzzy-matched characters emphasized.
func highlightName(name string, matched []int, selected bool) string {
	base := nameStyle
	if selected {
		base = nameSelStyle
	}
	if len(matched) == 0 {
		return base.Render(name)
	}

	set := make(map[int]bool, len(matched))
	for _, idx := range matched {
		set[idx] = true
	}

	var b strings.Builder
	for i, r := range name {
		if set[i] {
			b.WriteString(matchStyle.Render(string(r)))
		} else {
			b.WriteString(base.Render(string(r)))
		}
	}
	return b.String()
}
