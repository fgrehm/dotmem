package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
)

// -- styles --

var (
	typeBadgeStyles = map[string]lipgloss.Style{
		"user":      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")),
		"feedback":  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B")),
		"project":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34D399")),
		"reference": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#60A5FA")),
		"untyped":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9CA3AF")),
	}

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				PaddingLeft(1)

	detailMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			PaddingLeft(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

func typeBadge(t string) string {
	if t == "" {
		t = "untyped"
	}
	style, ok := typeBadgeStyles[t]
	if !ok {
		style = typeBadgeStyles["untyped"]
	}
	return style.Render(t)
}

// -- list item --

type memoryItem struct {
	memory memoryFile
}

func (i memoryItem) Title() string {
	return fmt.Sprintf("%s  %s", typeBadge(i.memory.Meta.Type), displayName(i.memory))
}

func (i memoryItem) Description() string {
	desc := i.memory.Meta.Description
	if desc == "" {
		desc = i.memory.File
	}
	return fmt.Sprintf("%-14s %s", i.memory.Project, desc)
}

func (i memoryItem) FilterValue() string {
	return displayName(i.memory) + " " + i.memory.Project + " " + i.memory.Meta.Type + " " + i.memory.Meta.Description
}

// -- model --

type browseView int

const (
	listView browseView = iota
	detailView
)

type browseModel struct {
	list     list.Model
	viewport viewport.Model
	view     browseView
	width    int
	height   int
	ready    bool
}

func newBrowseModel(memories []memoryFile) browseModel {
	items := make([]list.Item, len(memories))
	for i, m := range memories {
		items[i] = memoryItem{memory: m}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Memories"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return browseModel{
		list: l,
	}
}

func (m browseModel) Init() tea.Cmd {
	return nil
}

func (m browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		m.ready = true
		if m.view == detailView {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - 2)
		}
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		if m.view == detailView {
			if key == "esc" || key == "backspace" || key == "q" {
				m.view = listView
				return m, nil
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		// List view keys.
		if m.list.SettingFilter() {
			break
		}

		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		case "enter":
			selected, ok := m.list.SelectedItem().(memoryItem)
			if !ok {
				return m, nil
			}
			m.viewport = viewport.New(
				viewport.WithWidth(m.width),
				viewport.WithHeight(m.height-2),
			)
			m.viewport.SetContent(renderDetail(selected.memory, m.width))
			m.view = detailView
			return m, nil
		}
	}

	if m.view == listView {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m browseModel) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if !m.ready {
		v.SetContent("Loading...")
		return v
	}

	if m.view == detailView {
		footer := helpStyle.Render("esc: back  q: quit  ↑↓: scroll")
		v.SetContent(m.viewport.View() + "\n" + footer)
		return v
	}

	v.SetContent(m.list.View())
	return v
}

// -- detail rendering --

func renderDetail(mem memoryFile, width int) string {
	var b strings.Builder

	// Header.
	b.WriteString(detailHeaderStyle.Render(displayName(mem)))
	b.WriteString("\n")

	// Meta line.
	var meta []string
	meta = append(meta, typeBadge(mem.Meta.Type))
	meta = append(meta, mem.Project+"/"+mem.File)
	b.WriteString(detailMetaStyle.Render(strings.Join(meta, "  ")))
	b.WriteString("\n")

	if mem.Meta.Description != "" {
		b.WriteString(detailMetaStyle.Render(mem.Meta.Description))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Body rendered as markdown.
	body := strings.TrimSpace(mem.Body)
	if body != "" {
		w := max(width-2, 40)
		r, err := glamour.NewTermRenderer(
			glamour.WithWordWrap(w),
			glamour.WithStandardStyle("dark"),
		)
		if err == nil {
			rendered, err := r.Render(body)
			if err == nil {
				b.WriteString(rendered)
			} else {
				b.WriteString(body)
			}
		} else {
			b.WriteString(body)
		}
	}

	return b.String()
}

// -- entry point --

func cmdBrowseTUI(typeFilter, projectFilter string) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if err := requireInit(dir); err != nil {
		return err
	}

	memories, err := collectMemories(dir)
	if err != nil {
		return err
	}

	memories = filterMemories(memories, typeFilter, projectFilter)
	sortMemories(memories)

	if len(memories) == 0 {
		fmt.Println("no memories found")
		return nil
	}

	p := tea.NewProgram(newBrowseModel(memories))
	_, err = p.Run()
	return err
}
