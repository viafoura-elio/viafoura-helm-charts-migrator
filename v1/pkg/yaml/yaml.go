// Package yaml provides utilities for working with YAML files while preserving comments and formatting
package yaml

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

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

	// Ensure proper formatting before marshaling
	d.ensureProperFormatting()

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
	if opts.PreserveEmptyLines {
		encoder.SetPreserveBlankLines(true)
	}

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
	if opts.PreserveEmptyLines {
		encoder.SetPreserveBlankLines(true)
	}

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
// Node Merging Functions
// ============================================================================

// MergeYAMLNodes merges two YAML mapping nodes, with values from source overriding dest
// while preserving comments from both nodes intelligently
func MergeYAMLNodes(dest, source *yaml.Node) {
	if dest.Kind != yaml.MappingNode || source.Kind != yaml.MappingNode {
		return
	}

	// Iterate through source key-value pairs
	for i := 0; i < len(source.Content); i += 2 {
		keyNode := source.Content[i]
		valueNode := source.Content[i+1]

		// Find matching key in destination
		found := false
		for j := 0; j < len(dest.Content); j += 2 {
			destKeyNode := dest.Content[j]
			if destKeyNode.Value == keyNode.Value {
				// Key exists, merge or replace value
				destValueNode := dest.Content[j+1]

				// If both values are mappings, merge recursively
				if destValueNode.Kind == yaml.MappingNode && valueNode.Kind == yaml.MappingNode {
					MergeYAMLNodes(destValueNode, valueNode)
				} else {
					// Replace the value but preserve comments intelligently
					newNode := *valueNode // Create a copy

					// Reset line/column to allow proper formatting
					// These will be set correctly by the encoder
					newNode.Line = 0
					newNode.Column = 0

					// If source has a comment, use it; otherwise keep destination's comment
					if valueNode.HeadComment != "" {
						newNode.HeadComment = valueNode.HeadComment
					} else if destValueNode.HeadComment != "" {
						newNode.HeadComment = destValueNode.HeadComment
					}
					if valueNode.LineComment != "" {
						newNode.LineComment = valueNode.LineComment
					} else if destValueNode.LineComment != "" {
						newNode.LineComment = destValueNode.LineComment
					}
					if valueNode.FootComment != "" {
						newNode.FootComment = valueNode.FootComment
					} else if destValueNode.FootComment != "" {
						newNode.FootComment = destValueNode.FootComment
					}
					dest.Content[j+1] = &newNode
				}
				// Also update key comments if source has them
				if keyNode.HeadComment != "" || keyNode.LineComment != "" || keyNode.FootComment != "" {
					newKeyNode := *keyNode
					// Reset line/column for proper formatting
					newKeyNode.Line = 0
					newKeyNode.Column = 0
					dest.Content[j] = &newKeyNode
				}
				found = true
				break
			}
		}

		// If key not found, append it with all its comments
		if !found {
			// Create copies and reset line/column for proper formatting
			newKeyNode := *keyNode
			newValueNode := *valueNode
			newKeyNode.Line = 0
			newKeyNode.Column = 0
			newValueNode.Line = 0
			newValueNode.Column = 0
			dest.Content = append(dest.Content, &newKeyNode, &newValueNode)
		}
	}
}

// MergeWith MergeDocuments merges two YAML documents, with values from source overriding dest
// while preserving comments and structure
func (d *Document) MergeWith(source *Document) error {
	if d.node == nil || source.node == nil {
		return fmt.Errorf("cannot merge nil documents")
	}

	// Both nodes should have Document nodes as their first content
	if len(d.node.Content) == 0 || len(source.node.Content) == 0 {
		return fmt.Errorf("invalid YAML structure for merging")
	}

	// Get the root mapping nodes
	destRoot := d.node.Content[0]
	sourceRoot := source.node.Content[0]

	// Preserve head comments from source if they exist
	if sourceRoot.HeadComment != "" {
		destRoot.HeadComment = sourceRoot.HeadComment
	}

	// Merge source into dest
	MergeYAMLNodes(destRoot, sourceRoot)

	// Ensure proper formatting after merge to prevent line breaks
	d.ensureProperFormatting()

	return nil
}

// ============================================================================
// Comment Generation Functions
// ============================================================================

// SecretsHeadComment generates appropriate head comments for secrets.dec.yaml files
// based on their hierarchical path in the configuration structure
func SecretsHeadComment(path string) string {
	// Helper function to generate the comment with DRY principle
	generateComment := func(level string, override string) string {
		base := fmt.Sprintf("# Placeholder to %s secrets level.\n", level)
		cascade := "# Using Hierarchical configurations it will be cascaded to lower secrets.\n"
		overrideMsg := fmt.Sprintf("# It can override %s.", override)
		return base + cascade + overrideMsg
	}

	// Count the number of path segments to determine the hierarchy level
	segments := strings.Split(strings.TrimSuffix(path, "/secrets.dec.yaml"), "/")
	segmentCount := len(segments)

	// Check for specific patterns based on path depth
	// Pattern: apps/{service}/envs/...
	if strings.HasPrefix(path, "apps/") && len(segments) >= 3 && segments[2] == "envs" {
		switch segmentCount {
		case 3:
			// apps/{service}/envs/
			return generateComment("global", "the secrets on root values.yaml")
		case 4:
			// apps/{service}/envs/{cluster}
			return generateComment("cluster", "any previous secrets.dec.yaml file")
		case 5:
			// apps/{service}/envs/{cluster}/{environment}
			return generateComment("environment", "any previous secrets.dec.yaml file")
		case 6:
			// apps/{service}/envs/{cluster}/{environment}/{namespace}
			return generateComment("namespace", "any previous secrets.dec.yaml file")
		}
	}

	// Default comment for unmatched patterns
	return "# Configuration secrets file"
}

// ============================================================================
// Document Formatting Functions
// ============================================================================

// ensureProperFormatting ensures proper formatting for all nodes in the document
func (d *Document) ensureProperFormatting() {
	if d.node != nil {
		// First reset all line/column information to let encoder set it properly
		resetNodePositions(d.node)
		// Then ensure proper styles
		ensureProperNodeFormatting(d.node)
	}
}

// resetNodePositions resets line and column information in all nodes
func resetNodePositions(node *yaml.Node) {
	if node == nil {
		return
	}

	// Reset this node's position
	node.Line = 0
	node.Column = 0

	// Recursively reset children
	for _, child := range node.Content {
		resetNodePositions(child)
	}
}

// ensureProperNodeFormatting recursively ensures proper formatting for a node and its children
func ensureProperNodeFormatting(node *yaml.Node) {
	ensureProperNodeFormattingWithContext(node, false)
}

// ensureProperNodeFormattingWithContext recursively ensures proper formatting for a node and its children
// with context about whether the node is a key in a mapping
func ensureProperNodeFormattingWithContext(node *yaml.Node, isKey bool) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Process document content
		for _, child := range node.Content {
			ensureProperNodeFormattingWithContext(child, false)
		}

	case yaml.MappingNode:
		// Use block style for mappings to prevent inline formatting
		if len(node.Content) > 0 {
			node.Style = 0 // Block style
		}
		// Process all key-value pairs (keys at even indices, values at odd indices)
		for i, child := range node.Content {
			isChildKey := (i%2 == 0) // Even indices are keys, odd indices are values
			ensureProperNodeFormattingWithContext(child, isChildKey)
		}

	case yaml.SequenceNode:
		// Use block style for sequences
		if len(node.Content) > 0 {
			node.Style = 0 // Block style
		}
		// Process all sequence items (none are keys)
		for _, child := range node.Content {
			ensureProperNodeFormattingWithContext(child, false)
		}

	case yaml.ScalarNode:
		// Keys should generally not be quoted unless absolutely necessary
		if isKey {
			// For keys, only quote if absolutely necessary
			if len(node.Value) > 0 {
				firstChar := node.Value[0]
				// Only quote keys if they contain special YAML characters that would break parsing
				if firstChar == '-' || firstChar == '?' || firstChar == ':' ||
				   firstChar == '@' || firstChar == '`' || firstChar == '|' ||
				   firstChar == '>' || firstChar == '{' || firstChar == '[' ||
				   firstChar == '*' || firstChar == '&' || firstChar == '!' ||
				   firstChar == '%' || firstChar == '\\' || firstChar == '"' ||
				   firstChar == '\'' || strings.ContainsAny(node.Value, ":#") ||
				   strings.HasPrefix(node.Value, " ") || strings.HasSuffix(node.Value, " ") {
					node.Style = yaml.DoubleQuotedStyle
				} else {
					// Keep keys unquoted by default
					node.Style = 0
				}
			} else {
				// Empty keys need quotes
				node.Style = yaml.DoubleQuotedStyle
			}
		} else {
			// For values, ensure they are not broken across lines
			// Check if the value contains actual newlines (multi-line content)
			if strings.Contains(node.Value, "\n") {
				// Multi-line strings should use literal style (block style)
				node.Style = yaml.LiteralStyle // literal block scalar |
			} else if node.Style == 0 {
				// For single-line values, prevent line breaks by using appropriate style
				// The issue is that style 0 (default) allows the encoder to break lines

				// Check if value needs quotes
				needsQuotes := false

				// Check for YAML special characters at the start
				if len(node.Value) > 0 {
					firstChar := node.Value[0]
					if firstChar == '-' || firstChar == '?' || firstChar == ':' ||
					   firstChar == '@' || firstChar == '`' || firstChar == '|' ||
					   firstChar == '>' || firstChar == '{' || firstChar == '[' ||
					   firstChar == '*' || firstChar == '&' || firstChar == '!' ||
					   firstChar == '%' || firstChar == '\\' || firstChar == '"' ||
					   firstChar == '\'' {
						needsQuotes = true
					}
				}

				// Check if it's already tagged as a boolean
				if node.Tag == "!!bool" {
					// It's a boolean, keep it unquoted
					node.Style = 0
				} else {
					// Check for special values that look like booleans or numbers
					lowerVal := strings.ToLower(node.Value)
					if lowerVal == "true" || lowerVal == "false" || lowerVal == "yes" ||
					   lowerVal == "no" || lowerVal == "on" || lowerVal == "off" ||
					   lowerVal == "null" || lowerVal == "~" {
						// Check if the tag indicates it's actually a boolean
						if node.Tag == "" || node.Tag == "!!str" {
							// It's a string that looks like a boolean, needs quotes
							needsQuotes = true
						} else {
							// It's actually a boolean/null, don't quote
							node.Style = 0
						}
					}

					// Check if it's a number
					if _, err := strconv.ParseFloat(node.Value, 64); err == nil {
						// It's a number, doesn't need quotes but should stay on one line
						// Keep style as 0 for numbers
						node.Style = 0
					} else if needsQuotes || strings.ContainsAny(node.Value, ":#") ||
						strings.HasPrefix(node.Value, " ") || strings.HasSuffix(node.Value, " ") {
						// Use double quotes for strings that need protection
						node.Style = yaml.DoubleQuotedStyle
					} else if node.Style == yaml.DoubleQuotedStyle {
						// Preserve DoubleQuotedStyle if it already exists
						// Keep it as is
					} else {
						// For simple strings, preserve original style if set
						// If style is 0 (default), keep it that way to allow natural formatting
						// The encoder should handle simple values properly
						// node.Style remains unchanged
					}
				}
			}
		}
	}
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
