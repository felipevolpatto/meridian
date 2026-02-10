# Petstore example

This example demonstrates a pet store API with owners and pets, including nested resources.

## Features demonstrated

- Basic CRUD operations
- Nested resources (`/owners/{ownerId}/pets`)
- Pattern-based validation (phone number)
- Semantic field detection (email, name, address)
- Resource relationships (pet belongs to owner)

## Running the example

```bash
cd examples/petstore
meridian start
```

## Testing the API

### Pets

```bash
# List all pets
curl http://localhost:8080/pets

# Create a new pet
curl -X POST http://localhost:8080/pets \
  -H "Content-Type: application/json" \
  -d '{"name": "Max", "species": "dog", "breed": "Labrador", "age": 2}'

# Get a specific pet
curl http://localhost:8080/pets/pet-001

# Update a pet
curl -X PUT http://localhost:8080/pets/pet-001 \
  -H "Content-Type: application/json" \
  -d '{"name": "Buddy Jr", "species": "dog", "breed": "Golden Retriever", "age": 4}'

# Delete a pet
curl -X DELETE http://localhost:8080/pets/pet-001
```

### Owners

```bash
# List all owners
curl http://localhost:8080/owners

# Create an owner
curl -X POST http://localhost:8080/owners \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Smith", "email": "alice@example.com", "phone": "+15551234567"}'

# Get an owner
curl http://localhost:8080/owners/owner-001
```

### Nested resources (owner's pets)

```bash
# List pets for a specific owner
curl http://localhost:8080/owners/owner-001/pets

# Create a pet for an owner (owner_id is set automatically)
curl -X POST http://localhost:8080/owners/owner-001/pets \
  -H "Content-Type: application/json" \
  -d '{"name": "Fluffy", "species": "cat", "breed": "Persian", "age": 3}'

# Get a specific pet of an owner
curl http://localhost:8080/owners/owner-001/pets/pet-001

# Update an owner's pet
curl -X PUT http://localhost:8080/owners/owner-001/pets/pet-001 \
  -H "Content-Type: application/json" \
  -d '{"name": "Fluffy Jr", "species": "cat", "age": 4}'

# Delete an owner's pet
curl -X DELETE http://localhost:8080/owners/owner-001/pets/pet-001
```

## Files

- [openapi.yaml](openapi.yaml) - OpenAPI specification
- [meridian.yaml](meridian.yaml) - Meridian configuration
- [seed.json](seed.json) - Initial seed data
