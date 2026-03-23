package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cd [slug]",
		Short: "Open a subshell in a project or memory directory",
		Long: `Open a subshell in the project directory (resolved from .path) or the
central memory repo if no slug is given. Exit the subshell to return.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) > 0 {
				slug = args[0]
			}
			return cmdCd(slug)
		},
	}
}

func cmdCd(slug string) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return fmt.Errorf("not initialized. Run \"dotmem init\" first.")
	}

	target := dir
	if slug != "" {
		projectDir := filepath.Join(dir, slug)
		pathFile := filepath.Join(projectDir, ".path")
		data, err := os.ReadFile(pathFile)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("project %q not found or has no .path file", slug)
			}
			return err
		}
		target = strings.TrimSpace(string(data))
		if _, err := os.Stat(target); err != nil {
			return fmt.Errorf("project path %s no longer exists", target)
		}
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell)
	cmd.Dir = target
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
