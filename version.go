package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version and build information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdVersion(cmd.OutOrStdout())
		},
	}
}

func cmdVersion(w io.Writer) error {
	_, err := fmt.Fprintf(w,
		"dotmem version %s\n  commit: %s\n  built:  %s\n  go:     %s\n",
		version, commit, date, runtime.Version(),
	)
	return err
}
