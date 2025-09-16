// Package yaml provides utilities for working with YAML files while preserving comments and formatting
package yaml

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/elioetibr/yaml"
)

// init configures the yaml package for optimal processing
func init() {
	// Enable the global setting for preserving blank lines by default
	yaml.PreserveBlankLines = true
}

// ============================================================================
// Core Types and Configuration
// ============================================================================

// Document represents a YAML document with preserved comments and structure
type Document struct {
	node *yaml.Node
}

// Options configures the YAML processing behavior
type Options struct {
	// IndentSize sets the number of spaces for indentation (default: 2)
	IndentSize int
	// PreserveComments preserves comments when loading and saving (default: true)
	PreserveComments bool
	// PreserveEmptyLines preserves empty lines in the document (default: false)
	PreserveEmptyLines bool
}

// CommentPosition specifies where to place a comment relative to a node
type CommentPosition int

const (
	// CommentAbove places the comment on the line(s) above the node
	CommentAbove CommentPosition = iota
	// CommentInline places the comment on the same line as the node
	CommentInline
	// CommentBelow places the comment on the line(s) below the node
	CommentBelow
)

// DefaultOptions returns the default options for YAML processing
func DefaultOptions() *Options {
	// Ensure the global setting is enabled
	yaml.PreserveBlankLines = true
	return &Options{
		IndentSize:         2,
		PreserveComments:   true,
		PreserveEmptyLines: true,
	}
}

// DeepCopyMap creates a deep copy of a map[string]interface{}
func DeepCopyMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	
	result := make(map[string]interface{})
	for key, value := range input {
		result[key] = deepCopyValue(value)
	}
	return result
}

// deepCopyValue recursively copies values
func deepCopyValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return DeepCopyMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = deepCopyValue(item)
		}
		return result
	default:
		// For primitive types, return as-is
		return v
	}
}

// ============================================================================
// Document Loading Functions
// ============================================================================

// LoadFile loads a YAML file preserving comments and structure
func LoadFile(path string, opts *Options) (*Document, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Load(data, opts)
}

// Load parses YAML data preserving comments and structure
func Load(data []byte, opts *Options) (*Document, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &Document{node: &node}, nil
}

// FromMap creates a new Document from a map (no comments)
func FromMap(data map[string]interface{}) (*Document, error) {
	var node yaml.Node
	if err := node.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to encode map: %w", err)
	}

	return &Document{node: &node}, nil
}

// ============================================================================
// Document Saving/Writing Functions
// ============================================================================

// SaveFile saves the document to a file with the specified options
func (d *Document) SaveFile(path string, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	data, err := d.Marshal(opts)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// WriteTo writes the document to the given writer
func (d *Document) WriteTo(w io.Writer, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(opts.IndentSize)
	if opts.PreserveEmptyLines {
		encoder.SetPreserveBlankLines(true)
	}

	if err := encoder.Encode(d.node); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return encoder.Close()
}

// ============================================================================
// Document Marshaling Functions
// ============================================================================

// Marshal converts the document to YAML bytes with the specified options
func (d *Document) Marshal(opts *Options) ([]byte, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(opts.IndentSize)
	if opts.PreserveEmptyLines {
		encoder.SetPreserveBlankLines(true)
	}

	if err := encoder.Encode(d.node); err != nil {
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// ============================================================================
// Document Conversion Functions
// ============================================================================

// GetNode returns the underlying yaml.Node for advanced manipulation
func (d *Document) GetNode() *yaml.Node {
	return d.node
}

// ToMap converts the document to a map[string]interface{} (loses comments)
func (d *Document) ToMap() (map[string]interface{}, error) {
	if d.node == nil {
		return nil, fmt.Errorf("document is nil")
	}

	var result map[string]interface{}
	if err := d.node.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode to map: %w", err)
	}

	return result, nil
}

// ============================================================================
// Document Path-based Operations (Not Yet Implemented)
// ============================================================================

// Get retrieves a value from the document by path (e.g., "spec.containers[0].image")
func (d *Document) Get(path string) (interface{}, error) {
	// This would require implementing a path parser
	// For now, return an error indicating it's not implemented
	return nil, fmt.Errorf("path-based access not yet implemented")
}

// Set sets a value in the document by path, preserving structure
func (d *Document) Set(path string, value interface{}) error {
	// This would require implementing a path parser and node updater
	// For now, return an error indicating it's not implemented
	return fmt.Errorf("path-based setting not yet implemented")
}

// AddComment adds a comment to a specific path in the document
func (d *Document) AddComment(path string, comment string, position CommentPosition) error {
	// This would require implementing path navigation
	return fmt.Errorf("comment addition not yet implemented")
}

// ============================================================================
// Static Marshaling Functions (No Document Required)
// ============================================================================

// Marshal marshals data to YAML with default options (2-space indent)
func Marshal(data interface{}) ([]byte, error) {
	return MarshalWithOptions(data, DefaultOptions())
}

// MarshalWithOptions marshals data to YAML with specified options
func MarshalWithOptions(data interface{}, opts *Options) ([]byte, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(opts.IndentSize)

	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// MarshalNode marshals a yaml.Node with default options (2-space indent)
func MarshalNode(node *yaml.Node) ([]byte, error) {
	return MarshalNodeWithOptions(node, DefaultOptions())
}

// MarshalNodeWithOptions marshals a yaml.Node with specified options
func MarshalNodeWithOptions(node *yaml.Node, opts *Options) ([]byte, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(opts.IndentSize)

	if err := encoder.Encode(node); err != nil {
		return nil, fmt.Errorf("failed to encode YAML node: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// ============================================================================
// Static Unmarshaling Functions (No Document Required)
// ============================================================================

// Unmarshal unmarshals YAML data
func Unmarshal(data []byte, v interface{}) error {
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return nil
}

// UnmarshalStrict unmarshals YAML data strictly (disallowing unknown fields)
func UnmarshalStrict(data []byte, v interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return nil
}

// ============================================================================
// Specialized Document Creation Functions
// ============================================================================

// NewSecretsDecYaml creates a new YAML node structure for secrets.dec.yaml files
// If secretsData is provided, it will be used as the content, otherwise creates empty placeholder
func (d *Document) NewSecretsDecYaml(headComment string, secretsData interface{}) yaml.Node {
	// Create the secrets value node
	var secretsValueNode *yaml.Node
	
	if secretsData != nil {
		// We have actual secrets data, encode it
		secretsValueNode = &yaml.Node{}
		if err := secretsValueNode.Encode(secretsData); err != nil {
			// If encoding fails, fall back to empty mapping
			secretsValueNode = &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{}, // Empty mapping for {}
			}
		}
	} else {
		// No secrets data, create empty mapping placeholder
		secretsValueNode = &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{}, // Empty mapping for {}
		}
	}
	
	return yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind:        yaml.MappingNode,
				HeadComment: headComment,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "secrets"},
					secretsValueNode,
				},
			},
		},
	}
}
