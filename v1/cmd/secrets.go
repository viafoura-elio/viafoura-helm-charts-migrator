package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/sops"
	"helm-charts-migrator/v1/pkg/yaml"

	"github.com/spf13/cobra"
)

var (
	secretsServices []string
	sopsConfigPath  string
	extractOnly     bool
	encryptOnly     bool
	decryptOnly     bool
	validateOnly    bool
	awsProfile      string
	patterns        []string
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets [path]",
	Short: "Manage secrets extraction, encryption, and decryption",
	Long: `Extract secrets from values files and manage encryption/decryption using SOPS.

This command extracts the 'secrets' key from environment-specific values.yaml files
into separate secrets.dec.yaml files, and provides encryption/decryption capabilities
using SOPS with AWS KMS.

You can specify a path to process specific directories or files. If no path is provided,
it will process services specified with --services flag or all enabled services.

Examples:
  # Process all enabled services
  helm-charts-migrator secrets
  
  # Process specific services
  helm-charts-migrator secrets --services heimdall,livecomments
  
  # Process a specific path
  helm-charts-migrator secrets apps/heimdall
  
  # Encrypt only existing .dec files in a path
  helm-charts-migrator secrets apps/heimdall --encrypt-only
  
  # Decrypt .enc files in a specific environment
  helm-charts-migrator secrets apps/heimdall/envs/dev01 --decrypt-only
  
  # Validate all secrets files are properly encrypted
  helm-charts-migrator secrets --validate`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSecrets,
}

func init() {
	rootCmd.AddCommand(secretsCmd)

	secretsCmd.Flags().StringSliceVarP(&secretsServices, "services", "s", []string{}, "Services to process (comma-separated)")
	secretsCmd.Flags().StringVarP(&sopsConfigPath, "sops-config", "", ".sops.yaml", "Path to SOPS configuration file")
	secretsCmd.Flags().StringVarP(&awsProfile, "aws-profile", "p", "cicd-sre", "AWS profile to use for KMS operations")
	secretsCmd.Flags().BoolVar(&extractOnly, "extract-only", false, "Only extract secrets, don't encrypt")
	secretsCmd.Flags().BoolVar(&encryptOnly, "encrypt-only", false, "Only encrypt existing .dec files")
	secretsCmd.Flags().BoolVar(&decryptOnly, "decrypt-only", false, "Only decrypt existing .enc files")
	secretsCmd.Flags().BoolVar(&validateOnly, "validate", false, "Validate that all secrets files are properly encrypted")
	secretsCmd.Flags().StringSliceVar(&patterns, "pattern", []string{}, "File patterns to process (can be specified multiple times, default: current directory)")
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
}

func runSecrets(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.WithName("secrets")

	// Set AWS profile environment variable for SOPS
	if awsProfile != "" {
		if err := os.Setenv("AWS_PROFILE", awsProfile); err != nil {
			return fmt.Errorf("failed to set AWS_PROFILE: %w", err)
		}
		log.V(1).InfoS("Using AWS profile for SOPS operations", "profile", awsProfile)
	}

	// Create SOPS manager
	sopsManager := sops.NewManager(log, sopsConfigPath)

	// Handle validate mode
	if validateOnly {
		return validateSecrets(sopsManager, log, args)
	}

	// Check if a specific path was provided as argument or use patterns
	pathsToProcess := patterns
	if len(args) > 0 {
		// Override patterns with the provided path
		pathsToProcess = []string{args[0]}
	}

	// Set default to current directory if no patterns specified
	if len(pathsToProcess) == 0 {
		pathsToProcess = []string{"."}
	}

	// If we have specific paths to process (not just default ".")
	if len(pathsToProcess) > 1 || pathsToProcess[0] != "." {
		for _, targetPath := range pathsToProcess {
			log.InfoS("Processing path", "path", targetPath)

			// Process the specific path
			if decryptOnly {
				if err := decryptPath(targetPath, sopsManager, log); err != nil {
					log.Error(err, "Failed to decrypt path", "path", targetPath)
					continue
				}
			} else if encryptOnly {
				if err := encryptPath(targetPath, sopsManager, log); err != nil {
					log.Error(err, "Failed to encrypt path", "path", targetPath)
					continue
				}
			} else {
				// Extract and encrypt for the path
				if err := extractPathSecrets(targetPath, log); err != nil {
					log.Error(err, "Failed to extract secrets", "path", targetPath)
					continue
				}
				if !extractOnly {
					if err := encryptPath(targetPath, sopsManager, log); err != nil {
						log.Error(err, "Failed to encrypt secrets", "path", targetPath)
						continue
					}
				}
			}
		}
		return nil
	}

	// Check if config file exists before trying to load it for service-based processing
	configPath := cfgFile
	if configPath == "" {
		configPath = "./config.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file, but if we're in encrypt-only or decrypt-only mode, we can still process current directory
		if encryptOnly || decryptOnly {
			log.V(1).InfoS("Config file not found, processing current directory", "mode", getOperationMode())

			if decryptOnly {
				return decryptPath(".", sopsManager, log)
			} else if encryptOnly {
				return encryptPath(".", sopsManager, log)
			}
		}

		return fmt.Errorf("config file not found at %s - required for service-based extraction", configPath)
	}

	// Load configuration for service-based processing
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine which services to process
	servicesToProcess := getServicesToProcess(cfg, secretsServices)

	// Process each service
	for _, serviceName := range servicesToProcess {
		service, exists := cfg.Services[serviceName]
		if !exists || !service.Enabled {
			continue
		}

		log.InfoS("Processing service", "service", serviceName)

		// Process based on mode
		if decryptOnly {
			if err := decryptServiceSecrets(serviceName, sopsManager, log); err != nil {
				log.Error(err, "Failed to decrypt secrets", "service", serviceName)
				continue
			}
		} else if encryptOnly {
			if err := encryptServiceSecrets(serviceName, sopsManager, log); err != nil {
				log.Error(err, "Failed to encrypt secrets", "service", serviceName)
				continue
			}
		} else {
			// Default: extract and encrypt
			if err := extractServiceSecrets(serviceName, log); err != nil {
				log.Error(err, "Failed to extract secrets", "service", serviceName)
				continue
			}

			if !extractOnly {
				if err := encryptServiceSecrets(serviceName, sopsManager, log); err != nil {
					log.Error(err, "Failed to encrypt secrets", "service", serviceName)
					continue
				}
			}
		}
	}

	log.InfoS("Secrets processing completed")
	return nil
}

// extractServiceSecrets extracts secrets from all environment values files
func extractServiceSecrets(serviceName string, log *logger.NamedLogger) error {
	log.V(2).InfoS("Extracting secrets for service", "service", serviceName)

	// Pattern: apps/{service}/envs/{cluster}/{environment}/{namespace}/values.yaml
	pattern := filepath.Join("apps", serviceName, "envs", "*", "*", "*", "values.yaml")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find values files: %w", err)
	}

	for _, valuesFile := range files {
		// Create secrets file path
		dir := filepath.Dir(valuesFile)
		secretsFile := filepath.Join(dir, "secrets.dec.yaml")

		// Extract secrets
		if err := sops.ExtractSecretsToFile(valuesFile, secretsFile, log); err != nil {
			log.Error(err, "Failed to extract secrets from file", "file", valuesFile)
			continue
		}
	}

	return nil
}

// encryptServiceSecrets encrypts all .dec.yaml files for a service
func encryptServiceSecrets(serviceName string, manager *sops.Manager, log *logger.NamedLogger) error {
	log.V(2).InfoS("Encrypting secrets for service", "service", serviceName)

	// Find all .dec.yaml files
	err := filepath.Walk(filepath.Join("apps", serviceName), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "secrets.dec.yaml") {
			// Create encrypted file path
			encPath := strings.Replace(path, ".dec.yaml", ".enc.yaml", 1)

			// Encrypt the file
			if err := manager.EncryptFile(path, encPath); err != nil {
				log.Error(err, "Failed to encrypt file", "file", path)
			} else {
				// Optionally remove the .dec file after successful encryption
				// os.Remove(path)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	return nil
}

// decryptServiceSecrets decrypts all .enc.yaml files for a service
func decryptServiceSecrets(serviceName string, manager *sops.Manager, log *logger.NamedLogger) error {
	log.V(2).InfoS("Decrypting secrets for service", "service", serviceName)

	// Find all .enc.yaml files
	err := filepath.Walk(filepath.Join("apps", serviceName), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "secrets.enc.yaml") {
			// Create decrypted file path
			decPath := strings.Replace(path, ".enc.yaml", ".dec.yaml", 1)

			// Decrypt the file
			if err := manager.DecryptFile(path, decPath); err != nil {
				log.Error(err, "Failed to decrypt file", "file", path)
			} else {
				// Merge back into values.yaml if needed
				valuesPath := filepath.Join(filepath.Dir(path), "values.yaml")
				if err := sops.MergeSecretsIntoValues(valuesPath, decPath, log); err != nil {
					log.Error(err, "Failed to merge secrets into values", "file", valuesPath)
				}
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	return nil
}

// getOperationMode returns the current operation mode as a string
func getOperationMode() string {
	if validateOnly {
		return "validate"
	} else if decryptOnly {
		return "decrypt-only"
	} else if encryptOnly {
		return "encrypt-only"
	} else if extractOnly {
		return "extract-only"
	}
	return "extract-and-encrypt"
}

// getServicesToProcess determines which services to process
func getServicesToProcess(cfg *config.Config, specified []string) []string {
	if len(specified) > 0 {
		return specified
	}

	// Process all enabled services
	var services []string
	for name, service := range cfg.Services {
		if service.Enabled {
			services = append(services, name)
		}
	}
	return services
}

// extractPathSecrets extracts secrets from values files in a specific path
func extractPathSecrets(targetPath string, log *logger.NamedLogger) error {
	log.V(2).InfoS("Extracting secrets from path", "path", targetPath)

	// Check if it's a file or directory
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		// Process single file
		if strings.HasSuffix(targetPath, "values.yaml") {
			dir := filepath.Dir(targetPath)
			secretsFile := filepath.Join(dir, "secrets.dec.yaml")
			return sops.ExtractSecretsToFile(targetPath, secretsFile, log)
		}
		return fmt.Errorf("file must be values.yaml")
	}

	// Walk directory and find all values.yaml files
	return filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Base(path) == "values.yaml" {
			dir := filepath.Dir(path)
			secretsFile := filepath.Join(dir, "secrets.dec.yaml")
			if err := sops.ExtractSecretsToFile(path, secretsFile, log); err != nil {
				log.Error(err, "Failed to extract secrets", "file", path)
			}
		}

		return nil
	})
}

// encryptPath encrypts .dec files in a specific path using SOPS
func encryptPath(targetPath string, manager *sops.Manager, log *logger.NamedLogger) error {
	log.V(2).InfoS("Encrypting files in path", "path", targetPath)

	// Load SOPS config to get path regex
	sopsConfig := &SOPSConfig{}
	if data, err := os.ReadFile(sopsConfigPath); err == nil {
		if err := yaml.Unmarshal(data, sopsConfig); err == nil && len(sopsConfig.CreationRules) > 0 {
			pathRegex := sopsConfig.CreationRules[0].PathRegex
			if pathRegex != "" {
				regex, err := regexp.Compile(pathRegex)
				if err == nil {
					return encryptWithRegex(targetPath, manager, regex, log)
				}
			}
		}
	}

	// Fallback to .dec.yaml pattern
	return filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".dec.") {
			encPath := strings.Replace(path, ".dec.", ".enc.", 1)
			if err := manager.EncryptFile(path, encPath); err != nil {
				log.Error(err, "Failed to encrypt file", "file", path)
			}
		}

		return nil
	})
}

// encryptWithRegex encrypts files matching the SOPS regex pattern
func encryptWithRegex(targetPath string, manager *sops.Manager, regex *regexp.Regexp, log *logger.NamedLogger) error {
	return filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && regex.MatchString(path) {
			if strings.Contains(filepath.Base(path), ".dec.") {
				encPath := strings.Replace(path, ".dec.", ".enc.", 1)
				if err := manager.EncryptFile(path, encPath); err != nil {
					log.Error(err, "Failed to encrypt file", "file", path)
				}
			}
		}

		return nil
	})
}

// decryptPath decrypts .enc files in a specific path
func decryptPath(targetPath string, manager *sops.Manager, log *logger.NamedLogger) error {
	log.V(2).InfoS("Decrypting files in path", "path", targetPath)

	return filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".enc.") {
			decPath := strings.Replace(path, ".enc.", ".dec.", 1)
			if err := manager.DecryptFile(path, decPath); err != nil {
				log.Error(err, "Failed to decrypt file", "file", path)
			} else {
				// Merge back into values.yaml if needed
				valuesPath := filepath.Join(filepath.Dir(path), "values.yaml")
				if err := sops.MergeSecretsIntoValues(valuesPath, decPath, log); err != nil {
					log.Error(err, "Failed to merge secrets into values", "file", valuesPath)
				}
			}
		}

		return nil
	})
}

// SOPSConfig represents the SOPS configuration
type SOPSConfig struct {
	CreationRules []struct {
		PathRegex string `yaml:"path_regex"`
	} `yaml:"creation_rules"`
}

// validateSecrets validates that all secrets files are properly encrypted
func validateSecrets(manager *sops.Manager, log *logger.NamedLogger, args []string) error {
	log.InfoS("Validating secrets encryption status")

	// Determine search paths
	searchPaths := []string{"."}
	if len(args) > 0 {
		searchPaths = []string{args[0]}
	} else if len(patterns) > 0 && patterns[0] != "." {
		searchPaths = patterns
	}

	var totalFiles, encryptedFiles, decryptedFiles, missingEncFiles int
	var validationErrors []string

	for _, searchPath := range searchPaths {
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Check for .dec files
			if strings.Contains(filepath.Base(path), ".dec.") {
				totalFiles++
				decryptedFiles++

				// Check if corresponding .enc file exists
				encPath := strings.Replace(path, ".dec.", ".enc.", 1)
				if _, err := os.Stat(encPath); os.IsNotExist(err) {
					missingEncFiles++
					validationErrors = append(validationErrors, fmt.Sprintf("Missing encrypted file for: %s", path))
					log.V(1).InfoS("Missing encrypted file", "decrypted", path, "expected", encPath)
				} else {
					// Validate the encrypted file has SOPS metadata
					if err := validateSOPSFile(encPath, log); err != nil {
						validationErrors = append(validationErrors, fmt.Sprintf("Invalid SOPS file %s: %v", encPath, err))
					} else {
						encryptedFiles++
					}
				}
			} else if strings.Contains(filepath.Base(path), ".enc.") {
				// Check for orphaned .enc files
				decPath := strings.Replace(path, ".enc.", ".dec.", 1)
				if _, err := os.Stat(decPath); os.IsNotExist(err) {
					log.V(2).InfoS("Orphaned encrypted file (no .dec counterpart)", "encrypted", path)
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk path %s: %w", searchPath, err)
		}
	}

	// Report results
	log.InfoS("Validation complete",
		"totalDecFiles", decryptedFiles,
		"encryptedFiles", encryptedFiles,
		"missingEncFiles", missingEncFiles,
		"errors", len(validationErrors))

	if len(validationErrors) > 0 {
		log.InfoS("Validation errors found:")
		for _, err := range validationErrors {
			log.InfoS("  - " + err)
		}
		return fmt.Errorf("validation failed: %d issues found", len(validationErrors))
	}

	if decryptedFiles == 0 {
		log.InfoS("No secrets files found to validate")
	} else {
		log.InfoS("✓ All secrets files are properly encrypted")
	}

	return nil
}

// validateSOPSFile checks if a file has valid SOPS encryption markers
func validateSOPSFile(path string, log *logger.NamedLogger) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Check for SOPS metadata markers
	hasSOPSMarker := strings.Contains(content, "sops:") ||
		strings.Contains(content, "ENC[") ||
		strings.Contains(content, "sops_mac")

	if !hasSOPSMarker {
		return fmt.Errorf("no SOPS encryption markers found")
	}

	// Check for specific SOPS fields that should be present
	requiredFields := []string{"sops:", "mac:", "kms:"}
	missingFields := []string{}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		log.V(2).InfoS("File may have incomplete SOPS metadata", "file", path, "missing", missingFields)
	}

	// Check if values are actually encrypted (look for ENC[)
	if !strings.Contains(content, "ENC[") {
		log.V(1).InfoS("Warning: File has SOPS metadata but no encrypted values", "file", path)
	}

	return nil
}

// ExtractSecretsAfterMigration is called from migrate command to extract secrets
func ExtractSecretsAfterMigration(cfg *config.Config, services []string, log *logger.NamedLogger) error {
	log.InfoS("Extracting secrets after migration")

	// Use the same services that were migrated
	servicesToProcess := services
	if len(servicesToProcess) == 0 {
		// If no specific services, use all enabled ones
		for name, service := range cfg.Services {
			if service.Enabled {
				servicesToProcess = append(servicesToProcess, name)
			}
		}
	}

	// Extract secrets for each service
	for _, serviceName := range servicesToProcess {
		if err := extractServiceSecrets(serviceName, log); err != nil {
			log.Error(err, "Failed to extract secrets for service", "service", serviceName)
			// Continue with other services even if one fails
		}
	}

	// Optionally encrypt if SOPS config exists
	if _, err := os.Stat(".sops.yaml"); err == nil {
		log.InfoS("SOPS config found, encrypting secrets")
		sopsManager := sops.NewManager(log, ".sops.yaml")

		for _, serviceName := range servicesToProcess {
			if err := encryptServiceSecrets(serviceName, sopsManager, log); err != nil {
				log.Error(err, "Failed to encrypt secrets for service", "service", serviceName)
			}
		}
	}

	return nil
}
