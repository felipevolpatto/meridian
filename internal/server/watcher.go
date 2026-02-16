package server

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches files for changes and triggers callbacks
type FileWatcher struct {
	watcher     *fsnotify.Watcher
	files       []string
	callback    func()
	debounce    time.Duration
	stopChan    chan struct{}
	mu          sync.Mutex
	lastEvent   time.Time
	isRunning   bool
}

// WatcherConfig configures the file watcher
type WatcherConfig struct {
	// Files to watch
	Files []string
	// Callback function when files change
	Callback func()
	// Debounce duration to prevent rapid restarts
	Debounce time.Duration
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(config WatcherConfig) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if config.Debounce == 0 {
		config.Debounce = 500 * time.Millisecond
	}

	fw := &FileWatcher{
		watcher:  watcher,
		files:    config.Files,
		callback: config.Callback,
		debounce: config.Debounce,
		stopChan: make(chan struct{}),
	}

	return fw, nil
}

// Start begins watching files
func (fw *FileWatcher) Start() error {
	fw.mu.Lock()
	if fw.isRunning {
		fw.mu.Unlock()
		return nil
	}
	fw.isRunning = true
	fw.mu.Unlock()

	// Add files to watch
	for _, file := range fw.files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			log.Printf("Warning: could not resolve path %s: %v", file, err)
			continue
		}

		// Watch the directory containing the file for better compatibility
		dir := filepath.Dir(absPath)
		if err := fw.watcher.Add(dir); err != nil {
			log.Printf("Warning: could not watch %s: %v", dir, err)
		}
	}

	go fw.watch()
	return nil
}

// Stop stops watching files
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.isRunning {
		return nil
	}

	close(fw.stopChan)
	fw.isRunning = false
	return fw.watcher.Close()
}

// watch is the main event loop
func (fw *FileWatcher) watch() {
	var debounceTimer *time.Timer

	for {
		select {
		case <-fw.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Check if this event is for one of our watched files
			if !fw.isWatchedFile(event.Name) {
				continue
			}

			// Only react to write and create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Debounce: reset timer on each event
			fw.mu.Lock()
			now := time.Now()
			if now.Sub(fw.lastEvent) < fw.debounce {
				fw.mu.Unlock()
				continue
			}
			fw.lastEvent = now
			fw.mu.Unlock()

			// Use timer for debouncing
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(fw.debounce, func() {
				log.Printf("File changed: %s, triggering reload...", filepath.Base(event.Name))
				if fw.callback != nil {
					fw.callback()
				}
			})

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// isWatchedFile checks if the changed file is one we're watching
func (fw *FileWatcher) isWatchedFile(changedPath string) bool {
	changedAbs, err := filepath.Abs(changedPath)
	if err != nil {
		return false
	}
	changedBase := filepath.Base(changedAbs)

	for _, file := range fw.files {
		fileAbs, err := filepath.Abs(file)
		if err != nil {
			continue
		}

		// Check exact match
		if changedAbs == fileAbs {
			return true
		}

		// Check base name match (for when full path differs)
		if changedBase == filepath.Base(fileAbs) {
			return true
		}
	}

	return false
}

// IsRunning returns whether the watcher is running
func (fw *FileWatcher) IsRunning() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.isRunning
}

// WatchedFiles returns the list of watched files
func (fw *FileWatcher) WatchedFiles() []string {
	return fw.files
}
