package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felipevolpatto/meridian/internal/state"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import state from a file",
	Long:  `Import state from a JSON file into the mock server.`,
	RunE:  runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringP("input", "i", "export.json", "Input file path")
	importCmd.Flags().BoolP("merge", "m", false, "Merge with existing state instead of replacing")
	importCmd.Flags().StringSliceP("resources", "r", []string{}, "Specific resources to import (empty means all)")
	importCmd.Flags().BoolP("dry-run", "d", false, "Show what would be imported without making changes")
}

func runImport(cmd *cobra.Command, args []string) error {
	input, _ := cmd.Flags().GetString("input")
	merge, _ := cmd.Flags().GetBool("merge")
	resources, _ := cmd.Flags().GetStringSlice("resources")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	config, err := loadConfig("meridian.yaml")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var importData state.ExportData
	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to parse import data: %w", err)
	}

	if len(resources) > 0 {
		filteredResources := make(map[string][]interface{})
		filteredRelations := make(map[string]map[string]string)
		filteredMetadata := make(map[string]map[string]interface{})

		for _, resource := range resources {
			if data, ok := importData.Resources[resource]; ok {
				filteredResources[resource] = data
			}
			if relations, ok := importData.Relations[resource]; ok {
				filteredRelations[resource] = relations
			}
			if metadata, ok := importData.Metadata[resource]; ok {
				filteredMetadata[resource] = metadata
			}
		}

		importData.Resources = filteredResources
		importData.Relations = filteredRelations
		importData.Metadata = filteredMetadata
	}

	if dryRun {
		return showImportPlan(&importData, merge)
	}

	manager, err := state.New(config.State.Persistence)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	defer manager.Close()

	if err := manager.Import(&importData, merge); err != nil {
		return fmt.Errorf("failed to import state: %w", err)
	}

	fmt.Println("✅ Successfully imported state")
	fmt.Printf("   Resources imported: %d\n", len(importData.Resources))
	if merge {
		fmt.Println("   Mode: merge")
	} else {
		fmt.Println("   Mode: replace")
	}
	return nil
}

func showImportPlan(data *state.ExportData, merge bool) error {
	fmt.Println("Import Plan (dry run):")
	fmt.Printf("Version: %s\n", data.Version)
	fmt.Printf("Mode: %s\n", map[bool]string{true: "merge", false: "replace"}[merge])
	fmt.Println("\nResources:")
	for resource, items := range data.Resources {
		fmt.Printf("  %s: %d items\n", resource, len(items))
		if relations, ok := data.Relations[resource]; ok {
			fmt.Printf("    Relations:\n")
			for target, relType := range relations {
				fmt.Printf("      - %s (%s)\n", target, relType)
			}
		}
		if metadata, ok := data.Metadata[resource]; ok {
			fmt.Printf("    Metadata:\n")
			for key, value := range metadata {
				fmt.Printf("      - %s: %v\n", key, value)
			}
		}
	}
	fmt.Println("\nNo changes were made (dry run)")
	return nil
}
