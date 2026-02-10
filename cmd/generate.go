package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/felipevolpatto/meridian/internal/generator"
	"github.com/felipevolpatto/meridian/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate [resource]",
	Short: "Generates example data for a resource",
	Long:  `Generates example data for a given resource based on the OpenAPI specification.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resource := args[0]

		cfg, err := config.Load("meridian.yaml")
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		spec, err := openapi.ParseFile(cfg.OpenAPI)
		if err != nil {
			log.Fatalf("Error parsing OpenAPI spec: %v", err)
		}

		var resourceSchema *openapi3.SchemaRef
		if s, ok := spec.Components.Schemas[resource]; ok {
			resourceSchema = s
		} else {
			for _, pathItem := range spec.Paths.Map() {
				if pathItem.Get != nil && pathItem.Get.Responses != nil {
					if resp, ok := pathItem.Get.Responses.Map()["200"]; ok && resp.Value.Content.Get("application/json") != nil {
						if resp.Value.Content.Get("application/json").Schema != nil {
							if resp.Value.Content.Get("application/json").Schema.Ref == "#/components/schemas/"+resource {
								resourceSchema = resp.Value.Content.Get("application/json").Schema
								break
							}
						}
					}
				}
			}
		}

		if resourceSchema == nil {
			fmt.Printf("Error: Schema for resource '%s' not found in OpenAPI spec.", resource)
			return
		}

		data, err := generator.GenerateData(resourceSchema)
		if err != nil {
			fmt.Printf("Error generating data: %v\n", err)
			return
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling data: %v", err)
			return
		}
		fmt.Println(string(jsonData))
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
