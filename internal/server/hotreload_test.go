package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHotReloadServerCreation(t *testing.T) {
	hrs := NewHotReloadServer("meridian.yaml")

	if hrs == nil {
		t.Error("NewHotReloadServer() returned nil")
	}

	if hrs.configPath != "meridian.yaml" {
		t.Errorf("Expected configPath 'meridian.yaml', got %q", hrs.configPath)
	}

	if hrs.IsRunning() {
		t.Error("New server should not be running")
	}
}

func TestHotReloadServerDoubleStart(t *testing.T) {
	// Create temp config files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "meridian.yaml")
	openapiPath := filepath.Join(tmpDir, "openapi.yaml")

	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 19999
state:
  persistence: ":memory:"
`
	openapiContent := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	if err := os.WriteFile(openapiPath, []byte(openapiContent), 0644); err != nil {
		t.Fatalf("Failed to write openapi: %v", err)
	}

	// Change to temp dir for relative path resolution
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	hrs := NewHotReloadServer(configPath)

	// Start server
	if err := hrs.Start(); err != nil {
		t.Fatalf("First Start() error = %v", err)
	}

	// Give it time to initialize
	time.Sleep(100 * time.Millisecond)

	// Try to start again
	err := hrs.Start()
	if err == nil {
		t.Error("Expected error on double start")
	}

	// Stop server
	if err := hrs.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestHotReloadServerStop(t *testing.T) {
	hrs := NewHotReloadServer("meridian.yaml")

	// Stop without starting should not error
	if err := hrs.Stop(); err != nil {
		t.Errorf("Stop() on non-started server error = %v", err)
	}
}

func TestHotReloadServerTriggerReload(t *testing.T) {
	hrs := NewHotReloadServer("meridian.yaml")

	// TriggerReload should not panic on non-started server
	hrs.TriggerReload()

	// Multiple rapid triggers should not block
	for i := 0; i < 10; i++ {
		hrs.TriggerReload()
	}
}

func TestHotReloadConfig(t *testing.T) {
	config := HotReloadConfig{
		ConfigPath:    "meridian.yaml",
		Debounce:      500 * time.Millisecond,
		PreserveState: true,
	}

	if config.ConfigPath != "meridian.yaml" {
		t.Error("ConfigPath not set correctly")
	}

	if config.Debounce != 500*time.Millisecond {
		t.Error("Debounce not set correctly")
	}

	if !config.PreserveState {
		t.Error("PreserveState not set correctly")
	}
}

func TestHotReloadServerIsRunning(t *testing.T) {
	hrs := NewHotReloadServer("meridian.yaml")

	if hrs.IsRunning() {
		t.Error("New server should not be running")
	}
}
