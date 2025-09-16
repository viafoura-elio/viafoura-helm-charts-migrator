package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
)

// sopsService implements SOPSService interface
type sopsService struct {
	config *config.SOPSConfig
	log    *logger.NamedLogger
}

// NewSOPSService creates a new SOPSService
func NewSOPSService(cfg *config.SOPSConfig) SOPSService {
	if cfg == nil {
		cfg = &config.SOPSConfig{
			Enabled:         true,
			ParallelWorkers: 5,
			Timeout:         30,
		}
	}

	return &sopsService{
		config: cfg,
		log:    logger.WithName("sops-service"),
	}
}

// Encrypt encrypts a file using SOPS
func (s *sopsService) Encrypt(filePath string) error {
	if !s.config.Enabled {
		s.log.V(2).InfoS("SOPS encryption disabled", "file", filePath)
		return nil
	}

	// Check if file should be encrypted based on naming convention
	if !s.shouldEncrypt(filePath) {
		s.log.V(3).InfoS("Skipping file, doesn't match encryption pattern", "file", filePath)
		return nil
	}

	// Check if file is already encrypted
	if s.IsEncrypted(filePath) {
		s.log.V(2).InfoS("File already encrypted", "file", filePath)
		return nil
	}

	// Prepare SOPS command
	args := []string{"-e", "-i", filePath}

	// Add config file if specified
	if s.config.ConfigFile != "" {
		args = append(args, "--config", s.config.ConfigFile)
	}

	// Build command
	cmd := exec.Command("sops", args...)

	// Set AWS profile if configured
	if s.config.AwsProfile != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_PROFILE=%s", s.config.AwsProfile))
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	// Execute with context
	output, err := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to encrypt %s: %w\nOutput: %s", filePath, err, output)
	}

	s.log.V(2).InfoS("File encrypted successfully", "file", filePath)
	return nil
}

// Decrypt decrypts a file using SOPS
func (s *sopsService) Decrypt(filePath string) ([]byte, error) {
	if !s.config.Enabled {
		// If SOPS is disabled, just read the file
		return os.ReadFile(filePath)
	}

	// Check if file is encrypted
	if !s.IsEncrypted(filePath) {
		// File is not encrypted, just read it
		return os.ReadFile(filePath)
	}

	// Prepare SOPS command
	args := []string{"-d", filePath}

	// Add config file if specified
	if s.config.ConfigFile != "" {
		args = append(args, "--config", s.config.ConfigFile)
	}

	// Build command
	cmd := exec.Command("sops", args...)

	// Set AWS profile if configured
	if s.config.AwsProfile != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_PROFILE=%s", s.config.AwsProfile))
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	// Execute with context
	output, err := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt %s: %w", filePath, err)
	}

	return output, nil
}

// EncryptBatch encrypts multiple files in parallel
func (s *sopsService) EncryptBatch(filePaths []string, workers int) error {
	if !s.config.Enabled {
		s.log.InfoS("SOPS encryption disabled, skipping batch encryption")
		return nil
	}

	if workers <= 0 {
		workers = s.config.ParallelWorkers
	}
	if workers <= 0 {
		workers = 5 // Default
	}

	// Filter files that need encryption
	var toEncrypt []string
	for _, path := range filePaths {
		if s.shouldEncrypt(path) && !s.IsEncrypted(path) {
			toEncrypt = append(toEncrypt, path)
		}
	}

	if len(toEncrypt) == 0 {
		s.log.InfoS("No files need encryption")
		return nil
	}

	s.log.InfoS("Starting parallel SOPS encryption", "files", len(toEncrypt), "workers", workers)
	startTime := time.Now()

	var (
		wg           sync.WaitGroup
		sem          = make(chan struct{}, workers)
		successCount int32
		failCount    int32
		errors       []error
		errorsMu     sync.Mutex
	)

	for _, filePath := range toEncrypt {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Encrypt file
			if err := s.Encrypt(path); err != nil {
				atomic.AddInt32(&failCount, 1)
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("failed to encrypt %s: %w", path, err))
				errorsMu.Unlock()
				s.log.Error(err, "Failed to encrypt file", "file", path)
			} else {
				atomic.AddInt32(&successCount, 1)
				s.log.V(2).InfoS("Encrypted file", "file", path)
			}
		}(filePath)
	}

	wg.Wait()

	duration := time.Since(startTime)
	s.log.InfoS("Parallel SOPS encryption completed",
		"duration", duration.Round(time.Millisecond),
		"total", len(toEncrypt),
		"success", atomic.LoadInt32(&successCount),
		"failed", atomic.LoadInt32(&failCount),
		"workers", workers)

	// Return first error if any
	if len(errors) > 0 {
		return fmt.Errorf("encryption failed for %d files: %w", len(errors), errors[0])
	}

	return nil
}

// IsEncrypted checks if a file is SOPS encrypted
func (s *sopsService) IsEncrypted(filePath string) bool {
	// Check if file exists
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Check for SOPS metadata in the file
	content := string(data)
	return strings.Contains(content, "sops:") &&
		(strings.Contains(content, "kms:") ||
			strings.Contains(content, "pgp:") ||
			strings.Contains(content, "age:") ||
			strings.Contains(content, "azure_kv:") ||
			strings.Contains(content, "gcp_kms:"))
}

// shouldEncrypt checks if a file should be encrypted based on naming convention
func (s *sopsService) shouldEncrypt(filePath string) bool {
	// Check if it's a .dec.yaml file that should be encrypted
	base := filepath.Base(filePath)

	// Files ending with .dec.yaml or .dec.yml should be encrypted
	if strings.HasSuffix(base, ".dec.yaml") || strings.HasSuffix(base, ".dec.yml") {
		return true
	}

	// Check against configured regex if provided
	if s.config.PathRegex != "" {
		// This would require regex compilation and matching
		// For now, we'll use the simple suffix check
	}

	return false
}
