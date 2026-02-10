package cmd

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate OpenAPI specification",
	Long:  `Validate the OpenAPI specification for correctness and best practices.`,
	RunE:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().StringP("spec", "s", "openapi.yaml", "Path to OpenAPI specification file")
	checkCmd.Flags().BoolP("strict", "t", false, "Enable strict validation mode")
	checkCmd.Flags().BoolP("verbose", "v", false, "Show detailed validation results")
}

func runCheck(cmd *cobra.Command, args []string) error {
	specPath, _ := cmd.Flags().GetString("spec")
	strict, _ := cmd.Flags().GetBool("strict")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if len(args) > 0 {
		specPath = args[0]
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		return fmt.Errorf("OpenAPI spec validation failed: %w", err)
	}

	if strict {
		if err := validateStrict(doc, verbose); err != nil {
			return err
		}
	}

	fmt.Println("✅ OpenAPI specification is valid")
	return nil
}

func validateStrict(doc *openapi3.T, verbose bool) error {
	var errors []string

	opIDs := make(map[string]string)
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op.OperationID == "" {
				errors = append(errors, fmt.Sprintf("Missing operation ID for %s %s", method, path))
				continue
			}
			if existingPath, exists := opIDs[op.OperationID]; exists {
				errors = append(errors, fmt.Sprintf("Duplicate operation ID '%s' found in:\n  - %s\n  - %s",
					op.OperationID, existingPath, path))
			}
			opIDs[op.OperationID] = path
		}
	}

	for path, item := range doc.Paths.Map() {
		for _, param := range item.Parameters {
			if param.Value.Description == "" {
				errors = append(errors, fmt.Sprintf("Missing parameter description for '%s' in path %s",
					param.Value.Name, path))
			}
		}

		for method, op := range item.Operations() {
			for _, param := range op.Parameters {
				if param.Value.Description == "" {
					errors = append(errors, fmt.Sprintf("Missing parameter description for '%s' in %s %s",
						param.Value.Name, method, path))
				}
			}
		}
	}

	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op.Responses != nil {
				for _, code := range []int{200, 201, 204, 400, 401, 403, 404, 500} {
					if response := op.Responses.Status(code); response != nil {
						if response.Value != nil && (response.Value.Description == nil || *response.Value.Description == "") {
							errors = append(errors, fmt.Sprintf("Missing response description for status %d in %s %s",
								code, method, path))
						}
					}
				}
				if defaultResponse := op.Responses.Default(); defaultResponse != nil {
					if defaultResponse.Value != nil && (defaultResponse.Value.Description == nil || *defaultResponse.Value.Description == "") {
						errors = append(errors, fmt.Sprintf("Missing default response description in %s %s",
							method, path))
					}
				}
			}
		}
	}

	if len(doc.Components.SecuritySchemes) == 0 {
		errors = append(errors, "No security schemes defined")
	}

	if len(doc.Tags) == 0 {
		errors = append(errors, "No tags defined")
	}

	if len(errors) > 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, "Strict validation errors:")
			for _, err := range errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", err)
			}
		}
		return fmt.Errorf("strict validation failed with %d errors", len(errors))
	}

	if verbose {
		fmt.Println("✅ Strict validation passed")
	}
	return nil
}
