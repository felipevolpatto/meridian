package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/felipevolpatto/meridian/internal/generator"
	"github.com/felipevolpatto/meridian/internal/openapi"
	"github.com/felipevolpatto/meridian/internal/state"
	"github.com/getkin/kin-openapi/openapi3"
)

// HotReloadServer wraps a server with hot reload capabilities
type HotReloadServer struct {
	configPath string
	server     *Server
	watcher    *FileWatcher
	mu         sync.Mutex
	isRunning  bool
	stopChan   chan struct{}
	reloadChan chan struct{}
}

// HotReloadConfig configures hot reload behavior
type HotReloadConfig struct {
	// Path to meridian.yaml config file
	ConfigPath string
	// Debounce duration for file changes
	Debounce time.Duration
	// Whether to preserve state on reload
	PreserveState bool
}

// NewHotReloadServer creates a new hot reload server
func NewHotReloadServer(configPath string) *HotReloadServer {
	return &HotReloadServer{
		configPath: configPath,
		stopChan:   make(chan struct{}),
		reloadChan: make(chan struct{}, 1),
	}
}

// Start starts the server with hot reload enabled
func (hrs *HotReloadServer) Start() error {
	hrs.mu.Lock()
	if hrs.isRunning {
		hrs.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	hrs.isRunning = true
	hrs.mu.Unlock()

	// Initial load
	cfg, spec, err := hrs.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load initial config: %w", err)
	}

	// Setup file watcher
	watchFiles := []string{hrs.configPath}
	if cfg.OpenAPI != "" {
		watchFiles = append(watchFiles, cfg.OpenAPI)
	}
	if cfg.State.Seed != "" {
		watchFiles = append(watchFiles, cfg.State.Seed)
	}

	hrs.watcher, err = NewFileWatcher(WatcherConfig{
		Files:    watchFiles,
		Debounce: 500 * time.Millisecond,
		Callback: func() {
			select {
			case hrs.reloadChan <- struct{}{}:
			default:
				// Channel full, reload already pending
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start watcher
	if err := hrs.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	log.Printf("Hot reload enabled, watching: %v", watchFiles)

	// Start server loop
	go hrs.serverLoop(cfg, spec)

	return nil
}

// serverLoop manages the server lifecycle with reloads
func (hrs *HotReloadServer) serverLoop(cfg *config.Config, spec *openapi3.T) {
	for {
		// Initialize state
		if err := hrs.initializeState(cfg, spec); err != nil {
			log.Printf("Error initializing state: %v", err)
		}

		// Create server
		srv := NewServer(spec, cfg)
		hrs.mu.Lock()
		hrs.server = srv
		hrs.mu.Unlock()

		addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			if err := StartServer(spec, cfg); err != nil {
				serverErr <- err
			}
			close(serverErr)
		}()

		log.Printf("Server started on http://%s", addr)

		// Wait for reload signal, stop signal, or server error
		select {
		case <-hrs.stopChan:
			hrs.shutdownServer(srv)
			state.Close()
			return

		case <-hrs.reloadChan:
			log.Println("Reloading server...")
			hrs.shutdownServer(srv)
			state.Close()

			// Reload config
			newCfg, newSpec, err := hrs.loadConfig()
			if err != nil {
				log.Printf("Error reloading config: %v, keeping old config", err)
				// Retry with old config
				continue
			}

			cfg = newCfg
			spec = newSpec

			// Update watched files if changed
			hrs.updateWatchedFiles(cfg)
			log.Println("Server reloaded successfully")

		case err := <-serverErr:
			if err != nil {
				log.Printf("Server error: %v", err)
			}
			return
		}
	}
}

// loadConfig loads the configuration and OpenAPI spec
func (hrs *HotReloadServer) loadConfig() (*config.Config, *openapi3.T, error) {
	cfg, err := config.Load(hrs.configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	spec, err := openapi.ParseFile(cfg.OpenAPI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return cfg, spec, nil
}

// initializeState initializes the state manager
func (hrs *HotReloadServer) initializeState(cfg *config.Config, spec *openapi3.T) error {
	initOpts := state.InitializeOptions{
		DBPath:          cfg.State.Persistence,
		SeedPath:        cfg.State.Seed,
		AutoSeedEnabled: cfg.State.AutoSeed.Enabled,
		Spec:            spec,
	}

	if cfg.State.AutoSeed.Enabled {
		initOpts.AutoSeedConfig = generator.AutoSeedConfig{
			ItemsPerResource: cfg.State.AutoSeed.ItemsPerResource,
			IncludeResources: cfg.State.AutoSeed.IncludeResources,
			ExcludeResources: cfg.State.AutoSeed.ExcludeResources,
		}
		if initOpts.AutoSeedConfig.ItemsPerResource <= 0 {
			initOpts.AutoSeedConfig.ItemsPerResource = 5
		}
	}

	return state.InitializeWithOptions(initOpts)
}

// shutdownServer gracefully shuts down the server
func (hrs *HotReloadServer) shutdownServer(srv *Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
}

// updateWatchedFiles updates the files being watched
func (hrs *HotReloadServer) updateWatchedFiles(cfg *config.Config) {
	// For simplicity, we keep watching the same files
	// A more sophisticated implementation could update the watcher
}

// Stop stops the hot reload server
func (hrs *HotReloadServer) Stop() error {
	hrs.mu.Lock()
	defer hrs.mu.Unlock()

	if !hrs.isRunning {
		return nil
	}

	close(hrs.stopChan)
	hrs.isRunning = false

	if hrs.watcher != nil {
		hrs.watcher.Stop()
	}

	return nil
}

// IsRunning returns whether the server is running
func (hrs *HotReloadServer) IsRunning() bool {
	hrs.mu.Lock()
	defer hrs.mu.Unlock()
	return hrs.isRunning
}

// TriggerReload manually triggers a reload
func (hrs *HotReloadServer) TriggerReload() {
	select {
	case hrs.reloadChan <- struct{}{}:
	default:
	}
}
