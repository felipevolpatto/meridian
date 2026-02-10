package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/felipevolpatto/meridian/internal/openapi"
	"github.com/felipevolpatto/meridian/internal/server"
	"github.com/felipevolpatto/meridian/internal/state"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the mock API server",
	Long:  `Reads the spec and the optional config, and starts the live server on a local port.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load("meridian.yaml")
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		spec, err := openapi.ParseFile(cfg.OpenAPI)
		if err != nil {
			log.Fatalf("Error parsing OpenAPI spec: %v", err)
		}

		if err := state.Initialize(cfg.State.Persistence, cfg.State.Seed); err != nil {
			log.Fatalf("Error initializing state: %v", err)
		}

		srv := server.NewServer(spec, cfg)

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		go func() {
			if err := server.StartServer(spec, cfg); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Error starting server: %v", err)
			}
		}()

		<-stop
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}

		if err := state.Close(); err != nil {
			log.Printf("Error closing state: %v", err)
		}

		log.Println("Server stopped")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
