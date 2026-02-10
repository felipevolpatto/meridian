# Meridian

Meridian is a mock server that generates realistic API responses based on OpenAPI 3.0 specifications. It provides persistent state management, resource relationships, request/response validation, and a web interface for inspection and testing.

## Table of contents

- [Features](#features)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Middleware](#middleware)
- [CLI reference](#cli-reference)
- [API endpoints](#api-endpoints)
- [Examples](#examples)
- [Data generation](#data-generation)
- [Validation](#validation)
- [Web interface](#web-interface)
- [Development](#development)
- [Project structure](#project-structure)
- [License](#license)

## Features

### Implemented

- **OpenAPI 3.0 support**: parses your specification and generates mock responses
- **Persistent state**: SQLite-based storage maintains data across restarts
- **CRUD operations**: automatic handling of create, read, update, and delete
- **Resource relationships**: maintains referential integrity between resources
- **Request validation**: validates incoming requests against the specification
- **Response validation**: validates responses conform to defined schemas
- **Seed data**: initialize the server with predefined data
- **Rate limiting**: configurable request limits per client or globally
- **Error simulation**: simulate random errors for resilience testing
- **Response caching**: ETag support and configurable cache TTL
- **Gzip compression**: automatic response compression
- **CORS support**: configurable cross-origin resource sharing
- **Latency simulation**: add artificial delays to responses
- **Web interface**: built-in UI for state inspection and API testing
- **Advanced data generation**: pattern-based generation (regex), `oneOf`/`anyOf`/`allOf` support, semantic field detection

### In development

- **Nested resources**: support for routes like `/users/{id}/posts`
- **Auto seeding with relationships**: automatic seed data generation respecting referential integrity
- **Hot reload**: `--watch` flag for automatic server restart on file changes
- **WebSocket support**: real-time state updates

## Installation

### Using Go

```bash
go install github.com/felipevolpatto/meridian@latest
```

### From source

```bash
git clone https://github.com/felipevolpatto/meridian.git
cd meridian
go build -o meridian .
```

### Verify installation

```bash
meridian --version
```

## Quick start

### 1. Initialize a new project

```bash
meridian init
```

This creates the following files:

- `meridian.yaml` - configuration file
- `openapi.yaml` - sample OpenAPI specification
- `seed.json` - sample seed data

### 2. Start the server

```bash
meridian start
```

### 3. Test the API

```bash
# List users
curl http://localhost:8080/users

# Create a user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"id": "1", "name": "John Doe", "email": "john@example.com"}'

# Get a specific user
curl http://localhost:8080/users/1

# Update a user
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"id": "1", "name": "Jane Doe", "email": "jane@example.com"}'

# Delete a user
curl -X DELETE http://localhost:8080/users/1
```

### 4. Open the web interface

Navigate to [http://localhost:8080/_meridian/](http://localhost:8080/_meridian/) to access the state inspector and API tester.

## Configuration

Create a `meridian.yaml` file in your project directory:

```yaml
# Path to OpenAPI specification
openapi: openapi.yaml

# Server settings
server:
  address: localhost
  port: 8080

# State management
state:
  persistence: meridian_state.db
  seed: seed.json
  max_items: 1000
  ttl: 24h

# Behavior settings
behavior:
  # Latency simulation
  latency:
    enabled: false
    min: 50
    max: 200

  # Error simulation
  errors:
    enabled: false
    rate: 0.01
    types:
      - internal
      - timeout
    status_codes:
      - 500
      - 503

  # CORS settings
  cors:
    enabled: true
    allowed_origins:
      - "*"
    allowed_methods:
      - GET
      - POST
      - PUT
      - PATCH
      - DELETE
      - OPTIONS
    allowed_headers:
      - Content-Type
      - Authorization
    allow_credentials: false
    max_age: 12h

  # Rate limiting
  rate_limit:
    enabled: false
    rate: 100/minute
    per_client: true

  # Response compression
  compression: true

  # Response caching
  caching:
    enabled: false
    ttl: 5m
    use_etag: true
    resources:
      - users
      - posts
```

### Environment variables

All configuration options can be overridden using environment variables with the `MERIDIAN_` prefix:

| Variable | Description |
|----------|-------------|
| `MERIDIAN_OPENAPI` | Path to OpenAPI specification |
| `MERIDIAN_SERVER_ADDRESS` | Server bind address |
| `MERIDIAN_SERVER_PORT` | Server port |
| `MERIDIAN_STATE_PERSISTENCE` | Database file path |
| `MERIDIAN_STATE_SEED` | Seed data file path |

## Middleware

Meridian includes several middleware components that can be enabled via configuration.

### Rate limiting

Limits the number of requests per time window. Supports per-client (by IP) or global limiting.

```yaml
behavior:
  rate_limit:
    enabled: true
    rate: 100/minute    # Format: count/second|minute|hour
    per_client: true    # Limit per IP address
```

Response headers when rate limiting is enabled:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed |
| `X-RateLimit-Remaining` | Requests remaining in window |
| `X-RateLimit-Reset` | Unix timestamp when limit resets |
| `Retry-After` | Seconds until retry (when limited) |

When the limit is exceeded, returns `429 Too Many Requests`:

```json
{
  "error": "Rate limit exceeded",
  "code": "rate_limit_exceeded",
  "retry_after": 45
}
```

### Error simulation

Simulates random errors for testing client resilience.

```yaml
behavior:
  errors:
    enabled: true
    rate: 0.1           # 10% of requests will fail
    types:
      - internal
      - timeout
      - validation
    status_codes:
      - 500
      - 503
      - 504
```

Simulated error response:

```json
{
  "error": "Simulated internal error",
  "code": "simulated_error",
  "type": "internal",
  "simulated": true
}
```

### Response caching

Caches GET responses with ETag support for conditional requests.

```yaml
behavior:
  caching:
    enabled: true
    ttl: 5m             # Cache duration
    use_etag: true      # Enable ETag headers
    resources:          # Resources to cache (empty = all)
      - users
      - posts
```

Response headers:

| Header | Description |
|--------|-------------|
| `ETag` | Hash of response content |
| `Cache-Control` | Cache directives with max-age |

Supports `If-None-Match` header for conditional requests, returning `304 Not Modified` when content hasn't changed.

### Compression

Automatically compresses responses using gzip when the client supports it.

```yaml
behavior:
  compression: true
```

Response headers when compression is applied:

| Header | Value |
|--------|-------|
| `Content-Encoding` | gzip |
| `Vary` | Accept-Encoding |

### CORS

Configures Cross-Origin Resource Sharing headers.

```yaml
behavior:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "https://myapp.com"
    allowed_methods:
      - GET
      - POST
      - PUT
      - DELETE
    allowed_headers:
      - Content-Type
      - Authorization
    allow_credentials: true
    max_age: 12h
```

### Latency simulation

Adds artificial delay to responses for testing slow network conditions.

```yaml
behavior:
  latency:
    enabled: true
    min: 100            # Minimum delay in milliseconds
    max: 500            # Maximum delay in milliseconds
```

### Middleware order

Middleware is applied in the following order:

1. **CORS** - handles preflight requests first
2. **Latency** - delays before processing
3. **Error simulation** - may short-circuit request
4. **Rate limiting** - may reject request
5. **Caching** - may return cached response
6. **Compression** - compresses final response

## CLI reference

### start

Start the mock server.

```bash
meridian start [flags]
```

Flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Configuration file (default: `meridian.yaml`) |
| `--port` | `-p` | Server port |
| `--host` | | Server host address |
| `--spec` | `-s` | OpenAPI specification file |
| `--reset` | | Reset state before starting |
| `--no-seed` | | Skip loading seed data |

Examples:

```bash
# Start with default configuration
meridian start

# Start on a different port
meridian start -p 3000

# Start with a specific spec file
meridian start -s api/openapi.yaml

# Start with fresh state
meridian start --reset
```

### validate

Validate requests and responses against an OpenAPI specification.

```bash
meridian validate [flags]
```

Flags:

| Flag | Description |
|------|-------------|
| `--spec` | OpenAPI specification file (required) |
| `--request` | Request JSON file to validate |
| `--response` | Response JSON file to validate |
| `--verbose` | Show detailed validation output |

Examples:

```bash
# Validate a request
meridian validate --spec openapi.yaml --request request.json

# Validate a response
meridian validate --spec openapi.yaml --response response.json

# Validate both with verbose output
meridian validate --spec openapi.yaml \
  --request request.json \
  --response response.json \
  --verbose
```

### check

Validate an OpenAPI specification file.

```bash
meridian check <spec-file>
```

Examples:

```bash
# Check a local file
meridian check openapi.yaml

# Check a remote specification
meridian check https://api.example.com/openapi.yaml
```

### generate

Generate sample data based on a schema.

```bash
meridian generate <resource> [flags]
```

Flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--spec` | `-s` | OpenAPI specification file |
| `--count` | `-n` | Number of items to generate (default: 1) |
| `--output` | `-o` | Output format: `json` or `yaml` |

Examples:

```bash
# Generate a single user
meridian generate users -s openapi.yaml

# Generate multiple items
meridian generate products -s openapi.yaml -n 5

# Output as YAML
meridian generate orders -s openapi.yaml -o yaml
```

### export

Export current state to a JSON file.

```bash
meridian export [flags]
```

Flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output file path |
| `--database` | `-d` | Database file to export from |

Examples:

```bash
# Export to stdout
meridian export

# Export to a file
meridian export -o backup.json

# Export from a specific database
meridian export -d production.db -o backup.json
```

### import

Import state from a JSON file.

```bash
meridian import <file> [flags]
```

Flags:

| Flag | Description |
|------|-------------|
| `--database` | Database file to import into |
| `--merge` | Merge with existing data instead of replacing |

Examples:

```bash
# Import and replace existing data
meridian import backup.json

# Import and merge with existing data
meridian import backup.json --merge
```

### reset

Reset the state database.

```bash
meridian reset [flags]
```

Flags:

| Flag | Description |
|------|-------------|
| `--database` | Database file to reset |
| `--force` | Skip confirmation prompt |

### init

Initialize a new Meridian project with sample files.

```bash
meridian init [flags]
```

Flags:

| Flag | Description |
|------|-------------|
| `--force` | Overwrite existing files |

## API endpoints

### Resource endpoints

Meridian automatically creates REST endpoints for each path defined in your OpenAPI specification:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/{resource}` | List all items |
| POST | `/{resource}` | Create a new item |
| GET | `/{resource}/{id}` | Get a specific item |
| PUT | `/{resource}/{id}` | Update an item |
| PATCH | `/{resource}/{id}` | Partially update an item |
| DELETE | `/{resource}/{id}` | Delete an item |

### Admin endpoints

| Endpoint | Description |
|----------|-------------|
| `/_meridian/` | Web interface |
| `/_meridian/status` | Server status and statistics |
| `/_meridian/state` | Current state as JSON |
| `/_meridian/spec` | OpenAPI specification |

## Examples

The [examples](examples/) directory contains complete working examples:

### Petstore

A simple pet store API demonstrating basic CRUD operations.

```bash
cd examples/petstore
meridian start
```

See [examples/petstore/README.md](examples/petstore/README.md) for details.

### E-commerce

A more complex example with products, customers, orders, and relationships.

```bash
cd examples/ecommerce
meridian start
```

See [examples/ecommerce/README.md](examples/ecommerce/README.md) for details.

### Validation

Examples of request and response validation.

```bash
# Validate a correct request
meridian validate --spec examples/validation/openapi.yaml \
  --request examples/validation/valid-request.json

# Validate an incorrect request
meridian validate --spec examples/validation/openapi.yaml \
  --request examples/validation/invalid-request.json
```

See [examples/validation/README.md](examples/validation/README.md) for details.

## Data generation

Meridian generates realistic mock data based on your OpenAPI schema with advanced features.

### Pattern-based generation

When a schema includes a `pattern` property, Meridian generates strings that match the regex:

```yaml
properties:
  sku:
    type: string
    pattern: "SKU-[A-Z]{3}-[0-9]{4}"
  phone:
    type: string
    pattern: "\\+1-\\d{3}-\\d{3}-\\d{4}"
```

Generated values:
- `sku`: `SKU-XYZ-1234`
- `phone`: `+1-555-123-4567`

Supported regex features:
- Character classes: `[a-z]`, `[A-Z]`, `[0-9]`, `[^abc]`
- Escape sequences: `\d`, `\w`, `\s`, `\D`, `\W`, `\S`
- Quantifiers: `*`, `+`, `?`, `{n}`, `{n,m}`
- Groups and alternation: `(a|b|c)`
- Literals and escaped characters

### Semantic field detection

Meridian automatically detects field semantics based on naming and generates appropriate data:

| Field pattern | Generated data |
|---------------|----------------|
| `email`, `email_address` | Valid email address |
| `first_name`, `firstName` | Realistic first name |
| `last_name`, `lastName` | Realistic last name |
| `phone`, `phone_number` | Phone number |
| `address`, `street` | Street address |
| `city`, `state`, `country` | Location data |
| `zip_code`, `postal_code` | Postal code |
| `url`, `website` | Valid URL |
| `username`, `login` | Username |
| `price`, `amount` | Monetary value |
| `age` | Age between 18-80 |
| `created_at`, `updated_at` | ISO 8601 timestamp |
| `avatar`, `image` | Image URL |
| `latitude`, `longitude` | Geographic coordinates |
| `currency` | Currency code (USD, EUR, etc.) |
| `ip_address` | IPv4 address |
| `sku`, `product_code` | SKU format |

### Schema composition

Meridian supports OpenAPI schema composition keywords:

**oneOf**: Randomly selects one schema from the list:

```yaml
pet:
  oneOf:
    - $ref: '#/components/schemas/Cat'
    - $ref: '#/components/schemas/Dog'
```

**anyOf**: Randomly selects one schema (similar to oneOf for generation):

```yaml
notification:
  anyOf:
    - $ref: '#/components/schemas/EmailNotification'
    - $ref: '#/components/schemas/SMSNotification'
```

**allOf**: Merges all schemas into a single object:

```yaml
employee:
  allOf:
    - $ref: '#/components/schemas/Person'
    - type: object
      properties:
        employeeId:
          type: string
        department:
          type: string
```

## Validation

Meridian validates requests and responses against your OpenAPI specification.

### Request validation

The following constraints are validated:

- Required fields
- Field types (string, integer, number, boolean, array, object)
- String formats (email, uuid, date, date-time, uri)
- String constraints (minLength, maxLength, pattern)
- Numeric constraints (minimum, maximum, multipleOf)
- Array constraints (minItems, maxItems, uniqueItems)
- Enum values
- Required headers and query parameters

### Response validation

Response validation includes:

- Status code matching
- Content-Type header
- Response body schema
- Required response headers

### Request file format

```json
{
  "method": "POST",
  "path": "/users",
  "headers": {
    "Content-Type": "application/json",
    "Authorization": "Bearer token"
  },
  "query": "limit=10&offset=0",
  "body": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

### Response file format

```json
{
  "status_code": 200,
  "headers": {
    "Content-Type": "application/json",
    "X-Request-Id": "abc123"
  },
  "body": {
    "id": "1",
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

## Web interface

The web interface is available at `/_meridian/` and provides:

### State inspector

- View all resources and their current state
- Inspect individual items
- Monitor changes in real-time

### API tester

- Send requests to any endpoint
- Automatic request body generation
- View response timing and headers

## Development

### Prerequisites

- Go 1.21 or later
- Node.js 18 or later (for web UI development)

### Building from source

```bash
git clone https://github.com/felipevolpatto/meridian.git
cd meridian
go build -o meridian .
```

### Running tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with verbose output
go test ./... -v
```

### Building the web interface

```bash
cd web
npm install
npm run build
```

## Project structure

```
meridian/
├── cmd/                    # CLI commands
│   ├── check.go           # Specification validation
│   ├── export.go          # State export
│   ├── generate.go        # Data generation
│   ├── import.go          # State import
│   ├── init.go            # Project initialization
│   ├── reset.go           # State reset
│   ├── root.go            # Root command
│   ├── start.go           # Server start
│   └── validate.go        # Request/response validation
├── internal/
│   ├── cli/               # CLI utilities
│   ├── config/            # Configuration handling
│   ├── generator/         # Data generation
│   ├── openapi/           # OpenAPI parsing
│   ├── server/            # HTTP server and middleware
│   ├── state/             # State management
│   └── validation/        # Request/response validation
├── examples/              # Example projects
│   ├── petstore/          # Simple pet store API
│   ├── ecommerce/         # E-commerce API
│   └── validation/        # Validation examples
├── web/                   # Web interface (Svelte)
├── docs/                  # Documentation
├── go.mod                 # Go module file
├── go.sum                 # Go dependencies
├── main.go                # Entry point
└── README.md              # This file
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [kin-openapi](https://github.com/getkin/kin-openapi) - OpenAPI parsing and validation
- [faker](https://github.com/jaswdr/faker) - Realistic data generation
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Svelte](https://svelte.dev/) - Web interface framework
