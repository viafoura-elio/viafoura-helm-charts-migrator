package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"helm-charts-migrator/v1/pkg/logger"

	"github.com/spf13/cobra"
)

//go:embed default-config.yaml
var defaultConfigContent string // This will contain the entire file content at compile time

var (
	initForce  bool
	initOutput string
)

var initCmd = &cobra.Command{
	Use:   "init [flags]",
	Short: "Initialize a new configuration file",
	Long: `Initialize a new configuration file with default values.

This command creates a new config.yaml file with sensible defaults
for the Helm Charts Migrator. The configuration includes settings for:
  - Cluster configurations (dev01, prod01)
  - Service definitions
  - Secret extraction patterns
  - Transformation rules
  - SOPS encryption settings

Examples:
  # Create default config.yaml in current directory
  helm-charts-migrator init

  # Create config in specific location
  helm-charts-migrator init --output /path/to/config.yaml

  # Overwrite existing config
  helm-charts-migrator init --force`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config file")
	initCmd.Flags().StringVarP(&initOutput, "output", "o", "./config.yaml", "Output path for the config file")
}

func runInit(_ *cobra.Command, _ []string) error {
	log := logger.WithName("init")

	// Resolve output path
	configPath, err := filepath.Abs(initOutput)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
	}

	// Use the embedded default config
	content := defaultConfigContent

	// Create directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the config file
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.InfoS("Configuration file created successfully",
		"path", configPath)

	// Print next steps
	fmt.Println("\n✅ Configuration file created successfully!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Edit the configuration file to match your environment:")
	fmt.Printf("     $ vi %s\n", configPath)
	fmt.Println("  2. Update cluster contexts (dev01, prod01) to match your Kubernetes clusters")
	fmt.Println("  3. Configure services you want to migrate")
	fmt.Println("  4. Run migration:")
	fmt.Printf("     $ helm-charts-migrator migrate --config %s\n", configPath)

	return nil
}
