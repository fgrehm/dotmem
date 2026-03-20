package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dotmem",
		Short: "Centralize Claude Code memory files into a single git-tracked repo",
		Long: `dotmem centralizes Claude Code memory files from all projects into a
single git repo with automatic versioning via Stop hooks.`,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version, commit, buildTime),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetVersionTemplate("dotmem v{{.Version}}\n")

	cmd.AddCommand(
		newInitCmd(),
		newLinkCmd(),
		newCommitCmd(),
		newCompactCmd(),
		newInstallHookCmd(),
		newStatusCmd(),
	)

	return cmd
}
