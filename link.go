package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link [slug]",
		Short: "Link the current project to the memory repo",
		Long: `Link the current project to the central dotmem repo. The slug defaults
to the project directory name. Must be run inside a git repo with a
remote origin.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) > 0 {
				slug = args[0]
			}
			yes, _ := cmd.Flags().GetBool("yes")
			return cmdLink(cmd.OutOrStdout(), cmd.InOrStdin(), slug, yes)
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation prompts")
	return cmd
}

func cmdLink(w io.Writer, r io.Reader, slug string, force bool) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if err := requireInit(dir); err != nil {
		return err
	}

	toplevel, err := gitExec(".", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	remoteURL, err := gitExec(toplevel, "remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("no remote origin found; dotmem link requires a git remote named \"origin\"")
	}

	if slug == "" {
		slug = filepath.Base(toplevel)
	}
	slug = normalizeSlug(slug)
	if err := validateSlug(slug); err != nil {
		return err
	}

	projectDir := filepath.Join(dir, slug)
	repoFile := filepath.Join(projectDir, repoMarker)
	canonical := mainWorktree(toplevel)

	if _, err := os.Stat(projectDir); err == nil {
		existing, err := os.ReadFile(repoFile)
		if err == nil {
			existingURL := strings.TrimSpace(string(existing))
			if existingURL != remoteURL {
				return fmt.Errorf("%s already linked to %s (current: %s). Use: dotmem link <different-slug>", slug, existingURL, remoteURL)
			}
		}
	} else {
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(repoFile, []byte(remoteURL+"\n"), 0644); err != nil {
			return err
		}
		if _, err := gitExec(dir, "add", "-A"); err != nil {
			return err
		}
		if _, err := gitExec(dir, "commit", "-m", fmt.Sprintf("link: add %s", slug)); err != nil {
			return err
		}
	}

	pathFile := filepath.Join(projectDir, pathMarker)
	if err := os.WriteFile(pathFile, []byte(canonical+"\n"), 0644); err != nil {
		return err
	}
	gitignorePath := filepath.Join(dir, ".gitignore")
	if err := ensureGitignoreRule(gitignorePath, "**/.path"); err != nil {
		return err
	}
	statusOut, err := gitExec(dir, "status", "--porcelain", ".gitignore")
	if err != nil {
		return err
	}
	if strings.TrimSpace(statusOut) != "" {
		if _, err := gitExec(dir, "add", ".gitignore"); err != nil {
			return err
		}
		if _, err := gitExec(dir, "commit", "-m", "link: update .gitignore for legacy repos"); err != nil {
			return err
		}
	}

	claudeDir := filepath.Join(toplevel, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	targetPath := projectDir

	settings, err := readJSONSettings(settingsPath)
	if err != nil {
		return err
	}

	if existing, ok := settings["autoMemoryDirectory"].(string); ok {
		if existing == targetPath {
			fmt.Fprintf(w, "dotmem: already linked %q -> %s\n", slug, targetPath)
			return nil
		}
		if !force {
			fmt.Fprintf(w, "dotmem: autoMemoryDirectory is currently set to %q\n", existing)
			fmt.Fprintf(w, "dotmem: new value would be %q\n", targetPath)
			if err := confirmPrompt(w, r, "overwrite? [y/N] "); err != nil {
				return err
			}
		}
	}

	settings["autoMemoryDirectory"] = targetPath
	if err := writeJSONSettings(settingsPath, settings); err != nil {
		return err
	}

	fmt.Fprintf(w, "dotmem: linked %q -> %s\n", slug, targetPath)
	fmt.Fprintf(w, "hint: ensure .claude/settings.local.json is gitignored (globally or per-project)\n")
	return nil
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func normalizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSuffix(s, ".git")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}
