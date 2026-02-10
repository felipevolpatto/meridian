package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var globalManager *Manager

func Initialize(dbPath string, seedPath string) error {
	var err error
	globalManager, err = New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	if seedPath != "" {
		if err := loadSeedFile(seedPath); err != nil {
			fmt.Printf("Warning: failed to load seed data: %v\n", err)
		}
	}

	return nil
}

func Close() error {
	if globalManager == nil {
		return nil
	}
	err := globalManager.Close()
	globalManager = nil
	return err
}

func GetManager() *Manager {
	return globalManager
}

func loadSeedFile(seedPath string) error {
	if _, err := os.Stat(seedPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(seedPath)
	if err != nil {
		return fmt.Errorf("failed to read seed file: %w", err)
	}

	var seedData map[string][]interface{}
	if err := json.Unmarshal(data, &seedData); err != nil {
		return fmt.Errorf("failed to parse seed data: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	importData := &ExportData{
		Version:   "1.0",
		Resources: seedData,
		Timestamps: Timestamps{
			ExportedAt: now,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	if err := globalManager.Import(importData, false); err != nil {
		return fmt.Errorf("failed to import seed data: %w", err)
	}

	return nil
}

type Manager struct {
	db *sql.DB
}

type Resource struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ResourceRelation struct {
	SourceID   string    `json:"source_id"`
	SourceType string    `json:"source_type"`
	TargetID   string    `json:"target_id"`
	TargetType string    `json:"target_type"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
}

func New(dbPath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS resources (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			data JSON NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create resources table: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS relationships (
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (source_id, target_id, type),
			FOREIGN KEY (source_id) REFERENCES resources(id),
			FOREIGN KEY (target_id) REFERENCES resources(id)
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create relationships table: %w", err)
	}

	return &Manager{db: db}, nil
}

func (m *Manager) Close() error {
	if m.db == nil {
		return nil
	}
	return m.db.Close()
}

func (m *Manager) Export() (*ExportData, error) {
	data := &ExportData{
		Version:    "1.0",
		Resources:  make(map[string][]interface{}),
		Relations:  make(map[string]map[string]string),
		Metadata:   make(map[string]map[string]interface{}),
		Timestamps: Timestamps{
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	rows, err := m.db.Query("SELECT id, type, data, created_at, updated_at FROM resources")
	if err != nil {
		return nil, fmt.Errorf("failed to query resources: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var resource Resource
		var createdAt, updatedAt string
		if err := rows.Scan(&resource.ID, &resource.Type, &resource.Data, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}

		var resourceData interface{}
		if err := json.Unmarshal(resource.Data, &resourceData); err != nil {
			return nil, fmt.Errorf("failed to parse resource data: %w", err)
		}

		if data.Resources[resource.Type] == nil {
			data.Resources[resource.Type] = make([]interface{}, 0)
		}
		data.Resources[resource.Type] = append(data.Resources[resource.Type], resourceData)

		if data.Timestamps.CreatedAt == "" || createdAt < data.Timestamps.CreatedAt {
			data.Timestamps.CreatedAt = createdAt
		}
		if data.Timestamps.UpdatedAt == "" || updatedAt > data.Timestamps.UpdatedAt {
			data.Timestamps.UpdatedAt = updatedAt
		}
	}

	rows, err = m.db.Query("SELECT source_type, target_type, type FROM relationships GROUP BY source_type, target_type, type")
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sourceType, targetType, relType string
		if err := rows.Scan(&sourceType, &targetType, &relType); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		if data.Relations[sourceType] == nil {
			data.Relations[sourceType] = make(map[string]string)
		}
		data.Relations[sourceType][targetType] = relType
	}

	for resourceType, resources := range data.Resources {
		if data.Metadata[resourceType] == nil {
			data.Metadata[resourceType] = make(map[string]interface{})
		}
		data.Metadata[resourceType]["total_count"] = len(resources)
	}

	return data, nil
}

func (m *Manager) Import(data *ExportData, merge bool) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if !merge {
		if _, err := tx.Exec("DELETE FROM relationships"); err != nil {
			return fmt.Errorf("failed to clear relationships: %w", err)
		}
		if _, err := tx.Exec("DELETE FROM resources"); err != nil {
			return fmt.Errorf("failed to clear resources: %w", err)
		}
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO resources (id, type, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare resource statement: %w", err)
	}
	defer stmt.Close()

	for resourceType, resources := range data.Resources {
		for _, resource := range resources {
			resourceData, err := json.Marshal(resource)
			if err != nil {
				return fmt.Errorf("failed to marshal resource data: %w", err)
			}

			resourceMap := resource.(map[string]interface{})
			id := fmt.Sprintf("%v", resourceMap["id"])

			_, err = stmt.Exec(
				id,
				resourceType,
				resourceData,
				data.Timestamps.CreatedAt,
				data.Timestamps.UpdatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to insert resource: %w", err)
			}
		}
	}

	stmt, err = tx.Prepare(`
		INSERT OR REPLACE INTO relationships (source_id, source_type, target_id, target_type, type, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare relationship statement: %w", err)
	}
	defer stmt.Close()

	for sourceType, relations := range data.Relations {
		for targetType, relType := range relations {
			sourceResources := data.Resources[sourceType]
			targetResources := data.Resources[targetType]

			for _, source := range sourceResources {
				sourceMap := source.(map[string]interface{})
				sourceID := fmt.Sprintf("%v", sourceMap["id"])

				if relType == "one_to_many" || relType == "many_to_many" {
					for _, target := range targetResources {
						targetMap := target.(map[string]interface{})
						targetID := fmt.Sprintf("%v", targetMap["id"])

						_, err = stmt.Exec(
							sourceID,
							sourceType,
							targetID,
							targetType,
							relType,
							data.Timestamps.CreatedAt,
						)
						if err != nil {
							return fmt.Errorf("failed to insert relationship: %w", err)
						}
					}
				} else {
					if len(targetResources) > 0 {
						targetMap := targetResources[0].(map[string]interface{})
						targetID := fmt.Sprintf("%v", targetMap["id"])

						_, err = stmt.Exec(
							sourceID,
							sourceType,
							targetID,
							targetType,
							relType,
							data.Timestamps.CreatedAt,
						)
						if err != nil {
							return fmt.Errorf("failed to insert relationship: %w", err)
						}
					}
				}
			}
		}
	}

	return tx.Commit()
}

func (m *Manager) Reset() error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM relationships"); err != nil {
		return fmt.Errorf("failed to clear relationships: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM resources"); err != nil {
		return fmt.Errorf("failed to clear resources: %w", err)
	}

	return tx.Commit()
}

type ExportData struct {
	Version    string                            `json:"version"`
	Resources  map[string][]interface{}          `json:"resources"`
	Relations  map[string]map[string]string      `json:"relations"`
	Metadata   map[string]map[string]interface{} `json:"metadata"`
	Timestamps Timestamps                        `json:"timestamps"`
}

type Timestamps struct {
	ExportedAt string `json:"exported_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (m *Manager) GetResources(resourceType string) ([]interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	rows, err := m.db.Query("SELECT data FROM resources WHERE type = ?", resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to query resources: %w", err)
	}
	defer rows.Close()

	var resources []interface{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}

		var resource interface{}
		if err := json.Unmarshal(data, &resource); err != nil {
			return nil, fmt.Errorf("failed to parse resource data: %w", err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func (m *Manager) GetResource(resourceType, id string) (interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	var data []byte
	err := m.db.QueryRow("SELECT data FROM resources WHERE type = ? AND id = ?", resourceType, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("resource not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	var resource interface{}
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse resource data: %w", err)
	}

	return resource, nil
}

func (m *Manager) AddResource(resourceType string, data interface{}) error {
	resourceData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal resource data: %w", err)
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid resource data format")
	}

	idVal, ok := dataMap["id"]
	if !ok {
		return fmt.Errorf("invalid resource data format: missing id field")
	}

	id := fmt.Sprintf("%v", idVal)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = m.db.Exec(`
		INSERT INTO resources (id, type, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, resourceType, resourceData, now, now)
	if err != nil {
		return fmt.Errorf("failed to insert resource: %w", err)
	}

	return nil
}

func (m *Manager) UpdateResource(resourceType, id string, data interface{}) error {
	resourceData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal resource data: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	result, err := m.db.Exec(`
		UPDATE resources
		SET data = ?, updated_at = ?
		WHERE type = ? AND id = ?
	`, resourceData, now, resourceType, id)
	if err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("resource not found")
	}

	return nil
}

func (m *Manager) DeleteResource(resourceType, id string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM relationships
		WHERE (source_type = ? AND source_id = ?)
		   OR (target_type = ? AND target_id = ?)
	`, resourceType, id, resourceType, id)
	if err != nil {
		return fmt.Errorf("failed to delete relationships: %w", err)
	}

	result, err := tx.Exec(`
		DELETE FROM resources
		WHERE type = ? AND id = ?
	`, resourceType, id)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("resource not found")
	}

	return tx.Commit()
}
