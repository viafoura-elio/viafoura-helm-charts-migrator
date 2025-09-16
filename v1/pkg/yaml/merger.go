package yaml

import (
	"fmt"
	"strings"

	"github.com/elioetibr/yaml"
)

// MergeStrategy defines how values should be merged
type MergeStrategy int

const (
	// MergeOverwrite replaces destination values with source values
	MergeOverwrite MergeStrategy = iota
	// MergeDeep recursively merges maps and replaces other types
	MergeDeep
	// MergeAppend appends source arrays to destination arrays
	MergeAppend
)

// MergeOptions configures the merge behavior
type MergeOptions struct {
	// Strategy defines how to merge values
	Strategy MergeStrategy
	// PreferSourceComments uses comments from source when available
	PreferSourceComments bool
	// KeepDestinationComments preserves destination comments when source has none
	KeepDestinationComments bool
}

// DefaultMergeOptions returns default merge options
func DefaultMergeOptions() *MergeOptions {
	return &MergeOptions{
		Strategy:                MergeDeep,
		PreferSourceComments:    true,
		KeepDestinationComments: true,
	}
}

// Merge merges two documents according to the specified options
func Merge(dst, src *Document, opts *MergeOptions) (*Document, error) {
	if dst == nil {
		return src, nil
	}
	if src == nil {
		return dst, nil
	}
	if opts == nil {
		opts = DefaultMergeOptions()
	}

	mergedNode := mergeNodes(dst.node, src.node, opts)

	// Ensure all complex structures use block style to prevent inline formatting
	ensureBlockStyle(mergedNode)

	return &Document{node: mergedNode}, nil
}

// MergeFiles loads and merges multiple YAML files
func MergeFiles(paths []string, opts *MergeOptions) (*Document, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	// Load the first file as base
	result, err := LoadFile(paths[0], nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load base file %s: %w", paths[0], err)
	}

	// Merge remaining files
	for i := 1; i < len(paths); i++ {
		doc, err := LoadFile(paths[i], nil)
		if err != nil {
			return nil, fmt.Errorf("failed to load file %s: %w", paths[i], err)
		}

		result, err = Merge(result, doc, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to merge file %s: %w", paths[i], err)
		}
	}

	return result, nil
}

// mergeNodes recursively merges two yaml.Node structures
func mergeNodes(dst, src *yaml.Node, opts *MergeOptions) *yaml.Node {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}

	// Handle document nodes
	if dst.Kind == yaml.DocumentNode && src.Kind == yaml.DocumentNode {
		result := &yaml.Node{
			Kind:        yaml.DocumentNode,
			HeadComment: mergeComment(dst.HeadComment, src.HeadComment, opts),
			LineComment: mergeComment(dst.LineComment, src.LineComment, opts),
			FootComment: mergeComment(dst.FootComment, src.FootComment, opts),
		}

		if len(dst.Content) > 0 && len(src.Content) > 0 {
			result.Content = []*yaml.Node{mergeNodes(dst.Content[0], src.Content[0], opts)}
		} else if len(src.Content) > 0 {
			result.Content = src.Content
		} else {
			result.Content = dst.Content
		}

		return result
	}

	// Handle mapping nodes (objects)
	if dst.Kind == yaml.MappingNode && src.Kind == yaml.MappingNode {
		return mergeMappingNodes(dst, src, opts)
	}

	// Handle sequence nodes (arrays)
	if dst.Kind == yaml.SequenceNode && src.Kind == yaml.SequenceNode {
		if opts.Strategy == MergeAppend {
			return appendSequenceNodes(dst, src, opts)
		}
		// For other strategies, replace with source
		return copyNodeWithComments(src, dst, opts)
	}

	// For scalar nodes or type mismatches, use source
	return copyNodeWithComments(src, dst, opts)
}

// mergeMappingNodes merges two mapping (object) nodes
func mergeMappingNodes(dst, src *yaml.Node, opts *MergeOptions) *yaml.Node {
	result := &yaml.Node{
		Kind:        yaml.MappingNode,
		Style:       dst.Style,
		HeadComment: mergeComment(dst.HeadComment, src.HeadComment, opts),
		LineComment: mergeComment(dst.LineComment, src.LineComment, opts),
		FootComment: mergeComment(dst.FootComment, src.FootComment, opts),
		Content:     make([]*yaml.Node, 0),
		Line:        dst.Line,   // Preserve line information
		Column:      dst.Column, // Preserve column information
	}

	// Create a map of source keys for quick lookup
	srcMap := make(map[string]*yaml.Node)
	srcValues := make(map[string]*yaml.Node)
	for i := 0; i < len(src.Content); i += 2 {
		if i+1 < len(src.Content) && src.Content[i].Kind == yaml.ScalarNode {
			srcMap[src.Content[i].Value] = src.Content[i]
			srcValues[src.Content[i].Value] = src.Content[i+1]
		}
	}

	// Process destination keys
	processed := make(map[string]bool)
	for i := 0; i < len(dst.Content); i += 2 {
		if i+1 >= len(dst.Content) || dst.Content[i].Kind != yaml.ScalarNode {
			continue
		}

		key := dst.Content[i].Value
		dstKey := dst.Content[i]
		dstValue := dst.Content[i+1]

		if srcKey, exists := srcMap[key]; exists {
			// Key exists in both - merge values
			processed[key] = true

			// Merge key comments
			mergedHeadComment := mergeComment(dstKey.HeadComment, srcKey.HeadComment, opts)

			// Ensure @schema blocks have a blank line before them
			if strings.Contains(mergedHeadComment, "# @schema") && !strings.HasPrefix(mergedHeadComment, "\n") {
				mergedHeadComment = "\n" + mergedHeadComment
			}

			mergedKey := &yaml.Node{
				Kind:        dstKey.Kind,
				Style:       dstKey.Style,
				Tag:         dstKey.Tag,
				Value:       dstKey.Value,
				HeadComment: mergedHeadComment,
				LineComment: mergeComment(dstKey.LineComment, srcKey.LineComment, opts),
				FootComment: mergeComment(dstKey.FootComment, srcKey.FootComment, opts),
				Line:        dstKey.Line,
				Column:      dstKey.Column,
			}

			// Merge values based on strategy
			var mergedValue *yaml.Node
			if opts.Strategy == MergeDeep && dstValue.Kind == yaml.MappingNode && srcValues[key].Kind == yaml.MappingNode {
				mergedValue = mergeNodes(dstValue, srcValues[key], opts)
			} else {
				mergedValue = copyNodeWithComments(srcValues[key], dstValue, opts)
			}

			// Preserve proper style for complex structures
			if mergedValue.Kind == yaml.MappingNode && len(mergedValue.Content) > 0 {
				// Force literal style (block style) for nested mappings
				mergedValue.Style = 0
			}

			result.Content = append(result.Content, mergedKey, mergedValue)
		} else {
			// Key only in destination - keep it, but ensure @schema blocks have blank lines
			keyNode := copyNode(dstKey)
			if strings.Contains(keyNode.HeadComment, "# @schema") && !strings.HasPrefix(keyNode.HeadComment, "\n") {
				keyNode.HeadComment = "\n" + keyNode.HeadComment
			}
			result.Content = append(result.Content, keyNode, copyNode(dstValue))
		}
	}

	// Add keys that only exist in source
	for i := 0; i < len(src.Content); i += 2 {
		if i+1 >= len(src.Content) || src.Content[i].Kind != yaml.ScalarNode {
			continue
		}

		key := src.Content[i].Value
		if !processed[key] {
			// Copy nodes preserving any HeadComments (which may include spacing)
			keyNode := copyNode(src.Content[i])
			valueNode := copyNode(src.Content[i+1])

			// Ensure @schema blocks have a blank line before them
			if strings.Contains(keyNode.HeadComment, "# @schema") && !strings.HasPrefix(keyNode.HeadComment, "\n") {
				keyNode.HeadComment = "\n" + keyNode.HeadComment
			} else if len(result.Content) > 0 && keyNode.HeadComment == "" {
				// If this is a new top-level key and the previous node exists,
				// we might want to add spacing
				if valueNode.Kind == yaml.MappingNode {
					// Add a newline before major sections for readability
					keyNode.HeadComment = "\n" + keyNode.HeadComment
				}
			}

			result.Content = append(result.Content, keyNode, valueNode)
		}
	}

	return result
}

// appendSequenceNodes appends source sequence to destination
func appendSequenceNodes(dst, src *yaml.Node, opts *MergeOptions) *yaml.Node {
	result := &yaml.Node{
		Kind:        yaml.SequenceNode,
		Style:       dst.Style,
		HeadComment: mergeComment(dst.HeadComment, src.HeadComment, opts),
		LineComment: mergeComment(dst.LineComment, src.LineComment, opts),
		FootComment: mergeComment(dst.FootComment, src.FootComment, opts),
		Content:     make([]*yaml.Node, 0, len(dst.Content)+len(src.Content)),
	}

	// Copy destination elements
	for _, node := range dst.Content {
		result.Content = append(result.Content, copyNode(node))
	}

	// Append source elements
	for _, node := range src.Content {
		result.Content = append(result.Content, copyNode(node))
	}

	return result
}

// copyNodeWithComments copies a node, potentially merging comments from another node
func copyNodeWithComments(src, dst *yaml.Node, opts *MergeOptions) *yaml.Node {
	result := copyNode(src)

	// Force block style for complex structures to avoid inline formatting
	if result.Kind == yaml.MappingNode && len(result.Content) > 2 {
		result.Style = 0 // Block style
		// Also ensure all nested maps use block style
		for i := 1; i < len(result.Content); i += 2 {
			if result.Content[i].Kind == yaml.MappingNode {
				result.Content[i].Style = 0
			}
		}
	}

	if opts.KeepDestinationComments && dst != nil {
		if result.HeadComment == "" && dst.HeadComment != "" {
			result.HeadComment = dst.HeadComment
		}
		if result.LineComment == "" && dst.LineComment != "" {
			result.LineComment = dst.LineComment
		}
		if result.FootComment == "" && dst.FootComment != "" {
			result.FootComment = dst.FootComment
		}
	}

	return result
}

// copyNode creates a deep copy of a yaml.Node
func copyNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	result := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       copyNode(node.Alias),
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	// Ensure proper style for nested mappings
	if result.Kind == yaml.MappingNode && len(node.Content) > 2 {
		// Force block style for maps with multiple entries
		result.Style = 0
	}

	if len(node.Content) > 0 {
		result.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			result.Content[i] = copyNode(child)
		}
	}

	return result
}

// mergeComment merges two comments based on options
func mergeComment(dst, src string, opts *MergeOptions) string {
	// Always preserve destination comments when KeepDestinationComments is true
	// This includes empty lines that are encoded as newlines in comments
	if opts.KeepDestinationComments && dst != "" {
		return dst
	}

	// If both have comments, we may need to combine them intelligently
	// to preserve spacing
	if dst != "" && src != "" {
		// If destination has spacing (starts with newlines), preserve it
		if len(dst) > 0 && dst[0] == '\n' {
			return dst
		}
		if opts.PreferSourceComments {
			return src
		}
		return dst
	}

	if opts.PreferSourceComments && src != "" {
		return src
	}
	if src != "" {
		return src
	}
	return dst
}

// ensureBlockStyle recursively ensures all mapping nodes use block style
func ensureBlockStyle(node *yaml.Node) {
	if node == nil {
		return
	}

	if node.Kind == yaml.MappingNode {
		// Use block style for any mapping with content
		if len(node.Content) > 0 {
			node.Style = 0 // yaml.LiteralStyle = 0 for block style
		}

		// Recursively process all child nodes
		for _, child := range node.Content {
			ensureBlockStyle(child)
		}
	} else if node.Kind == yaml.SequenceNode {
		// Also ensure block style for sequences
		if len(node.Content) > 0 {
			node.Style = 0
		}

		// Process child nodes in sequences
		for _, child := range node.Content {
			ensureBlockStyle(child)
		}
	} else if node.Kind == yaml.DocumentNode {
		// Process document content
		for _, child := range node.Content {
			ensureBlockStyle(child)
		}
	}
}
