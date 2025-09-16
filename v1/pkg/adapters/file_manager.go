package adapters

import (
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
)

// FileManager handles file operations
type FileManager interface {
	ReadYAML(path string) (map[string]interface{}, error)
	WriteYAML(path string, data interface{}) error
	CopyFile(src, dst string) error
	EnsureDir(path string) error
}

// fileManager implements FileManager
type fileManager struct {
	file services.FileService
	log  *logger.NamedLogger
}

// NewFileManager creates a new FileManager
func NewFileManager(file services.FileService) FileManager {
	return &fileManager{
		file: file,
		log:  logger.WithName("file-manager"),
	}
}

// ReadYAML reads a YAML file
func (f *fileManager) ReadYAML(path string) (map[string]interface{}, error) {
	return f.file.ReadYAML(path)
}

// WriteYAML writes data to a YAML file
func (f *fileManager) WriteYAML(path string, data interface{}) error {
	return f.file.WriteYAML(path, data)
}

// CopyFile copies a file
func (f *fileManager) CopyFile(src, dst string) error {
	return f.file.CopyFile(src, dst)
}

// EnsureDir ensures directory exists
func (f *fileManager) EnsureDir(path string) error {
	return f.file.EnsureDir(path)
}