package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "dotmem: %s\n", err)
		os.Exit(1)
	}
}
