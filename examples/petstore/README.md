# Petstore example

This example demonstrates a simple pet store API with owners and pets.

## Running the example

```bash
cd examples/petstore
meridian start
```

## Testing the API

### List all pets

```bash
curl http://localhost:8080/pets
```

### Create a new pet

```bash
curl -X POST http://localhost:8080/pets \
  -H "Content-Type: application/json" \
  -d '{"name": "Max", "species": "dog", "breed": "Labrador", "age": 2}'
```

### Get a specific pet

```bash
curl http://localhost:8080/pets/pet-001
```

### Update a pet

```bash
curl -X PUT http://localhost:8080/pets/pet-001 \
  -H "Content-Type: application/json" \
  -d '{"name": "Buddy Jr", "species": "dog", "breed": "Golden Retriever", "age": 4}'
```

### Delete a pet

```bash
curl -X DELETE http://localhost:8080/pets/pet-001
```

## Files

- [openapi.yaml](openapi.yaml) - OpenAPI specification
- [meridian.yaml](meridian.yaml) - Meridian configuration
- [seed.json](seed.json) - Initial seed data
