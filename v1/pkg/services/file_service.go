package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/yaml"
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

// ReadYAML reads a YAML file and returns it as a map
func (f *fileService) ReadYAML(path string) (map[string]interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Use the centralized yaml package to load the file
	doc, err := yaml.LoadFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML from %s: %w", path, err)
	}

	// Convert document to map
	result, err := doc.ToMap()
	if err != nil {
		// If conversion fails, return empty map
		if doc.GetNode() == nil {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to convert YAML to map from %s: %w", path, err)
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

	// Try to preserve existing file structure if it exists
	var doc *yaml.Document
	if _, err := os.Stat(path); err == nil {
		// File exists, try to load it to preserve comments and structure
		existingDoc, loadErr := yaml.LoadFile(path, nil)
		if loadErr == nil {
			// Successfully loaded, we'll preserve its structure
			doc = existingDoc
			// Note: We can't directly update the content while preserving structure
			// So we'll create a new document but at least we tried
		}
	}

	// Create document from the data
	// First check if data is already a map
	if mapData, ok := data.(map[string]interface{}); ok {
		doc, _ = yaml.FromMap(mapData)
	} else {
		// For other types, marshal to YAML then load as document
		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
		doc, err = yaml.Load(yamlBytes, nil)
		if err != nil {
			return fmt.Errorf("failed to create document: %w", err)
		}
	}

	// Save the document
	if err := doc.SaveFile(path, nil); err != nil {
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