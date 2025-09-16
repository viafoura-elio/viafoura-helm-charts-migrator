package sops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elioetibr/yaml"
	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"

	"helm-charts-migrator/v1/pkg/logger"
)

// Manager handles SOPS encryption and decryption operations
type Manager struct {
	log        *logger.NamedLogger
	configPath string
}

// NewManager creates a new SOPS manager
func NewManager(log *logger.NamedLogger, configPath string) *Manager {
	if configPath == "" {
		configPath = ".sops.yaml"
	}
	return &Manager{
		log:        log,
		configPath: configPath,
	}
}

// EncryptFile encrypts a file using SOPS
func (m *Manager) EncryptFile(inputPath, outputPath string) error {
	m.log.V(2).InfoS("Encrypting file", "input", inputPath, "output", outputPath)

	// Read input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Load SOPS config
	conf, err := config.LoadCreationRuleForFile(m.configPath, outputPath, nil)
	if err != nil {
		// If no config found, try with input path
		conf, err = config.LoadCreationRuleForFile(m.configPath, inputPath, nil)
		if err != nil {
			return fmt.Errorf("failed to load SOPS config for file: %w", err)
		}
	}

	// Get the appropriate store based on file extension
	storeConfig := config.StoresConfig{}
	store := common.DefaultStoreForPath(&storeConfig, outputPath)

	// Create branches for encryption
	branches, err := store.LoadPlainFile(data)
	if err != nil {
		return fmt.Errorf("failed to load plain file: %w", err)
	}

	// Create sops.Tree for encryption
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:         conf.KeyGroups,
			ShamirThreshold:   conf.ShamirThreshold,
			UnencryptedRegex:  conf.UnencryptedRegex,
			EncryptedRegex:    conf.EncryptedRegex,
			UnencryptedSuffix: conf.UnencryptedSuffix,
			EncryptedSuffix:   conf.EncryptedSuffix,
			MACOnlyEncrypted:  conf.MACOnlyEncrypted,
			Version:           "3.10.2",
		},
		FilePath: outputPath,
	}

	// Generate data key
	dataKey, errs := tree.GenerateDataKey()
	if len(errs) > 0 {
		return fmt.Errorf("failed to generate data key: %v", errs)
	}

	// Encrypt the tree
	cipher := aes.NewCipher()
	_, err = tree.Encrypt(dataKey, cipher)
	if err != nil {
		return fmt.Errorf("failed to encrypt tree: %w", err)
	}
	// Note: The MAC is handled internally by the store when emitting

	// Emit encrypted file
	encryptedData, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return fmt.Errorf("failed to emit encrypted file: %w", err)
	}

	// Write encrypted file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, encryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	m.log.V(1).InfoS("File encrypted successfully", "output", outputPath)
	return nil
}

// DecryptFile decrypts a SOPS encrypted file
func (m *Manager) DecryptFile(inputPath, outputPath string) error {
	m.log.V(2).InfoS("Decrypting file", "input", inputPath, "output", outputPath)

	// Read encrypted file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Decrypt the data
	decryptedData, err := decrypt.Data(data, "yaml")
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	// Write decrypted file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, decryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	m.log.V(1).InfoS("File decrypted successfully", "output", outputPath)
	return nil
}

// ExtractSecretsToFile extracts secrets from a values file and saves them to a separate file
func ExtractSecretsToFile(valuesPath, secretsPath string, log *logger.NamedLogger) error {
	log.V(2).InfoS("Extracting secrets from values", "input", valuesPath, "output", secretsPath)

	// Read values file
	data, err := os.ReadFile(valuesPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.V(3).InfoS("Values file does not exist, skipping", "path", valuesPath)
			return nil
		}
		return fmt.Errorf("failed to read values file: %w", err)
	}

	// Parse YAML
	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("failed to parse values YAML: %w", err)
	}

	// Extract secrets key
	secrets, exists := values["secrets"]
	if !exists {
		log.V(3).InfoS("No secrets key found in values", "path", valuesPath)
		return nil
	}

	// Remove secrets from original values
	delete(values, "secrets")

	// Save secrets to separate file
	secretsData, err := yaml.Marshal(map[string]interface{}{"secrets": secrets})
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Check if this is a hierarchical secrets file and determine its level
	// Using simple string matching for KISS principle
	secretsLevel := ""
	if filepath.Base(secretsPath) == "secrets.dec.yaml" {
		// Convert to forward slashes for consistent matching
		normalizedPath := filepath.ToSlash(secretsPath)
		// Check if path matches hierarchical patterns: apps/*/envs/*
		// Handle both absolute and relative paths
		if (strings.Contains(normalizedPath, "/apps/") || strings.HasPrefix(normalizedPath, "apps/")) &&
			strings.Contains(normalizedPath, "/envs/") {
			// Split path to check structure
			parts := strings.Split(normalizedPath, "/")
			// Find where "apps" starts
			for i := 0; i < len(parts); i++ {
				if parts[i] == "apps" && i+2 < len(parts) && parts[i+2] == "envs" {
					// Determine the level based on path structure after "envs"
					// Pattern variations:
					// - apps/{service}/envs/{cluster}/secrets.dec.yaml -> cluster level (i+4 parts total)
					// - apps/{service}/envs/{cluster}/{environment}/secrets.dec.yaml -> environment level (i+5 parts total)
					// - apps/{service}/envs/{cluster}/{environment}/{namespace}/secrets.dec.yaml -> namespace level (i+6 parts total)

					if parts[len(parts)-1] == "secrets.dec.yaml" {
						pathLength := len(parts)
						if pathLength == i+5 { // cluster level
							secretsLevel = "cluster"
						} else if pathLength == i+6 { // environment level
							secretsLevel = "environment"
						} else if pathLength == i+7 { // namespace level
							secretsLevel = "namespace"
						}
						break
					}
				}
			}
		}
	}

	// Prepare the final data with appropriate comment based on level
	var finalData []byte
	switch secretsLevel {
	case "cluster":
		comment := "# Placeholder to cluster level secrets\n"
		finalData = append([]byte(comment), secretsData...)
	case "environment":
		comment := "# Placeholder to environment level secrets\n"
		finalData = append([]byte(comment), secretsData...)
	case "namespace":
		comment := "# Placeholder to namespace level secrets.\n# It can override any previous secrets as needed.\n"
		finalData = append([]byte(comment), secretsData...)
	default:
		finalData = secretsData
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(secretsPath), 0755); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Write secrets file
	if err := os.WriteFile(secretsPath, finalData, 0644); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	// Update original values file without secrets
	valuesData, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal updated values: %w", err)
	}

	if err := os.WriteFile(valuesPath, valuesData, 0644); err != nil {
		return fmt.Errorf("failed to update values file: %w", err)
	}

	log.V(1).InfoS("Secrets extracted successfully", "secretsFile", secretsPath)
	return nil
}

// MergeSecretsIntoValues merges secrets from a separate file back into the values file
func MergeSecretsIntoValues(valuesPath, secretsPath string, log *logger.NamedLogger) error {
	log.V(2).InfoS("Merging secrets into values", "values", valuesPath, "secrets", secretsPath)

	// Read values file
	valuesData, err := os.ReadFile(valuesPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read values file: %w", err)
	}

	var values map[string]interface{}
	if len(valuesData) > 0 {
		if err := yaml.Unmarshal(valuesData, &values); err != nil {
			return fmt.Errorf("failed to parse values YAML: %w", err)
		}
	} else {
		values = make(map[string]interface{})
	}

	// Read secrets file
	secretsData, err := os.ReadFile(secretsPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.V(3).InfoS("Secrets file does not exist, skipping", "path", secretsPath)
			return nil
		}
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	// Parse secrets
	var secretsWrapper map[string]interface{}
	if err := yaml.Unmarshal(secretsData, &secretsWrapper); err != nil {
		return fmt.Errorf("failed to parse secrets YAML: %w", err)
	}

	// Merge secrets into values
	if secrets, ok := secretsWrapper["secrets"]; ok {
		values["secrets"] = secrets
	}

	// Write merged values
	mergedData, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal merged values: %w", err)
	}

	if err := os.WriteFile(valuesPath, mergedData, 0644); err != nil {
		return fmt.Errorf("failed to write merged values: %w", err)
	}

	log.V(1).InfoS("Secrets merged successfully", "valuesFile", valuesPath)
	return nil
}

// AddCommentsToExistingSecrets adds appropriate hierarchical comments to existing secrets.dec.yaml files
func AddCommentsToExistingSecrets(secretsPath string, log *logger.NamedLogger) error {
	log.V(2).InfoS("Adding comments to existing secrets file", "path", secretsPath)

	// Read the existing file
	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	// Check if file already has a comment (starts with #)
	if len(data) > 0 && data[0] == '#' {
		log.V(3).InfoS("File already has comments, skipping", "path", secretsPath)
		return nil
	}

	// Determine the hierarchical level
	secretsLevel := ""
	if filepath.Base(secretsPath) == "secrets.dec.yaml" {
		normalizedPath := filepath.ToSlash(secretsPath)
		if (strings.Contains(normalizedPath, "/apps/") || strings.HasPrefix(normalizedPath, "apps/")) &&
			strings.Contains(normalizedPath, "/envs/") {
			parts := strings.Split(normalizedPath, "/")
			for i := 0; i < len(parts); i++ {
				if parts[i] == "apps" && i+2 < len(parts) && parts[i+2] == "envs" {
					pathLength := len(parts)
					if pathLength == i+5 {
						secretsLevel = "cluster"
					} else if pathLength == i+6 {
						secretsLevel = "environment"
					} else if pathLength == i+7 {
						secretsLevel = "namespace"
					}
					break
				}
			}
		}
	}

	// Prepare the comment based on level
	var comment string
	switch secretsLevel {
	case "cluster":
		comment = "# Placeholder to cluster level secrets\n"
	case "environment":
		comment = "# Placeholder to environment level secrets\n"
	case "namespace":
		comment = "# Placeholder to namespace level secrets.\n# It can override any previous secrets as needed.\n"
	default:
		// No comment for non-hierarchical files
		return nil
	}

	// Combine comment with existing content
	finalData := append([]byte(comment), data...)

	// Write back the file
	if err := os.WriteFile(secretsPath, finalData, 0644); err != nil {
		return fmt.Errorf("failed to write updated secrets file: %w", err)
	}

	log.V(1).InfoS("Added comments to secrets file", "path", secretsPath, "level", secretsLevel)
	return nil
}

// FindSecretsFiles finds all secrets files matching the pattern
func FindSecretsFiles(rootPath string, pattern string, log *logger.NamedLogger) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file matches pattern
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if matched {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}
