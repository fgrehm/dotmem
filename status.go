package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List linked projects",
		Long:  "List all projects linked to the central dotmem repo with last-modified dates.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdStatus(cmd.OutOrStdout())
		},
	}
}

func cmdStatus(w io.Writer) error {
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
		n := countMemoryFiles(filepath.Join(dir, e.Name()))
		unit := "files"
		if n == 1 {
			unit = "file"
		}
		fmt.Fprintf(w, "%-20s %2d %-5s  %s\n", e.Name(), n, unit, info.ModTime().Format("2006-01-02"))
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
		if !e.IsDir() && e.Name() != ".repo" {
			n++
		}
	}
	return n
}
