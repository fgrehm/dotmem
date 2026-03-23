package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func dotmemDir() (string, error) {
	dir := os.Getenv("DOTMEM_DIR")
	if dir != "" {
		if !filepath.IsAbs(dir) {
			return "", fmt.Errorf("DOTMEM_DIR must be an absolute path (got %q)", dir)
		}
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".mem"), nil
}

func mainWorktree(repoDir string) (string, error) {
	out, err := gitExec(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			return strings.TrimPrefix(line, "worktree "), nil
		}
	}
	return "", fmt.Errorf("could not determine main worktree")
}

func resolveSlug(dotmemDir string) (string, error) {
	toplevel, err := gitExec(".", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	canonical, err := mainWorktree(toplevel)
	if err != nil {
		canonical = toplevel
	}

	entries, err := os.ReadDir(dotmemDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".git" {
			continue
		}
		pathFile := filepath.Join(dotmemDir, e.Name(), ".path")
		data, err := os.ReadFile(pathFile)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == canonical {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("no linked project found for %s. Run \"dotmem link\" first.", canonical)
}

func gitExec(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func confirmPrompt(w io.Writer, r io.Reader, prompt string) error {
	if !isTerminal(r) {
		return fmt.Errorf("aborted (non-interactive)")
	}
	fmt.Fprint(w, prompt)
	var answer string
	fmt.Fscan(r, &answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		return fmt.Errorf("aborted")
	}
	return nil
}

func readJSONSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		content = "{}"
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(content), &settings); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return settings, nil
}

func writeJSONSettings(path string, settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
