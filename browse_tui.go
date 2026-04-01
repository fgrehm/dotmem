package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
)

// -- wrapping delegate --

type wrappingDelegate struct {
	list.DefaultDelegate
}

func newWrappingDelegate() wrappingDelegate {
	d := list.NewDefaultDelegate()
	d.SetHeight(3)
	d.SetSpacing(1)
	return wrappingDelegate{DefaultDelegate: d}
}

func (d wrappingDelegate) Height() int  { return d.DefaultDelegate.Height() }
func (d wrappingDelegate) Spacing() int { return d.DefaultDelegate.Spacing() }

func (d wrappingDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	di, ok := item.(list.DefaultItem)
	if !ok {
		return
	}
	if m.Width() <= 0 {
		return
	}

	s := &d.Styles
	padLeft := s.NormalTitle.GetPaddingLeft()
	textWidth := m.Width() - padLeft - s.NormalTitle.GetPaddingRight()

	title := ansi.Wordwrap(di.Title(), textWidth, " ")
	title = clampLines(title, 1)
	desc := ansi.Truncate(di.Description(), textWidth, "…")

	isSelected := index == m.Index()
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""

	switch {
	case emptyFilter:
		title = s.DimmedTitle.Render(title)
		desc = s.DimmedDesc.Render(desc)
	case isSelected && m.FilterState() != list.Filtering:
		title = s.SelectedTitle.Render(title)
		desc = s.SelectedDesc.Render(desc)
	default:
		title = s.NormalTitle.Render(title)
		desc = s.NormalDesc.Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc) //nolint:errcheck
}

func clampLines(s string, max int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > max {
		lines = lines[:max]
	}
	return strings.Join(lines, "\n")
}

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
	return i.memory.Project + "  " + desc
}

func (i memoryItem) FilterValue() string {
	return displayName(i.memory) + " " + i.memory.Project + " " + i.memory.Meta.Type + " " + i.memory.Meta.Description
}

// -- messages --

type (
	editorFinishedMsg  struct{ err error }
	editCommittedMsg   struct{ err error }
	deleteCommittedMsg struct{ index int }
)

// -- model --

type browseView int

const (
	listView browseView = iota
	detailView
)

type browseModel struct {
	list       list.Model
	viewport   viewport.Model
	view       browseView
	selected   memoryFile
	dotmemDir  string
	width      int
	height     int
	ready      bool
	confirming bool // waiting for y/n on delete
}

func newBrowseModel(memories []memoryFile, title, dotmemDir string) browseModel {
	items := make([]list.Item, len(memories))
	for i, m := range memories {
		items[i] = memoryItem{memory: m}
	}

	delegate := newWrappingDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return browseModel{
		list:      l,
		dotmemDir: dotmemDir,
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

	case editorFinishedMsg:
		if msg.err != nil {
			return m, nil
		}
		// Re-read the file and update the list item.
		filePath := filepath.Join(m.dotmemDir, m.selected.Project, m.selected.File)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return m, nil
		}
		content := string(data)
		meta, body := parseFrontmatter(content)
		m.selected = memoryFile{
			Project: m.selected.Project,
			File:    m.selected.File,
			Meta:    meta,
			Body:    body,
		}
		// Update the list item.
		idx := m.list.Index()
		_ = m.list.SetItem(idx, memoryItem{memory: m.selected})
		// Refresh viewport content.
		m.viewport.SetContent(renderDetail(m.selected, m.width))
		// Commit via tea.Cmd to sequence within the update loop.
		dotmemDir := m.dotmemDir
		project := m.selected.Project
		file := m.selected.File
		return m, func() tea.Msg {
			return editCommittedMsg{err: commitMemoryChange(dotmemDir, project, file, "browse: edit")}
		}

	case editCommittedMsg:
		if msg.err != nil {
			fmt.Fprintf(os.Stderr, "dotmem: commit after edit: %v\n", msg.err)
		}
		return m, nil

	case deleteCommittedMsg:
		m.list.RemoveItem(msg.index)
		m.view = listView
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		if m.view == detailView {
			if m.confirming {
				switch key {
				case "y", "Y":
					idx := m.list.Index()
					mem := m.selected
					dir := m.dotmemDir
					return m, func() tea.Msg {
						if err := deleteMemory(dir, mem); err != nil {
							fmt.Fprintf(os.Stderr, "dotmem: delete memory: %v\n", err)
							return nil
						}
						if cerr := commitMemoryChange(dir, mem.Project, mem.File, "browse: delete"); cerr != nil {
							fmt.Fprintf(os.Stderr, "dotmem: commit after delete: %v\n", cerr)
						}
						return deleteCommittedMsg{index: idx}
					}
				default:
					m.confirming = false
				}
				return m, nil
			}

			switch key {
			case "esc", "backspace":
				m.view = listView
				m.confirming = false
				return m, nil
			case "q":
				m.view = listView
				return m, nil
			case "d":
				m.confirming = true
				return m, nil
			case "e":
				filePath := filepath.Join(m.dotmemDir, m.selected.Project, m.selected.File)
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = os.Getenv("VISUAL")
				}
				if editor == "" {
					editor = "vi"
				}
				editorFields := strings.Fields(editor)
				if len(editorFields) == 0 {
					editorFields = []string{"vi"}
				}
				cmdArgs := make([]string, len(editorFields)-1, len(editorFields))
				copy(cmdArgs, editorFields[1:])
				cmdArgs = append(cmdArgs, filePath)
				return m, tea.ExecProcess(exec.Command(editorFields[0], cmdArgs...), func(err error) tea.Msg {
					return editorFinishedMsg{err: err}
				})
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
			m.selected = selected.memory
			m.confirming = false
			m.viewport = viewport.New(
				viewport.WithWidth(m.width),
				viewport.WithHeight(m.height-2),
			)
			m.viewport.SetContent(renderDetail(m.selected, m.width))
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
		var footer string
		if m.confirming {
			footer = warningStyle.Render("Delete this memory? ") + helpStyle.Render("y") + warningStyle.Render("/") + helpStyle.Render("n")
		} else {
			footer = helpStyle.Render("esc: back  d: delete  e: edit  ↑↓: scroll")
		}
		v.SetContent(m.viewport.View() + "\n" + footer)
		return v
	}

	v.SetContent(m.list.View())
	return v
}

// -- detail rendering --

func renderDetail(mem memoryFile, width int) string {
	var b strings.Builder

	b.WriteString(detailHeaderStyle.Render(displayName(mem)))
	b.WriteString("\n")

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

// -- delete --

func deleteMemory(dotmemDir string, mem memoryFile) error {
	filePath := filepath.Join(dotmemDir, mem.Project, mem.File)
	if err := os.Remove(filePath); err != nil {
		return err
	}
	return cascadeMemoryIndex(dotmemDir, mem.Project, mem.File)
}

// cascadeMemoryIndex removes any line in MEMORY.md that links to the deleted file.
func cascadeMemoryIndex(dotmemDir, project, file string) error {
	indexPath := filepath.Join(dotmemDir, project, "MEMORY.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read memory index: %w", err)
	}
	// Match markdown link lines referencing this file: - [...](file.md) ...
	pattern := regexp.MustCompile(`(?m)^[^\n]*\(` + regexp.QuoteMeta(file) + `\)[^\n]*\n?`)
	updated := pattern.ReplaceAll(data, nil)
	if len(updated) == len(data) {
		return nil
	}
	tmpPath := indexPath + ".tmp"
	if err := os.WriteFile(tmpPath, updated, 0o644); err != nil {
		return fmt.Errorf("write temp memory index: %w", err)
	}
	if err := os.Rename(tmpPath, indexPath); err != nil {
		return fmt.Errorf("replace memory index: %w", err)
	}
	return nil
}

// commitMemoryChange stages and commits changes to a single project's memory file.
func commitMemoryChange(dotmemDir, project, file, msg string) error {
	filePath := filepath.Join(project, file)
	indexPath := filepath.Join(project, "MEMORY.md")
	// Stage the memory file (may be deleted or modified) and MEMORY.md.
	if _, err := gitExec(dotmemDir, "add", filePath, indexPath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	commitMsg := msg + ": " + project + "/" + file
	if _, err := gitExec(dotmemDir, "commit", "-m", commitMsg, "--", filePath, indexPath); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// -- entry point --

func cmdBrowseTUI(w io.Writer, typeFilter, projectFilter string, allProjects bool) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if err := requireInit(dir); err != nil {
		return err
	}

	projectFilter, err = resolveProjectFilter(dir, projectFilter, allProjects)
	if err != nil {
		return err
	}

	memories, err := collectMemories(dir)
	if err != nil {
		return err
	}

	memories = filterMemories(memories, typeFilter, projectFilter)
	sortMemories(memories)

	if len(memories) == 0 {
		fmt.Fprintln(w, "no memories found")
		return nil
	}

	title := "Memories"
	if projectFilter != "" {
		title = projectFilter
	}

	p := tea.NewProgram(newBrowseModel(memories, title, dir))
	_, err = p.Run()
	return err
}
