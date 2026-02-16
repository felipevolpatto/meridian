package server

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileWatcherCreation(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte("test: data"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config := WatcherConfig{
		Files:    []string{tmpFile},
		Callback: func() {},
		Debounce: 100 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if watcher == nil {
		t.Error("NewFileWatcher() returned nil")
	}

	if len(watcher.WatchedFiles()) != 1 {
		t.Errorf("Expected 1 watched file, got %d", len(watcher.WatchedFiles()))
	}
}

func TestFileWatcherStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte("test: data"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config := WatcherConfig{
		Files:    []string{tmpFile},
		Callback: func() {},
		Debounce: 100 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	// Start watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("Expected watcher to be running")
	}

	// Stop watcher
	if err := watcher.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if watcher.IsRunning() {
		t.Error("Expected watcher to be stopped")
	}
}

func TestFileWatcherDetectsChanges(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte("test: data"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	var callbackCount int32 = 0
	config := WatcherConfig{
		Files: []string{tmpFile},
		Callback: func() {
			atomic.AddInt32(&callbackCount, 1)
		},
		Debounce: 50 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(tmpFile, []byte("test: modified"), 0644); err != nil {
		t.Fatalf("Failed to modify temp file: %v", err)
	}

	// Wait for debounce and callback
	time.Sleep(200 * time.Millisecond)

	count := atomic.LoadInt32(&callbackCount)
	if count == 0 {
		t.Error("Expected callback to be called after file change")
	}
}

func TestFileWatcherDebounce(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte("test: data"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	var callbackCount int32 = 0
	config := WatcherConfig{
		Files: []string{tmpFile},
		Callback: func() {
			atomic.AddInt32(&callbackCount, 1)
		},
		Debounce: 200 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Rapid file changes
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(tmpFile, []byte("test: "+string(rune('a'+i))), 0644); err != nil {
			t.Fatalf("Failed to modify temp file: %v", err)
		}
		time.Sleep(30 * time.Millisecond)
	}

	// Wait for debounce
	time.Sleep(400 * time.Millisecond)

	count := atomic.LoadInt32(&callbackCount)
	// Due to debouncing, we should have fewer callbacks than changes
	if count > 2 {
		t.Errorf("Expected debouncing to limit callbacks, got %d", count)
	}
}

func TestFileWatcherIgnoresUnwatchedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	watchedFile := filepath.Join(tmpDir, "watched.yaml")
	unwatchedFile := filepath.Join(tmpDir, "unwatched.yaml")

	if err := os.WriteFile(watchedFile, []byte("test: data"), 0644); err != nil {
		t.Fatalf("Failed to create watched file: %v", err)
	}
	if err := os.WriteFile(unwatchedFile, []byte("test: data"), 0644); err != nil {
		t.Fatalf("Failed to create unwatched file: %v", err)
	}

	var callbackCount int32 = 0
	config := WatcherConfig{
		Files: []string{watchedFile},
		Callback: func() {
			atomic.AddInt32(&callbackCount, 1)
		},
		Debounce: 50 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Modify unwatched file
	if err := os.WriteFile(unwatchedFile, []byte("test: modified"), 0644); err != nil {
		t.Fatalf("Failed to modify unwatched file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	count := atomic.LoadInt32(&callbackCount)
	if count != 0 {
		t.Errorf("Expected no callback for unwatched file, got %d", count)
	}
}

func TestFileWatcherMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config.yaml")
	file2 := filepath.Join(tmpDir, "openapi.yaml")

	if err := os.WriteFile(file1, []byte("test: 1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("test: 2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	var callbackCount int32 = 0
	config := WatcherConfig{
		Files: []string{file1, file2},
		Callback: func() {
			atomic.AddInt32(&callbackCount, 1)
		},
		Debounce: 50 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if len(watcher.WatchedFiles()) != 2 {
		t.Errorf("Expected 2 watched files, got %d", len(watcher.WatchedFiles()))
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Modify first file
	if err := os.WriteFile(file1, []byte("test: modified1"), 0644); err != nil {
		t.Fatalf("Failed to modify file1: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	count1 := atomic.LoadInt32(&callbackCount)
	if count1 == 0 {
		t.Error("Expected callback for file1 change")
	}

	// Modify second file
	if err := os.WriteFile(file2, []byte("test: modified2"), 0644); err != nil {
		t.Fatalf("Failed to modify file2: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	count2 := atomic.LoadInt32(&callbackCount)
	if count2 <= count1 {
		t.Error("Expected callback for file2 change")
	}
}

func TestIsWatchedFile(t *testing.T) {
	tmpDir := t.TempDir()
	watchedFile := filepath.Join(tmpDir, "watched.yaml")

	if err := os.WriteFile(watchedFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	watcher, _ := NewFileWatcher(WatcherConfig{
		Files:    []string{watchedFile},
		Callback: func() {},
	})

	tests := []struct {
		path     string
		expected bool
	}{
		{watchedFile, true},
		{filepath.Join(tmpDir, "other.yaml"), false},
		{filepath.Join(tmpDir, "subdir", "watched.yaml"), true}, // base name match
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := watcher.isWatchedFile(tt.path)
			if result != tt.expected {
				t.Errorf("isWatchedFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDefaultDebounce(t *testing.T) {
	config := WatcherConfig{
		Files:    []string{"/tmp/test.yaml"},
		Callback: func() {},
		// Debounce not set
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if watcher.debounce != 500*time.Millisecond {
		t.Errorf("Expected default debounce of 500ms, got %v", watcher.debounce)
	}
}
