package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List linked projects",
		Long:  "List all projects linked to the central dotmem repo with last-modified dates.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdLs(cmd.OutOrStdout())
		},
	}
}

func cmdLs(w io.Writer) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return fmt.Errorf("not initialized. Run \"dotmem init\" first.")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, dir)
	fmt.Fprintln(w)

	found := false
	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".git" {
			continue
		}
		found = true
		info, err := e.Info()
		if err != nil {
			continue
		}
		projectDir := filepath.Join(dir, e.Name())
		n := countMemoryFiles(projectDir)
		unit := "files"
		if n == 1 {
			unit = "file"
		}
		pathStr := ""
		if data, err := os.ReadFile(filepath.Join(projectDir, ".path")); err == nil {
			pathStr = "  " + strings.TrimSpace(string(data))
		}
		fmt.Fprintf(w, "%-20s %2d %-5s  %s%s\n", e.Name(), n, unit, info.ModTime().Format("2006-01-02"), pathStr)
	}

	if !found {
		fmt.Fprintln(w, "no linked projects")
	}
	return nil
}

func countMemoryFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && e.Name() != ".repo" && e.Name() != ".path" {
			n++
		}
	}
	return n
}
