package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	version = "0.1.0"
)

var (
	// Global flags
	profile        string
	region         string
	verbose        bool
	platformAPIURL string
)

// NewRootCommand creates the root command for rosactl
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "rosactl",
		Short: "ROSA Regional HCP CLI tool",
		Long: `rosactl is the command-line interface for ROSA Regional HCP platform.
It enables customers to provision and manage HyperShift clusters with AWS IAM authentication.`,
		Version: version,
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "AWS credential profile")
	rootCmd.PersistentFlags().StringVar(&region, "region", "", "AWS region")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&platformAPIURL, "platform-api-url", "", "Platform API endpoint URL")

	// Add subcommands
	rootCmd.AddCommand(NewInitCommand())
	rootCmd.AddCommand(NewSetupAccountCommand())

	return rootCmd
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getGlobalFlags returns the global flag values
func getGlobalFlags() (string, string, bool, string) {
	return profile, region, verbose, platformAPIURL
}
