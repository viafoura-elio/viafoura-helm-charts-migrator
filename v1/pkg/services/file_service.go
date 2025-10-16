package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"helm-charts-migrator/v1/pkg/logger"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// fileService implements FileService interface
type fileService struct {
	log *logger.NamedLogger
}

// NewFileService creates a new FileService
func NewFileService() FileService {
	return &fileService{
		log: logger.WithName("file-service"),
	}
}

// ReadYAML reads a YAML file and returns it as a NodeTree
func (f *fileService) ReadYAML(path string) (*yaml.NodeTree, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Parse with advanced yaml package into NodeTree
	result, err := yaml.UnmarshalYAML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return result, nil
}

// WriteYAML writes data to a YAML file
func (f *fileService) WriteYAML(path string, data interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var yamlBytes []byte
	var err error

	// Check if data is already a NodeTree
	if nodeTree, ok := data.(*yaml.NodeTree); ok {
		// Direct conversion from NodeTree
		yamlBytes, err = nodeTree.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to convert NodeTree to YAML: %w", err)
		}
	} else {
		// Marshal data to YAML using standard yaml package
		yamlBytes, err = yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}

		// Process through the advanced yaml package for proper formatting
		nodeTree, err := yaml.UnmarshalYAML(yamlBytes)
		if err == nil {
			// Successfully parsed, use formatted output
			formattedBytes, err := nodeTree.ToYAML()
			if err == nil {
				yamlBytes = formattedBytes
			}
			// If formatting fails, keep original yamlBytes
		}
		// If parsing fails, keep original yamlBytes
	}

	// Write the YAML
	if err := os.WriteFile(path, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	f.log.V(3).InfoS("Wrote YAML file", "path", path)
	return nil
}

// CopyDirectory copies a directory recursively
func (f *fileService) CopyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source %s is not a directory", src)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	// Walk through source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return f.CopyFile(path, dstPath)
	})
}

// CopyFile copies a single file
func (f *fileService) CopyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", src, err)
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dstDir, err)
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	f.log.V(3).InfoS("Copied file", "src", src, "dst", dst)
	return nil
}

// Exists checks if a file or directory exists
func (f *fileService) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory if it doesn't exist
func (f *fileService) EnsureDir(path string) error {
	if f.Exists(path) {
		return nil
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	f.log.V(3).InfoS("Created directory", "path", path)
	return nil
}

// ListFiles lists files in a directory matching a pattern
func (f *fileService) ListFiles(dir, pattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file matches pattern
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}

		if matched || strings.Contains(filepath.Base(path), pattern) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files in %s: %w", dir, err)
	}

	return files, nil
}
