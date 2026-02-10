# Examples

This directory contains working examples demonstrating Meridian features.

## Available examples

| Example | Description |
|---------|-------------|
| [petstore](petstore/) | Simple pet store API with basic CRUD operations |
| [ecommerce](ecommerce/) | E-commerce API with relationships and advanced features |
| [validation](validation/) | Request and response validation examples |

## Running an example

Each example directory contains:

- `openapi.yaml` - OpenAPI specification
- `meridian.yaml` - Meridian configuration
- `seed.json` - Initial seed data (where applicable)
- `README.md` - Example-specific documentation

To run an example:

```bash
cd examples/<example-name>
meridian start
```

## Petstore

A simple API demonstrating basic features:

- CRUD operations for pets and owners
- Query parameter filtering
- Basic data types and validation

```bash
cd examples/petstore
meridian start

# Test the API
curl http://localhost:8080/pets
curl http://localhost:8080/owners
```

## E-commerce

A more complex API demonstrating advanced features:

- Products, customers, and orders
- Resource relationships (orders reference customers and products)
- Nested resources (customer orders)
- Simulated latency
- SKU pattern validation

```bash
cd examples/ecommerce
meridian start

# Test the API
curl http://localhost:8080/products
curl http://localhost:8080/customers
curl http://localhost:8080/orders
curl http://localhost:8080/customers/cust-001/orders
```

## Validation

Examples for testing the validation command:

```bash
# Validate a correct request
meridian validate --spec examples/validation/openapi.yaml \
  --request examples/validation/valid-request.json

# Validate an incorrect request
meridian validate --spec examples/validation/openapi.yaml \
  --request examples/validation/invalid-request.json

# Validate a response
meridian validate --spec examples/validation/openapi.yaml \
  --response examples/validation/valid-response.json
```

## Creating your own example

1. Create a new directory under `examples/`
2. Add your OpenAPI specification as `openapi.yaml`
3. Create a `meridian.yaml` configuration file
4. Optionally add `seed.json` for initial data
5. Add a `README.md` with usage instructions
