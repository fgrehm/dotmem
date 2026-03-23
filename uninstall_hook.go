package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUninstallHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-hook",
		Short: "Remove the Stop hook from Claude Code settings",
		Long: `Remove the "dotmem commit" Stop hook from ~/.claude/settings.json.
Idempotent: succeeds even if the hook is not installed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdUninstallHook(cmd.OutOrStdout())
		},
	}
}

func cmdUninstallHook(w io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readJSONSettings(settingsPath)
	if err != nil {
		return err
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		fmt.Fprintf(w, "dotmem: hook not installed\n")
		return nil
	}

	stopHooks, _ := hooks["Stop"].([]any)
	if len(stopHooks) == 0 {
		fmt.Fprintf(w, "dotmem: hook not installed\n")
		return nil
	}

	var filtered []any
	found := false
	for _, entry := range stopHooks {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		innerHooks, _ := entryMap["hooks"].([]any)
		var kept []any
		for _, h := range innerHooks {
			hMap, ok := h.(map[string]any)
			if !ok {
				kept = append(kept, h)
				continue
			}
			if hMap["command"] == hookCommand {
				found = true
				continue
			}
			kept = append(kept, h)
		}
		if len(kept) > 0 {
			entryMap["hooks"] = kept
			filtered = append(filtered, entryMap)
		}
	}

	if !found {
		fmt.Fprintf(w, "dotmem: hook not installed\n")
		return nil
	}

	if len(filtered) == 0 {
		delete(hooks, "Stop")
	} else {
		hooks["Stop"] = filtered
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}

	if err := writeJSONSettings(settingsPath, settings); err != nil {
		return err
	}

	fmt.Fprintf(w, "dotmem: hook removed from %s\n", settingsPath)
	return nil
}
