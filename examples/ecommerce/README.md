# E-commerce example

This example demonstrates a complex e-commerce API with customers, products, and orders.

## Features demonstrated

- Resource relationships (orders belong to customers)
- Nested resources with full CRUD (`/customers/{customerId}/orders`)
- Pattern-based generation (SKU: `^[A-Z]{3}-[0-9]{6}$`)
- Semantic field detection (email, name, address, price)
- Query parameter filtering
- Simulated latency (50-200ms)

## Running the example

```bash
cd examples/ecommerce
meridian start
```

## Testing the API

### Products

```bash
# List all products
curl http://localhost:8080/products

# Filter by category
curl "http://localhost:8080/products?category=electronics"

# Filter by price range
curl "http://localhost:8080/products?min_price=20&max_price=50"

# Create a product (note: SKU must match pattern ^[A-Z]{3}-[0-9]{6}$)
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ergonomic Keyboard",
    "price": 89.99,
    "category": "electronics",
    "stock": 30,
    "sku": "ELE-000002"
  }'
```

### Orders (direct access)

```bash
# List all orders
curl http://localhost:8080/orders

# Filter by status
curl "http://localhost:8080/orders?status=pending"

# Create an order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-002",
    "items": [
      {"product_id": "prod-002", "quantity": 3}
    ],
    "shipping_address": {
      "street": "456 Oak Avenue",
      "city": "Los Angeles",
      "state": "CA",
      "postal_code": "90001",
      "country": "USA"
    }
  }'

# Update order status
curl -X PATCH http://localhost:8080/orders/ord-001 \
  -H "Content-Type: application/json" \
  -d '{"status": "shipped"}'
```

### Customers

```bash
# List all customers
curl http://localhost:8080/customers

# Create a customer
curl -X POST http://localhost:8080/customers \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Johnson", "email": "alice@example.com"}'

# Get customer details
curl http://localhost:8080/customers/cust-001
```

### Nested resources (customer orders)

```bash
# List orders for a specific customer
curl http://localhost:8080/customers/cust-001/orders

# Create an order for a customer (customer_id is set automatically)
curl -X POST http://localhost:8080/customers/cust-001/orders \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"product_id": "prod-001", "quantity": 2}
    ],
    "shipping_address": {
      "street": "123 Main St",
      "city": "New York",
      "postal_code": "10001"
    }
  }'

# Get a specific order for a customer
curl http://localhost:8080/customers/cust-001/orders/ord-001

# Update order status via nested route
curl -X PATCH http://localhost:8080/customers/cust-001/orders/ord-001 \
  -H "Content-Type: application/json" \
  -d '{"status": "delivered"}'

# Cancel an order
curl -X DELETE http://localhost:8080/customers/cust-001/orders/ord-001
```

## Files

- [openapi.yaml](openapi.yaml) - OpenAPI specification
- [meridian.yaml](meridian.yaml) - Meridian configuration
- [seed.json](seed.json) - Initial seed data
