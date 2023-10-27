package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Show information about the current binary build",
	Args:  cobra.NoArgs,
	Run:   printBuildInfo,
}

func printBuildInfo(_ *cobra.Command, _ []string) {
	buildInfo := GetBuildInfo()
	fmt.Printf("Semantic version: %s\n", buildInfo.SemanticVersion)
	fmt.Printf("Commit: %s\n", buildInfo.LastCommit)
	fmt.Printf("Build Date: %s\n", buildInfo.BuildTime)
	fmt.Printf("System version: %s\n", buildInfo.SystemVersion)
	fmt.Printf("Golang version: %s\n", buildInfo.GolangVersion)
}
