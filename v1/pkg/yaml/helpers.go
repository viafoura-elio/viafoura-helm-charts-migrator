package yaml

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elioetibr/yaml"
)

// FindNode finds a node at the given path in the document
// Path format: "key1.key2[0].key3" for nested structures
func (d *Document) FindNode(path string) (*yaml.Node, error) {
	if d.node == nil {
		return nil, fmt.Errorf("document is nil")
	}

	// Start from the document root
	current := d.node
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	segments := parsePath(path)
	for _, segment := range segments {
		next, err := findChild(current, segment)
		if err != nil {
			return nil, fmt.Errorf("failed to find path %s: %w", path, err)
		}
		current = next
	}

	return current, nil
}

// SetValue sets a scalar value at the given path
func (d *Document) SetValue(path string, value interface{}) error {
	node, err := d.FindNode(path)
	if err != nil {
		return err
	}

	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("path %s does not point to a scalar value", path)
	}

	// Convert value to string
	node.Value = fmt.Sprintf("%v", value)
	return nil
}

// GetValue retrieves a scalar value at the given path
func (d *Document) GetValue(path string) (string, error) {
	node, err := d.FindNode(path)
	if err != nil {
		return "", err
	}

	if node.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("path %s does not point to a scalar value", path)
	}

	return node.Value, nil
}

// SetComment sets a comment for the node at the given path
func (d *Document) SetComment(path string, comment string, position CommentPosition) error {
	node, err := d.FindNode(path)
	if err != nil {
		return err
	}

	switch position {
	case CommentAbove:
		node.HeadComment = comment
	case CommentInline:
		node.LineComment = comment
	case CommentBelow:
		node.FootComment = comment
	default:
		return fmt.Errorf("invalid comment position")
	}

	return nil
}

// GetComment retrieves a comment from the node at the given path
func (d *Document) GetComment(path string, position CommentPosition) (string, error) {
	node, err := d.FindNode(path)
	if err != nil {
		return "", err
	}

	switch position {
	case CommentAbove:
		return node.HeadComment, nil
	case CommentInline:
		return node.LineComment, nil
	case CommentBelow:
		return node.FootComment, nil
	default:
		return "", fmt.Errorf("invalid comment position")
	}
}

// RemoveKey removes a key from a mapping node
func (d *Document) RemoveKey(path string) error {
	// Parse parent path and key
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return fmt.Errorf("cannot remove root node")
	}

	parentPath := path[:lastDot]
	key := path[lastDot+1:]

	// Find parent node
	parentNode, err := d.FindNode(parentPath)
	if err != nil {
		// If parent path is empty, use root
		if parentPath == "" {
			parentNode = d.node
			if parentNode.Kind == yaml.DocumentNode && len(parentNode.Content) > 0 {
				parentNode = parentNode.Content[0]
			}
			key = path
		} else {
			return err
		}
	}

	if parentNode.Kind != yaml.MappingNode {
		return fmt.Errorf("parent is not a mapping node")
	}

	// Find and remove the key-value pair
	newContent := make([]*yaml.Node, 0, len(parentNode.Content))
	for i := 0; i < len(parentNode.Content); i += 2 {
		if i+1 >= len(parentNode.Content) {
			break
		}
		if parentNode.Content[i].Kind == yaml.ScalarNode && parentNode.Content[i].Value != key {
			newContent = append(newContent, parentNode.Content[i], parentNode.Content[i+1])
		}
	}

	parentNode.Content = newContent
	return nil
}

// HasKey checks if a key exists at the given path
func (d *Document) HasKey(path string) bool {
	_, err := d.FindNode(path)
	return err == nil
}

// pathSegment represents a segment in a path
type pathSegment struct {
	key   string
	index int // -1 if not an array index
}

// parsePath parses a path string into segments
func parsePath(path string) []pathSegment {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	segments := make([]pathSegment, 0, len(parts))

	for _, part := range parts {
		// Check for array index
		if idx := strings.Index(part, "["); idx != -1 {
			if idx > 0 {
				// Add the key part
				segments = append(segments, pathSegment{key: part[:idx], index: -1})
			}

			// Parse array indices
			remaining := part[idx:]
			for strings.HasPrefix(remaining, "[") {
				end := strings.Index(remaining, "]")
				if end == -1 {
					break
				}

				indexStr := remaining[1:end]
				if index, err := strconv.Atoi(indexStr); err == nil {
					segments = append(segments, pathSegment{key: "", index: index})
				}

				remaining = remaining[end+1:]
			}
		} else {
			segments = append(segments, pathSegment{key: part, index: -1})
		}
	}

	return segments
}

// findChild finds a child node based on the segment
func findChild(node *yaml.Node, segment pathSegment) (*yaml.Node, error) {
	if segment.index >= 0 {
		// Array index access
		if node.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("expected sequence node, got %v", node.Kind)
		}
		if segment.index >= len(node.Content) {
			return nil, fmt.Errorf("index %d out of bounds", segment.index)
		}
		return node.Content[segment.index], nil
	}

	// Key access
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node for key %s, got %v", segment.key, node.Kind)
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == segment.key {
			return node.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("key %s not found", segment.key)
}

// IsCommented checks if the document or a specific path has comments
func (d *Document) IsCommented(path string) (bool, error) {
	if path == "" {
		// Check root document
		return d.node.HeadComment != "" || d.node.LineComment != "" || d.node.FootComment != "", nil
	}

	node, err := d.FindNode(path)
	if err != nil {
		return false, err
	}

	return node.HeadComment != "" || node.LineComment != "" || node.FootComment != "", nil
}

// StripComments removes all comments from the document
func (d *Document) StripComments() {
	if d.node != nil {
		stripNodeComments(d.node)
	}
}

// stripNodeComments recursively removes comments from a node
func stripNodeComments(node *yaml.Node) {
	if node == nil {
		return
	}

	node.HeadComment = ""
	node.LineComment = ""
	node.FootComment = ""

	for _, child := range node.Content {
		stripNodeComments(child)
	}
}

// Clone creates a deep copy of the document
func (d *Document) Clone() *Document {
	if d.node == nil {
		return &Document{}
	}
	return &Document{node: copyNode(d.node)}
}
