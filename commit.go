package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commit",
		Short: "Auto-commit changed memory files (used by hooks)",
		Long: `Auto-commit any changed memory files in the dotmem repo. Designed for
use as a Claude Code Stop hook. Always exits 0, even on errors.
Set DOTMEM_DEBUG=1 for verbose output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdCommit(cmd.ErrOrStderr())
		},
	}
}

func cmdCommit(w io.Writer) error {
	debug := os.Getenv("DOTMEM_DEBUG") == "1"

	dir, err := dotmemDir()
	if err != nil {
		if debug {
			fmt.Fprintf(w, "dotmem: debug: %s\n", err)
		}
		return nil
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		if debug {
			fmt.Fprintf(w, "dotmem: debug: not initialized, skipping\n")
		}
		return nil
	}

	if debug {
		fmt.Fprintf(w, "dotmem: debug: running git add -A in %s\n", dir)
	}

	if _, err := gitExec(dir, "add", "-A"); err != nil {
		if debug {
			fmt.Fprintf(w, "dotmem: debug: git add failed: %s\n", err)
		}
		return nil
	}

	if _, err := gitExec(dir, "diff", "--cached", "--quiet"); err == nil {
		if debug {
			fmt.Fprintf(w, "dotmem: debug: no changes to commit\n")
		}
		return nil
	}

	msg := fmt.Sprintf("auto: %s", time.Now().UTC().Format("2006-01-02 15:04:05"))
	if _, err := gitExec(dir, "commit", "-m", msg); err != nil {
		if debug {
			fmt.Fprintf(w, "dotmem: debug: git commit failed: %s\n", err)
		}
		return nil
	}

	if debug {
		fmt.Fprintf(w, "dotmem: debug: committed %q\n", msg)
	}
	return nil
}
