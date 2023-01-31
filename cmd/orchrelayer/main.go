package main

import (
	"os"
	"path/filepath"

	"github.com/celestiaorg/orchestrator-relayer/cmd/orchrelayer/cmd"
)

const (
	Name = "orchrelayer"
)

var (
	DefaultNodeHome string
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}
func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
