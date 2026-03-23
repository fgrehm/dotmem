package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUnlinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink",
		Short: "Remove the memory link for the current project",
		Long: `Remove the autoMemoryDirectory setting from .claude/settings.local.json
for the current project. Does not delete memory files from the central repo.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdUnlink(cmd.OutOrStdout())
		},
	}
}

func cmdUnlink(w io.Writer) error {
	toplevel, err := gitExec(".", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	settingsPath := filepath.Join(toplevel, ".claude", "settings.local.json")
	settings, err := readJSONSettings(settingsPath)
	if err != nil {
		return err
	}

	if _, ok := settings["autoMemoryDirectory"]; !ok {
		fmt.Fprintf(w, "dotmem: not linked (no autoMemoryDirectory in settings)\n")
		return nil
	}

	delete(settings, "autoMemoryDirectory")

	if len(settings) == 0 {
		if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else {
		if err := writeJSONSettings(settingsPath, settings); err != nil {
			return err
		}
	}

	fmt.Fprintf(w, "dotmem: unlinked %s\n", filepath.Base(toplevel))
	return nil
}
