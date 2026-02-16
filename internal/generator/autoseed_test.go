package generator

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "users"},
		{"pet", "pets"},
		{"owner", "owners"},
		{"customer", "customers"},
		{"category", "categories"},
		{"company", "companies"},
		{"box", "boxes"},
		{"bus", "buses"},
		{"watch", "watches"},
		{"dish", "dishes"},
		{"person", "people"},
		{"child", "children"},
		{"baby", "babies"},
		{"key", "keys"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "user"},
		{"pets", "pet"},
		{"owners", "owner"},
		{"customers", "customer"},
		{"categories", "category"},
		{"companies", "company"},
		{"boxes", "box"},
		{"buses", "bus"},
		{"watches", "watch"},
		{"dishes", "dish"},
		{"people", "person"},
		{"children", "child"},
		{"babies", "baby"},
		{"keys", "key"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := singularize(tt.input)
			if result != tt.expected {
				t.Errorf("singularize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/users", "users"},
		{"/users/{id}", "users"},
		{"/users/{userId}/posts", "users"},
		{"/pets", "pets"},
		{"/api/v1/users", "api"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractResourceName(tt.path)
			if result != tt.expected {
				t.Errorf("extractResourceName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestAutoSeederBasic(t *testing.T) {
	spec := createTestSpec()

	config := AutoSeedConfig{
		ItemsPerResource: 3,
	}

	seeder := NewAutoSeeder(spec, config)
	data, err := seeder.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("Generate() returned empty data")
	}

	// Check that owners were generated
	if owners, ok := data["owners"]; ok {
		if len(owners) != 3 {
			t.Errorf("Expected 3 owners, got %d", len(owners))
		}

		// Check that owners have IDs
		for _, owner := range owners {
			ownerMap, ok := owner.(map[string]interface{})
			if !ok {
				t.Error("Owner is not a map")
				continue
			}
			if _, hasID := ownerMap["id"]; !hasID {
				t.Error("Owner missing id field")
			}
		}
	}

	// Check that pets were generated
	if pets, ok := data["pets"]; ok {
		if len(pets) != 3 {
			t.Errorf("Expected 3 pets, got %d", len(pets))
		}

		// Check that pets have owner_id references
		for _, pet := range pets {
			petMap, ok := pet.(map[string]interface{})
			if !ok {
				t.Error("Pet is not a map")
				continue
			}
			if _, hasID := petMap["id"]; !hasID {
				t.Error("Pet missing id field")
			}
			// owner_id should be set because owners were generated first
			if ownerID, hasOwnerID := petMap["owner_id"]; hasOwnerID {
				if ownerID == "" || ownerID == nil {
					t.Error("Pet has empty owner_id")
				}
			}
		}
	}
}

func TestAutoSeederDependencyOrder(t *testing.T) {
	spec := createTestSpec()

	config := AutoSeedConfig{
		ItemsPerResource: 2,
	}

	seeder := NewAutoSeeder(spec, config)
	seeder.extractResources()
	seeder.detectDependencies()

	order := seeder.GetResourceOrder()

	// Owners should come before pets (pets depend on owners)
	ownersIdx := -1
	petsIdx := -1
	for i, r := range order {
		if r == "owners" {
			ownersIdx = i
		}
		if r == "pets" {
			petsIdx = i
		}
	}

	if ownersIdx >= 0 && petsIdx >= 0 && ownersIdx > petsIdx {
		t.Errorf("Owners should be generated before pets. Order: %v", order)
	}
}

func TestAutoSeederDependencyDetection(t *testing.T) {
	spec := createTestSpec()

	config := AutoSeedConfig{
		ItemsPerResource: 2,
	}

	seeder := NewAutoSeeder(spec, config)
	seeder.extractResources()
	seeder.detectDependencies()

	deps := seeder.GetDependencies()

	// Should detect pet -> owner dependency via owner_id field
	found := false
	for _, dep := range deps {
		if dep.Resource == "pets" && dep.DependsOn == "owners" && dep.ForeignKeyField == "owner_id" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected dependency pets -> owners via owner_id, got: %+v", deps)
	}
}

func TestAutoSeederIncludeExclude(t *testing.T) {
	spec := createTestSpec()

	// Test include
	config := AutoSeedConfig{
		ItemsPerResource: 2,
		IncludeResources: []string{"owners"},
	}

	seeder := NewAutoSeeder(spec, config)
	data, err := seeder.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, ok := data["owners"]; !ok {
		t.Error("Expected owners to be included")
	}
	if _, ok := data["pets"]; ok {
		t.Error("Expected pets to be excluded")
	}

	// Test exclude
	config = AutoSeedConfig{
		ItemsPerResource: 2,
		ExcludeResources: []string{"pets"},
	}

	seeder = NewAutoSeeder(spec, config)
	data, err = seeder.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, ok := data["owners"]; !ok {
		t.Error("Expected owners to be included")
	}
	if _, ok := data["pets"]; ok {
		t.Error("Expected pets to be excluded")
	}
}

func TestAutoSeederForeignKeyValues(t *testing.T) {
	spec := createTestSpec()

	config := AutoSeedConfig{
		ItemsPerResource: 3,
	}

	seeder := NewAutoSeeder(spec, config)
	data, err := seeder.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	owners := data["owners"]
	pets := data["pets"]

	if len(owners) == 0 || len(pets) == 0 {
		t.Skip("No owners or pets generated")
	}

	// Collect valid owner IDs
	validOwnerIDs := make(map[string]bool)
	for _, owner := range owners {
		ownerMap := owner.(map[string]interface{})
		if id, ok := ownerMap["id"].(string); ok {
			validOwnerIDs[id] = true
		}
	}

	// Check that all pet owner_ids reference valid owners
	for _, pet := range pets {
		petMap := pet.(map[string]interface{})
		if ownerID, ok := petMap["owner_id"].(string); ok && ownerID != "" {
			if !validOwnerIDs[ownerID] {
				t.Errorf("Pet references non-existent owner: %s", ownerID)
			}
		}
	}
}

func createTestSpec() *openapi3.T {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{},
		},
	}

	// Create Owner schema
	ownerSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:     "object",
			Required: []string{"id", "name", "email"},
			Properties: openapi3.Schemas{
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
				"name": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
				"email": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "email",
					},
				},
			},
		},
	}

	// Create Pet schema with owner_id foreign key
	petSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:     "object",
			Required: []string{"id", "name"},
			Properties: openapi3.Schemas{
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
				"name": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
				"species": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
				"owner_id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
			},
		},
	}

	spec.Components.Schemas["Owner"] = ownerSchema
	spec.Components.Schemas["Pet"] = petSchema

	// Create paths
	spec.Paths.Set("/owners", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Responses: &openapi3.Responses{},
		},
		Post: &openapi3.Operation{
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/Owner",
							},
						},
					},
				},
			},
		},
	})

	spec.Paths.Set("/owners/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Responses: &openapi3.Responses{},
		},
	})

	spec.Paths.Set("/pets", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Responses: &openapi3.Responses{},
		},
		Post: &openapi3.Operation{
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/Pet",
							},
						},
					},
				},
			},
		},
	})

	spec.Paths.Set("/pets/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Responses: &openapi3.Responses{},
		},
	})

	return spec
}

func TestAutoSeederEcommerceScenario(t *testing.T) {
	spec := createEcommerceSpec()

	config := AutoSeedConfig{
		ItemsPerResource: 4,
	}

	seeder := NewAutoSeeder(spec, config)
	data, err := seeder.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check dependency order
	order := seeder.GetResourceOrder()
	t.Logf("Resource order: %v", order)

	// Customers should come before orders
	customersIdx := -1
	ordersIdx := -1
	for i, r := range order {
		if r == "customers" {
			customersIdx = i
		}
		if r == "orders" {
			ordersIdx = i
		}
	}

	if customersIdx >= 0 && ordersIdx >= 0 && customersIdx > ordersIdx {
		t.Errorf("Customers should be generated before orders. Order: %v", order)
	}

	// Check that orders reference valid customers
	customers := data["customers"]
	orders := data["orders"]

	if len(customers) == 0 || len(orders) == 0 {
		t.Skip("No customers or orders generated")
	}

	validCustomerIDs := make(map[string]bool)
	for _, customer := range customers {
		customerMap := customer.(map[string]interface{})
		if id, ok := customerMap["id"].(string); ok {
			validCustomerIDs[id] = true
		}
	}

	for _, order := range orders {
		orderMap := order.(map[string]interface{})
		if customerID, ok := orderMap["customer_id"].(string); ok && customerID != "" {
			if !validCustomerIDs[customerID] {
				t.Errorf("Order references non-existent customer: %s", customerID)
			}
		}
	}
}

func createEcommerceSpec() *openapi3.T {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "E-commerce API",
			Version: "1.0.0",
		},
		Paths: &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{},
		},
	}

	// Create Customer schema
	customerSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:     "object",
			Required: []string{"id", "name", "email"},
			Properties: openapi3.Schemas{
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
				"name": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
				"email": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "email",
					},
				},
			},
		},
	}

	// Create Product schema
	productSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:     "object",
			Required: []string{"id", "name", "price"},
			Properties: openapi3.Schemas{
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
				"name": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
				"price": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "number",
					},
				},
			},
		},
	}

	// Create Order schema with customer_id foreign key
	orderSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:     "object",
			Required: []string{"id", "customer_id"},
			Properties: openapi3.Schemas{
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
				"customer_id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "string",
						Format: "uuid",
					},
				},
				"status": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
						Enum: []interface{}{"pending", "shipped", "delivered"},
					},
				},
				"total": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "number",
					},
				},
			},
		},
	}

	spec.Components.Schemas["Customer"] = customerSchema
	spec.Components.Schemas["Product"] = productSchema
	spec.Components.Schemas["Order"] = orderSchema

	// Create paths
	for _, resource := range []string{"customers", "products", "orders"} {
		schemaName := "Customer"
		if resource == "products" {
			schemaName = "Product"
		} else if resource == "orders" {
			schemaName = "Order"
		}

		spec.Paths.Set("/"+resource, &openapi3.PathItem{
			Get: &openapi3.Operation{
				Responses: &openapi3.Responses{},
			},
			Post: &openapi3.Operation{
				RequestBody: &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/" + schemaName,
								},
							},
						},
					},
				},
			},
		})

		spec.Paths.Set("/"+resource+"/{id}", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Responses: &openapi3.Responses{},
			},
		})
	}

	return spec
}
