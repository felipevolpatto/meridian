package cli

const (
	// HelpText is the main help text for the CLI
	HelpText = `Meridian - OpenAPI Mock Server

Usage:
  meridian [flags] [command]

Commands:
  start       Start the mock server
  validate    Validate OpenAPI specification
  generate    Generate example data
  help        Show help for any command

Global Flags:
  -c, --config string   Path to config file (default "meridian.yaml")
  -h, --help           Show help for command
  -v, --version        Show version information

Examples:
  # Start server with default configuration
  meridian start

  # Start server with custom configuration
  meridian start -c custom-config.yaml

  # Validate OpenAPI specification
  meridian validate openapi.yaml

  # Generate example data for a resource
  meridian generate users

For more information, visit: https://github.com/felipevolpatto/meridian`

	// StartHelpText is the help text for the start command
	StartHelpText = `Start the mock server

Usage:
  meridian start [flags]

Flags:
  -c, --config string   Path to config file (default "meridian.yaml")
  -p, --port int       Server port (overrides config) (default 8080)
  --host string        Server host (overrides config) (default "localhost")
  --no-seed           Disable automatic seeding on startup
  --reset             Reset state on startup
  -h, --help          Show help for command

Examples:
  # Start server with default configuration
  meridian start

  # Start server on custom port
  meridian start -p 3000

  # Start server with custom host and port
  meridian start --host 0.0.0.0 -p 8000

  # Start server without automatic seeding
  meridian start --no-seed

  # Start server with state reset
  meridian start --reset`

	// ValidateHelpText is the help text for the validate command
	ValidateHelpText = `Validate OpenAPI specification

Usage:
  meridian validate [flags] <spec-file>

Flags:
  -h, --help    Show help for command

Examples:
  # Validate OpenAPI specification
  meridian validate openapi.yaml

  # Validate OpenAPI specification from URL
  meridian validate https://example.com/openapi.yaml`

	// GenerateHelpText is the help text for the generate command
	GenerateHelpText = `Generate example data for resources

Usage:
  meridian generate [flags] <resource-name>

Flags:
  -n, --count int     Number of examples to generate (default 1)
  -o, --output string Output format (json|yaml) (default "json")
  -h, --help         Show help for command

Examples:
  # Generate single example
  meridian generate users

  # Generate multiple examples
  meridian generate users -n 5

  # Generate example in YAML format
  meridian generate users -o yaml`
)

// GetCommandHelp returns the help text for a specific command
func GetCommandHelp(command string) string {
	switch command {
	case "start":
		return StartHelpText
	case "validate":
		return ValidateHelpText
	case "generate":
		return GenerateHelpText
	default:
		return HelpText
	}
} 