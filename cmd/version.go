// Package cmd provides CLI commands for seedcli
package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	Version   = "2.0.0"
	BuildDate = "2026-01-26"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display the version of seedcli along with build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("seedcli version %s\n", Version)
		fmt.Printf("  Build date: %s\n", BuildDate)
		fmt.Printf("  Go version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println()
		fmt.Println("Supported adapters:")
		fmt.Println("  • postgres  - PostgreSQL 12+")
		fmt.Println("  • sqlite    - SQLite 3")
		fmt.Println()
		fmt.Println("For more information: https://github.com/kiridharan/seedcli")
	},
}
