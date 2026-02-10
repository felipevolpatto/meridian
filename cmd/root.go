package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "meridian",
	Short: "A mock API server based on OpenAPI specifications",
	Long: `Meridian is a powerful mock server that automatically generates realistic test data
based on your OpenAPI 3.0 specification. It provides a persistent state management system,
relationship handling between resources, and a modern web interface for inspecting and testing your API.`,
}

func Execute() error {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	return rootCmd.Execute()
}
