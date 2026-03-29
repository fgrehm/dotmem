package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version and build information",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dotmem version %s\n", version)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", commit)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  built:  %s\n", date)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  go:     %s\n", runtime.Version())
		},
	}
}
