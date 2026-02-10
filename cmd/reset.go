package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felipevolpatto/meridian/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset server state",
	Long:  `Reset the mock server state to its initial configuration.`,
	RunE:  runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().BoolP("seed", "s", true, "Reload seed data after reset")
	resetCmd.Flags().BoolP("backup", "b", true, "Create backup before reset")
	resetCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	resetCmd.Flags().StringP("config", "c", "meridian.yaml", "Path to configuration file")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func runReset(cmd *cobra.Command, args []string) error {
	seed, _ := cmd.Flags().GetBool("seed")
	backup, _ := cmd.Flags().GetBool("backup")
	force, _ := cmd.Flags().GetBool("force")
	configPath, _ := cmd.Flags().GetString("config")

	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	manager, err := state.New(config.State.Persistence)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	defer manager.Close()

	if !force {
		fmt.Print("⚠️  This will reset all server state. Are you sure? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Reset cancelled")
			return nil
		}
	}

	if backup {
		if err := createBackup(manager); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	if err := manager.Reset(); err != nil {
		return fmt.Errorf("failed to reset state: %w", err)
	}

	if seed {
		if err := loadSeedData(manager); err != nil {
			return fmt.Errorf("failed to load seed data: %w", err)
		}
	}

	fmt.Println("✅ Successfully reset server state")
	if backup {
		fmt.Println("   Backup created: backup_YYYYMMDD_HHMMSS.json")
	}
	if seed {
		fmt.Println("   Seed data loaded")
	}
	return nil
}

func createBackup(manager *state.Manager) error {
	data, err := manager.Export()
	if err != nil {
		return fmt.Errorf("failed to export current state: %w", err)
	}

	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("backup_%s.json", timestamp))

	if err := writeFileIfNotExists(backupPath, data, true); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

func loadSeedData(manager *state.Manager) error {
	seedPath := "seed_data.json"
	if _, err := os.Stat(seedPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("   No seed file found, skipping")
			return nil
		}
		return fmt.Errorf("failed to check seed file: %w", err)
	}

	data, err := os.ReadFile(seedPath)
	if err != nil {
		return fmt.Errorf("failed to read seed file: %w", err)
	}

	var seedData map[string][]interface{}
	if err := json.Unmarshal(data, &seedData); err != nil {
		return fmt.Errorf("failed to parse seed data: %w", err)
	}

	importData := &state.ExportData{
		Version:   "1.0",
		Resources: seedData,
		Timestamps: state.Timestamps{
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		},
	}

	if err := manager.Import(importData, false); err != nil {
		return fmt.Errorf("failed to import seed data: %w", err)
	}

	return nil
}
