package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInstallHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-hook",
		Short: "Register the Stop hook in Claude Code settings",
		Long: `Register a Claude Code Stop hook that runs "dotmem commit" after every
session. The hook is added to ~/.claude/settings.json. Idempotent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdInstallHook(cmd.OutOrStdout())
		},
	}
}

func cmdInstallHook(w io.Writer) error {
	if _, err := exec.LookPath("dotmem"); err != nil {
		return fmt.Errorf("\"dotmem\" not found on PATH; install it first or add its location to your PATH")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	claudeDir := filepath.Join(home, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")

	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return err
	}

	settings, err := readJSONSettings(settingsPath)
	if err != nil {
		return err
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	stopHooks, _ := hooks["Stop"].([]any)

	for _, entry := range stopHooks {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		innerHooks, _ := entryMap["hooks"].([]any)
		for _, h := range innerHooks {
			hMap, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if hMap["command"] == hookCommand {
				fmt.Fprintf(w, "dotmem: hook already installed\n")
				return nil
			}
		}
	}

	newEntry := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": hookCommand,
			},
		},
	}
	stopHooks = append(stopHooks, newEntry)
	hooks["Stop"] = stopHooks
	settings["hooks"] = hooks

	if err := writeJSONSettings(settingsPath, settings); err != nil {
		return err
	}

	fmt.Fprintf(w, "dotmem: hook installed in %s\n", settingsPath)
	return nil
}
