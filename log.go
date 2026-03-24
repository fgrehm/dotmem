package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "log [slug]",
		Short: "Show memory change history for a project",
		Long: `Show the git commit history of memory changes for a project.

If slug is omitted, auto-detects from the current project directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) > 0 {
				slug = args[0]
			}
			return cmdLog(cmd.OutOrStdout(), slug)
		},
	}
}

func cmdLog(w io.Writer, slug string) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if err := requireInit(dir); err != nil {
		return err
	}

	if slug == "" {
		resolved, err := resolveSlug(dir)
		if err != nil {
			return err
		}
		slug = resolved
	} else {
		slug = normalizeSlug(slug)
		if err := validateSlug(slug); err != nil {
			return err
		}
	}

	projectDir := filepath.Join(dir, slug)
	if _, err := os.Stat(projectDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project %q not found in %s", slug, dir)
		}
		return fmt.Errorf("failed to stat project %q in %s: %w", slug, dir, err)
	}

	out, err := gitExec(dir, "log", "--oneline", "--", slug+"/")
	if err != nil {
		return err
	}

	if out == "" {
		fmt.Fprintf(w, "dotmem: no history for %q\n", slug)
		return nil
	}

	fmt.Fprintln(w, out)
	return nil
}
