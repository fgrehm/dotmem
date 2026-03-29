package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dotmem",
		Short: "Centralize Claude Code memory files into a single git-tracked repo",
		Long: `dotmem centralizes Claude Code memory files from all projects into a
single git repo with automatic versioning via Stop hooks.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetVersionTemplate("dotmem version {{.Version}}\n")

	cmd.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newLinkCmd(),
		newUnlinkCmd(),
		newCommitCmd(),
		newCompactCmd(),
		newInstallHookCmd(),
		newUninstallHookCmd(),
		newLsCmd(),
		newLogCmd(),
		newPushCmd(),
		newCdCmd(),
	)

	return cmd
}
