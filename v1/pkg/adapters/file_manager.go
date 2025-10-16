package adapters

import (
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// FileManager handles file operations
type FileManager interface {
	ReadYAML(path string) (*yaml.NodeTree, error)
	ReadYAMLAsMap(path string) (map[string]interface{}, error)
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

// ReadYAML reads a YAML file and returns a NodeTree
func (f *fileManager) ReadYAML(path string) (*yaml.NodeTree, error) {
	return f.file.ReadYAML(path)
}

// ReadYAMLAsMap reads a YAML file and converts it to a map
func (f *fileManager) ReadYAMLAsMap(path string) (map[string]interface{}, error) {
	nodeTree, err := f.file.ReadYAML(path)
	if err != nil {
		return nil, err
	}

	// Convert NodeTree to map[string]interface{} by marshaling then unmarshaling
	yamlBytes, err := nodeTree.ToYAML()
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
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