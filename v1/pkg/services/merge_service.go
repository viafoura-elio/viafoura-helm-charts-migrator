package services

import (
	"fmt"
	"reflect"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/yaml"
)

// mergeService implements MergeService
type mergeService struct {
	config *config.Config
	log    *logger.NamedLogger
}

// NewMergeService creates a new MergeService
func NewMergeService(cfg *config.Config) MergeService {
	return &mergeService{
		config: cfg,
		log:    logger.WithName("merge-service"),
	}
}

// MergeWithComments merges YAML while preserving structure
// Note: Comment preservation requires yaml.v3 Node API which is not available in our yaml package
// This implementation focuses on value merging with tracking
func (m *mergeService) MergeWithComments(baseData, overrideData []byte) ([]byte, *MergeReport, error) {
	// Parse YAML to maps
	var baseMap map[string]interface{}
	if err := yaml.Unmarshal(baseData, &baseMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse base YAML: %w", err)
	}

	var overrideMap map[string]interface{}
	if err := yaml.Unmarshal(overrideData, &overrideMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse override YAML: %w", err)
	}

	// Create report
	report := &MergeReport{
		AddedKeys:   []string{},
		UpdatedKeys: []string{},
		DeletedKeys: []string{},
		Conflicts:   []string{},
	}

	// Merge maps
	mergedMap := m.mergeMaps(baseMap, overrideMap, "", report)

	// Marshal back to YAML
	mergedData, err := yaml.Marshal(mergedMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode merged YAML: %w", err)
	}

	m.log.InfoS("Merged YAML",
		"added", len(report.AddedKeys),
		"updated", len(report.UpdatedKeys),
		"deleted", len(report.DeletedKeys),
		"conflicts", len(report.Conflicts))

	return mergedData, report, nil
}

// mergeMaps recursively merges two maps
func (m *mergeService) mergeMaps(base, override map[string]interface{}, path string, report *MergeReport) map[string]interface{} {
	if base == nil {
		base = make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// Copy all base values
	for key, baseValue := range base {
		result[key] = baseValue
	}

	// Merge override values
	for key, overrideValue := range override {
		fullPath := m.buildPath(path, key)

		if baseValue, exists := base[key]; exists {
			// Key exists in both - need to merge or replace
			baseMap, baseIsMap := baseValue.(map[string]interface{})
			overrideMap, overrideIsMap := overrideValue.(map[string]interface{})

			if baseIsMap && overrideIsMap {
				// Both are maps - recurse
				result[key] = m.mergeMaps(baseMap, overrideMap, fullPath, report)
			} else {
				// Different types or scalar values - replace
				if !m.valuesEqual(baseValue, overrideValue) {
					report.UpdatedKeys = append(report.UpdatedKeys, fullPath)
				}
				result[key] = overrideValue
			}
		} else {
			// Key only in override - add it
			report.AddedKeys = append(report.AddedKeys, fullPath)
			result[key] = overrideValue
		}
	}

	// Track deleted keys (in base but not in override)
	// Note: This is optional based on merge strategy
	// For hierarchical config merging, we typically don't delete keys

	return result
}

// buildPath builds a path string for tracking
func (m *mergeService) buildPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// TrackChanges tracks changes between two value maps
func (m *mergeService) TrackChanges(before, after map[string]interface{}) *ChangeSet {
	changes := &ChangeSet{
		Added:   make(map[string]interface{}),
		Updated: make(map[string]interface{}),
		Deleted: make(map[string]interface{}),
	}

	// Find added and updated keys
	for key, afterValue := range after {
		if beforeValue, exists := before[key]; exists {
			// Check if value changed
			if !m.valuesEqual(beforeValue, afterValue) {
				changes.Updated[key] = map[string]interface{}{
					"before": beforeValue,
					"after":  afterValue,
				}
			}
			// Recurse for nested maps
			if beforeMap, ok1 := beforeValue.(map[string]interface{}); ok1 {
				if afterMap, ok2 := afterValue.(map[string]interface{}); ok2 {
					nestedChanges := m.TrackChanges(beforeMap, afterMap)
					m.mergeNestedChanges(changes, key, nestedChanges)
				}
			}
		} else {
			// Key added
			changes.Added[key] = afterValue
		}
	}

	// Find deleted keys
	for key, beforeValue := range before {
		if _, exists := after[key]; !exists {
			changes.Deleted[key] = beforeValue
		}
	}

	m.log.V(2).InfoS("Tracked changes",
		"added", len(changes.Added),
		"updated", len(changes.Updated),
		"deleted", len(changes.Deleted))

	return changes
}

// valuesEqual checks if two values are equal
func (m *mergeService) valuesEqual(a, b interface{}) bool {
	// Use reflect.DeepEqual for comprehensive comparison
	return reflect.DeepEqual(a, b)
}

// mergeNestedChanges merges nested changes into the parent changeset
func (m *mergeService) mergeNestedChanges(parent *ChangeSet, prefix string, nested *ChangeSet) {
	for key, value := range nested.Added {
		parent.Added[prefix+"."+key] = value
	}
	for key, value := range nested.Updated {
		parent.Updated[prefix+"."+key] = value
	}
	for key, value := range nested.Deleted {
		parent.Deleted[prefix+"."+key] = value
	}
}

// FormatChanges formats changes for logging
func (m *mergeService) FormatChanges(changes *ChangeSet) string {
	var sb strings.Builder

	if len(changes.Added) > 0 {
		sb.WriteString("Added keys:\n")
		for key := range changes.Added {
			sb.WriteString("  + " + key + "\n")
		}
	}

	if len(changes.Updated) > 0 {
		sb.WriteString("Updated keys:\n")
		for key := range changes.Updated {
			sb.WriteString("  ~ " + key + "\n")
		}
	}

	if len(changes.Deleted) > 0 {
		sb.WriteString("Deleted keys:\n")
		for key := range changes.Deleted {
			sb.WriteString("  - " + key + "\n")
		}
	}

	return sb.String()
}
