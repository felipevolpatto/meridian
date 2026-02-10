package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felipevolpatto/meridian/internal/state"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export current state",
	Long:  `Export the current state of the mock server to a JSON file.`,
	RunE:  runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "export.json", "Output file path")
	exportCmd.Flags().BoolP("pretty", "p", true, "Pretty print JSON output")
	exportCmd.Flags().StringSliceP("resources", "r", []string{}, "Specific resources to export (empty means all)")
}

func runExport(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	pretty, _ := cmd.Flags().GetBool("pretty")
	resources, _ := cmd.Flags().GetStringSlice("resources")

	config, err := loadConfig("meridian.yaml")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	manager, err := state.New(config.State.Persistence)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	defer manager.Close()

	data, err := manager.Export()
	if err != nil {
		return fmt.Errorf("failed to export state: %w", err)
	}

	if len(resources) > 0 {
		filteredResources := make(map[string][]interface{})
		filteredRelations := make(map[string]map[string]string)
		filteredMetadata := make(map[string]map[string]interface{})

		for _, resource := range resources {
			if data, ok := data.Resources[resource]; ok {
				filteredResources[resource] = data
			}
			if relations, ok := data.Relations[resource]; ok {
				filteredRelations[resource] = relations
			}
			if metadata, ok := data.Metadata[resource]; ok {
				filteredMetadata[resource] = metadata
			}
		}

		data.Resources = filteredResources
		data.Relations = filteredRelations
		data.Metadata = filteredMetadata
	}

	if err := writeFileIfNotExists(output, data, pretty); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("✅ Exported state to %s\n", output)
	if len(resources) > 0 {
		fmt.Printf("   Resources: %v\n", resources)
	} else {
		fmt.Printf("   Resources: all\n")
	}
	return nil
}
