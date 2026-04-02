package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/spf13/cobra"
)

type memoryMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

type memoryFile struct {
	Project string
	File    string
	Meta    memoryMeta
	Body    string
}

// parseFrontmatter extracts YAML frontmatter from a memory file.
// Returns a zero-value memoryMeta and the full content as body when
// frontmatter is missing or malformed (fail open).
func parseFrontmatter(content string) (memoryMeta, string) {
	var meta memoryMeta
	rest, err := frontmatter.Parse(strings.NewReader(content), &meta)
	if err != nil {
		return memoryMeta{}, content
	}
	return meta, string(rest)
}

func collectMemories(dotmemDir string) ([]memoryFile, error) {
	entries, err := os.ReadDir(dotmemDir)
	if err != nil {
		return nil, err
	}

	var memories []memoryFile
	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".git" {
			continue
		}
		slug := e.Name()
		projectDir := filepath.Join(dotmemDir, slug)
		files, err := readMemoryFiles(projectDir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", slug, err)
		}
		for name, content := range files {
			if name == "MEMORY.md" {
				continue
			}
			meta, body := parseFrontmatter(content)
			memories = append(memories, memoryFile{
				Project: slug,
				File:    name,
				Meta:    meta,
				Body:    body,
			})
		}
	}
	return memories, nil
}

func filterMemories(memories []memoryFile, typeFilter, projectFilter string) []memoryFile {
	if typeFilter == "" && projectFilter == "" {
		return memories
	}
	var filtered []memoryFile
	for _, m := range memories {
		if typeFilter != "" {
			mt := m.Meta.Type
			if mt == "" {
				mt = "untyped"
			}
			if mt != typeFilter {
				continue
			}
		}
		if projectFilter != "" && m.Project != projectFilter {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

func sortMemories(memories []memoryFile) {
	sort.Slice(memories, func(i, j int) bool {
		ti, tj := memoryTypeKey(memories[i]), memoryTypeKey(memories[j])
		if ti != tj {
			return ti < tj
		}
		if memories[i].Project != memories[j].Project {
			return memories[i].Project < memories[j].Project
		}
		return displayName(memories[i]) < displayName(memories[j])
	})
}

func memoryTypeKey(m memoryFile) int {
	t := m.Meta.Type
	for i, o := range typeOrder {
		if o == t {
			return i
		}
	}
	if t == "" {
		return len(typeOrder) + 1
	}
	return len(typeOrder)
}

// typeOrder defines the display order for memory types.
var typeOrder = []string{"user", "feedback", "project", "reference"}

// validTypeFilter returns an error if the type filter is non-empty and not a
// recognized memory type (including "untyped").
func validTypeFilter(t string) error {
	if t == "" || t == "untyped" {
		return nil
	}
	if !slices.Contains(typeOrder, t) {
		return fmt.Errorf("unknown memory type %q; valid types: %s, untyped", t, strings.Join(typeOrder, ", "))
	}
	return nil
}

func displayName(m memoryFile) string {
	if m.Meta.Name != "" {
		return m.Meta.Name
	}
	return m.File
}

func newBrowseCmd() *cobra.Command {
	var typeFilter string
	var projectFilter string
	var allProjects bool
	var plain bool

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse memories for the current project, grouped by type",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validTypeFilter(typeFilter); err != nil {
				return err
			}
			if plain {
				return cmdBrowsePlain(cmd.OutOrStdout(), typeFilter, projectFilter, allProjects)
			}
			return cmdBrowseTUI(cmd.OutOrStdout(), typeFilter, projectFilter, allProjects)
		},
	}

	cmd.Flags().StringVarP(&typeFilter, "type", "t", "", "filter by memory type (user, feedback, project, reference)")
	cmd.Flags().StringVarP(&projectFilter, "project", "p", "", "filter by project slug")
	cmd.Flags().BoolVarP(&allProjects, "all", "a", false, "show memories from all projects (default: current project only)")
	cmd.Flags().BoolVar(&plain, "plain", false, "plain text output (no TUI)")

	return cmd
}

// resolveProjectFilter returns the slug to filter by based on flags and cwd.
// If --project is set explicitly, that wins (after normalization/validation).
// If --all is set, no filter. Otherwise, tries to auto-detect the current
// project; falls back to all on failure.
func resolveProjectFilter(dotmemDir, projectFilter string, allProjects bool) (string, error) {
	if projectFilter != "" {
		normalized := normalizeSlug(projectFilter)
		if err := validateSlug(normalized); err != nil {
			return "", fmt.Errorf("invalid project slug %q: %w", projectFilter, err)
		}
		info, err := os.Stat(filepath.Join(dotmemDir, normalized))
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("project not found: %s", normalized)
			}
			return "", fmt.Errorf("stat project: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("project not found: %s", normalized)
		}
		return normalized, nil
	}
	if allProjects {
		return "", nil
	}
	slug, err := resolveSlug(dotmemDir)
	if err != nil {
		return "", nil
	}
	return slug, nil
}

func cmdBrowsePlain(w io.Writer, typeFilter, projectFilter string, allProjects bool) error {
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

	// Group by type (entries within each group are already sorted by
	// sortMemories: type key, then project, then display name).
	groups := make(map[string][]memoryFile)
	for _, m := range memories {
		t := m.Meta.Type
		if t == "" {
			t = "untyped"
		}
		groups[t] = append(groups[t], m)
	}

	// Print groups in fixed order, then any unknown types, then untyped.
	printed := 0
	seen := make(map[string]bool)
	for _, t := range typeOrder {
		seen[t] = true
		if entries, ok := groups[t]; ok {
			printGroup(w, t, entries, printed > 0)
			printed++
		}
	}
	var unknown []string
	for t := range groups {
		if !seen[t] && t != "untyped" {
			unknown = append(unknown, t)
		}
	}
	sort.Strings(unknown)
	for _, t := range unknown {
		printGroup(w, t, groups[t], printed > 0)
		printed++
	}
	if entries, ok := groups["untyped"]; ok {
		printGroup(w, "untyped", entries, printed > 0)
		printed++
	}

	if printed == 0 {
		fmt.Fprintln(w, "no memories found")
	}

	return nil
}

func printGroup(w io.Writer, typeName string, entries []memoryFile, needBlank bool) {
	if needBlank {
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "%s (%d)\n", typeName, len(entries))
	for _, m := range entries {
		fmt.Fprintf(w, "  %-14s %s\n", m.Project, displayName(m))
	}
}
