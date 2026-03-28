package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the central memory repo",
		Long:  "Create ~/.mem as a git repo for storing Claude Code memory files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdInit(cmd.OutOrStdout())
		},
	}
}

func cmdInit(w io.Writer) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("already initialized at %s", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".DS_Store\n*.swp\n*~\n**/.path\n"), 0o644); err != nil {
		return err
	}

	readme := "# dotmem\nCentralized AI memory. Managed by [dotmem](https://github.com/fgrehm/dotmem).\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		return err
	}

	if _, err := gitExec(dir, "init"); err != nil {
		return err
	}
	if _, err := gitExec(dir, "add", "-A"); err != nil {
		return err
	}
	if _, err := gitExec(dir, "commit", "-m", "init: create dotmem repo"); err != nil {
		return err
	}

	fmt.Fprintf(w, "dotmem: initialized at %s\n", dir)
	fmt.Fprintf(w, "dotmem: next, run \"dotmem install-hook\" to register the Claude Code hook\n")
	fmt.Fprintf(w, "        then \"dotmem link\" inside each project\n")
	return nil
}
