package main

import (
	"context"
	"os"

	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/root"
)

func main() {
	rootCmd := root.Cmd()
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
