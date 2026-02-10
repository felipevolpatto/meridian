package state

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateManagement(t *testing.T) {
	// Create temporary files for testing
	tmpDB, err := os.CreateTemp("", "meridian_test_*.db")
	assert.NoError(t, err)
	defer os.Remove(tmpDB.Name())

	// Create state manager
	manager, err := New(tmpDB.Name())
	assert.NoError(t, err)
	defer manager.Close()

	// Test adding new data
	err = manager.AddResource("users", map[string]interface{}{
		"id":   1,
		"name": "Alice",
	})
	assert.NoError(t, err)

	err = manager.AddResource("users", map[string]interface{}{
		"id":   2,
		"name": "Bob",
	})
	assert.NoError(t, err)

	// Verify data was added
	users, err := manager.GetResources("users")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Contains(t, users, map[string]interface{}{"id": float64(1), "name": "Alice"})
	assert.Contains(t, users, map[string]interface{}{"id": float64(2), "name": "Bob"})

	// Test adding more data
	err = manager.AddResource("users", map[string]interface{}{
		"id":   3,
		"name": "Charlie",
	})
	assert.NoError(t, err)

	// Verify data was added
	users, err = manager.GetResources("users")
	assert.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Contains(t, users, map[string]interface{}{"id": float64(3), "name": "Charlie"})

	// Test getting a specific resource
	alice, err := manager.GetResource("users", "1")
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"id": float64(1), "name": "Alice"}, alice)

	// Test updating data
	err = manager.UpdateResource("users", "1", map[string]interface{}{
		"id":   1,
		"name": "Alice Updated",
	})
	assert.NoError(t, err)

	// Verify update
	alice, err = manager.GetResource("users", "1")
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"id": float64(1), "name": "Alice Updated"}, alice)

	// Test deleting a resource
	err = manager.DeleteResource("users", "2")
	assert.NoError(t, err)

	// Verify deletion
	users, err = manager.GetResources("users")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.NotContains(t, users, map[string]interface{}{"id": float64(2), "name": "Bob"})

	// Test resetting state
	err = manager.Reset()
	assert.NoError(t, err)

	// Verify reset
	users, err = manager.GetResources("users")
	assert.NoError(t, err)
	assert.Empty(t, users)
}

func TestImportExport(t *testing.T) {
	// Create temporary file for testing
	tmpDB, err := os.CreateTemp("", "meridian_test_*.db")
	assert.NoError(t, err)
	defer os.Remove(tmpDB.Name())

	// Create state manager
	manager, err := New(tmpDB.Name())
	assert.NoError(t, err)
	defer manager.Close()

	// Add some test data
	err = manager.AddResource("users", map[string]interface{}{
		"id":   1,
		"name": "Alice",
	})
	assert.NoError(t, err)

	err = manager.AddResource("users", map[string]interface{}{
		"id":   2,
		"name": "Bob",
	})
	assert.NoError(t, err)

	// Export data
	exportData, err := manager.Export()
	assert.NoError(t, err)
	assert.Equal(t, "1.0", exportData.Version)
	assert.Len(t, exportData.Resources["users"], 2)

	// Reset state
	err = manager.Reset()
	assert.NoError(t, err)

	// Import data
	err = manager.Import(exportData, false)
	assert.NoError(t, err)

	// Verify imported data
	users, err := manager.GetResources("users")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Contains(t, users, map[string]interface{}{"id": float64(1), "name": "Alice"})
	assert.Contains(t, users, map[string]interface{}{"id": float64(2), "name": "Bob"})

	// Test merging data
	newData := &ExportData{
		Version: "1.0",
		Resources: map[string][]interface{}{
			"users": {
				map[string]interface{}{
					"id":   3,
					"name": "Charlie",
				},
			},
		},
		Relations:  make(map[string]map[string]string),
		Metadata:   make(map[string]map[string]interface{}),
		Timestamps: Timestamps{},
	}

	err = manager.Import(newData, true)
	assert.NoError(t, err)

	// Verify merged data
	users, err = manager.GetResources("users")
	assert.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Contains(t, users, map[string]interface{}{"id": float64(1), "name": "Alice"})
	assert.Contains(t, users, map[string]interface{}{"id": float64(2), "name": "Bob"})
	assert.Contains(t, users, map[string]interface{}{"id": float64(3), "name": "Charlie"})
}

func TestInvalidOperations(t *testing.T) {
	// Create temporary file for testing
	tmpDB, err := os.CreateTemp("", "meridian_test_*.db")
	assert.NoError(t, err)
	defer os.Remove(tmpDB.Name())

	// Create state manager
	manager, err := New(tmpDB.Name())
	assert.NoError(t, err)
	defer manager.Close()

	// Test getting non-existent resource
	_, err = manager.GetResource("users", "1")
	assert.Error(t, err)

	// Test updating non-existent resource
	err = manager.UpdateResource("users", "1", map[string]interface{}{
		"id":   1,
		"name": "Alice",
	})
	assert.Error(t, err)

	// Test deleting non-existent resource
	err = manager.DeleteResource("users", "1")
	assert.Error(t, err)

	// Test adding resource with invalid data
	err = manager.AddResource("users", "not an object")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource data format")

	// Test adding resource without ID
	err = manager.AddResource("users", map[string]interface{}{
		"name": "Alice",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource data format")
}

func TestNonExistentFiles(t *testing.T) {
	// Test with non-existent file
	manager, err := New("non_existent.db")
	assert.NoError(t, err)
	defer manager.Close()

	// Verify empty state
	users, err := manager.GetResources("users")
	assert.NoError(t, err)
	assert.Empty(t, users)
}