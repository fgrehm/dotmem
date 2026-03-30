package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// typeOrder defines the display order for memory types.
var typeOrder = []string{"user", "feedback", "project", "reference"}

func newBrowseCmd() *cobra.Command {
	var typeFilter string
	var projectFilter string

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse memories across all projects, grouped by type",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdBrowse(cmd.OutOrStdout(), typeFilter, projectFilter)
		},
	}

	cmd.Flags().StringVarP(&typeFilter, "type", "t", "", "filter by memory type (user, feedback, project, reference)")
	cmd.Flags().StringVarP(&projectFilter, "project", "p", "", "filter by project slug")

	return cmd
}

func cmdBrowse(w io.Writer, typeFilter, projectFilter string) error {
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

	// Apply filters.
	if typeFilter != "" || projectFilter != "" {
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
		memories = filtered
	}

	// Group by type.
	groups := make(map[string][]memoryFile)
	for _, m := range memories {
		t := m.Meta.Type
		if t == "" {
			t = "untyped"
		}
		groups[t] = append(groups[t], m)
	}

	// Sort entries within each group by project, then display name.
	for _, entries := range groups {
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Project != entries[j].Project {
				return entries[i].Project < entries[j].Project
			}
			return displayName(entries[i]) < displayName(entries[j])
		})
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
	// Unknown types (alphabetical).
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
	// Untyped last.
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

func displayName(m memoryFile) string {
	if m.Meta.Name != "" {
		return m.Meta.Name
	}
	return m.File
}
