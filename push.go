package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the memory repo to its remote",
		Long:  "Run git push in the central dotmem repo. Fails if no remote is configured.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdPush(cmd.OutOrStdout())
		},
	}
}

func cmdPush(w io.Writer) error {
	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if err := requireInit(dir); err != nil {
		return err
	}

	if _, err := gitExec(dir, "remote", "get-url", "origin"); err != nil {
		return fmt.Errorf("no remote configured; run \"git -C %s remote add origin <url>\" first: %w", dir, err)
	}

	if _, err := gitExec(dir, "push", "-u", "origin", "HEAD"); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Fprintf(w, "dotmem: pushed to remote\n")
	return nil
}
