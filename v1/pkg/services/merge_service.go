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

// MergeWithComments merges YAML while preserving comments and structure
// This uses the yaml package's advanced merger that preserves comments
func (m *mergeService) MergeWithComments(baseData, overrideData []byte) ([]byte, *MergeReport, error) {
	// First, track changes using the map-based approach
	var baseMap map[string]interface{}
	if err := yaml.Unmarshal(baseData, &baseMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse base YAML: %w", err)
	}

	var overrideMap map[string]interface{}
	if err := yaml.Unmarshal(overrideData, &overrideMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse override YAML: %w", err)
	}

	// Create report by analyzing the maps
	report := &MergeReport{
		AddedKeys:   []string{},
		UpdatedKeys: []string{},
		DeletedKeys: []string{},
		Conflicts:   []string{},
	}

	// Track changes before merging
	m.analyzeMergeChanges(baseMap, overrideMap, "", report)

	// Now use the yaml package's merger for comment-preserving merge
	baseDoc, err := yaml.Load(baseData, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base YAML document: %w", err)
	}

	overrideDoc, err := yaml.Load(overrideData, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse override YAML document: %w", err)
	}

	// Configure merge to preserve comments
	mergeOpts := &yaml.MergeOptions{
		Strategy:                yaml.MergeDeep,
		PreferSourceComments:    false, // Keep base comments when available
		KeepDestinationComments: true,  // Preserve existing comments
	}

	// Perform the merge with comment preservation
	mergedDoc, err := yaml.Merge(baseDoc, overrideDoc, mergeOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge documents: %w", err)
	}

	// Marshal the merged document preserving comments
	mergedData, err := mergedDoc.Marshal(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode merged YAML: %w", err)
	}

	m.log.InfoS("Merged YAML with comments preserved",
		"added", len(report.AddedKeys),
		"updated", len(report.UpdatedKeys),
		"deleted", len(report.DeletedKeys),
		"conflicts", len(report.Conflicts))

	return mergedData, report, nil
}

// analyzeMergeChanges analyzes what will change in the merge
func (m *mergeService) analyzeMergeChanges(base, override map[string]interface{}, path string, report *MergeReport) {
	// Track keys that exist in override
	overrideKeys := make(map[string]bool)

	// Check for additions and updates
	for key, overrideValue := range override {
		overrideKeys[key] = true
		fullPath := m.buildPath(path, key)

		if baseValue, exists := base[key]; exists {
			// Key exists in both
			baseMap, baseIsMap := baseValue.(map[string]interface{})
			overrideMap, overrideIsMap := overrideValue.(map[string]interface{})

			if baseIsMap && overrideIsMap {
				// Both are maps - recurse
				m.analyzeMergeChanges(baseMap, overrideMap, fullPath, report)
			} else if !m.valuesEqual(baseValue, overrideValue) {
				// Values differ - this is an update
				report.UpdatedKeys = append(report.UpdatedKeys, fullPath)

				// Check for potential conflicts (type changes)
				if baseIsMap != overrideIsMap {
					report.Conflicts = append(report.Conflicts,
						fmt.Sprintf("%s: type change from %T to %T", fullPath, baseValue, overrideValue))
				}
			}
		} else {
			// Key only in override - this is an addition
			report.AddedKeys = append(report.AddedKeys, fullPath)
		}
	}

	// Check for deletions (keys in base but not in override)
	// Note: In a merge operation, we typically don't delete keys
	// This is for tracking purposes only
	for key := range base {
		if !overrideKeys[key] {
			fullPath := m.buildPath(path, key)
			// Mark as potential deletion (though merge won't actually delete)
			// You might want to handle this differently based on merge strategy
			m.log.V(3).InfoS("Key exists only in base (will be preserved)", "path", fullPath)
		}
	}
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

// MergeWithStrategy merges YAML with specific conflict resolution strategy
func (m *mergeService) MergeWithStrategy(baseData, overrideData []byte, strategy MergeStrategy) ([]byte, *MergeReport, error) {
	// Parse documents
	baseDoc, err := yaml.Load(baseData, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base YAML: %w", err)
	}

	overrideDoc, err := yaml.Load(overrideData, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse override YAML: %w", err)
	}

	// Create report
	report := &MergeReport{
		AddedKeys:   []string{},
		UpdatedKeys: []string{},
		DeletedKeys: []string{},
		Conflicts:   []string{},
	}

	// Map strategy to yaml package strategy
	var yamlStrategy yaml.MergeStrategy
	var preferSource bool
	var keepDestComments bool

	switch strategy {
	case MergeStrategyDeep:
		yamlStrategy = yaml.MergeDeep
		preferSource = false
		keepDestComments = true
	case MergeStrategyOverwrite:
		yamlStrategy = yaml.MergeOverwrite
		preferSource = true
		keepDestComments = false
	case MergeStrategyAppend:
		yamlStrategy = yaml.MergeAppend
		preferSource = false
		keepDestComments = true
	case MergeStrategyPreferBase:
		yamlStrategy = yaml.MergeDeep
		preferSource = false
		keepDestComments = true
	case MergeStrategyPreferOverride:
		yamlStrategy = yaml.MergeDeep
		preferSource = true
		keepDestComments = false
	default:
		yamlStrategy = yaml.MergeDeep
		preferSource = false
		keepDestComments = true
	}

	// Configure merge options
	mergeOpts := &yaml.MergeOptions{
		Strategy:                yamlStrategy,
		PreferSourceComments:    preferSource,
		KeepDestinationComments: keepDestComments,
	}

	// Track changes before merge
	var baseMap, overrideMap map[string]interface{}
	yaml.Unmarshal(baseData, &baseMap)
	yaml.Unmarshal(overrideData, &overrideMap)
	m.analyzeMergeChanges(baseMap, overrideMap, "", report)

	// Perform merge
	mergedDoc, err := yaml.Merge(baseDoc, overrideDoc, mergeOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge with strategy %v: %w", strategy, err)
	}

	// Marshal result
	mergedData, err := mergedDoc.Marshal(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal merged data: %w", err)
	}

	m.log.InfoS("Merged with strategy",
		"strategy", strategy,
		"added", len(report.AddedKeys),
		"updated", len(report.UpdatedKeys),
		"conflicts", len(report.Conflicts))

	return mergedData, report, nil
}

// ResolveConflicts applies conflict resolution to a merge report
func (m *mergeService) ResolveConflicts(report *MergeReport, resolution ConflictResolution) *MergeReport {
	if report == nil || len(report.Conflicts) == 0 {
		return report
	}

	resolvedReport := &MergeReport{
		AddedKeys:   report.AddedKeys,
		UpdatedKeys: report.UpdatedKeys,
		DeletedKeys: report.DeletedKeys,
		Conflicts:   []string{},
	}

	for _, conflict := range report.Conflicts {
		switch resolution {
		case ConflictResolutionError:
			// Keep all conflicts - they will cause errors
			resolvedReport.Conflicts = append(resolvedReport.Conflicts, conflict)

		case ConflictResolutionPreferBase:
			// Log that we're using base value
			m.log.V(2).InfoS("Resolved conflict by keeping base value", "conflict", conflict)
			// Move from conflicts to updated (base wins, so no actual update)

		case ConflictResolutionPreferOverride:
			// Log that we're using override value
			m.log.V(2).InfoS("Resolved conflict by using override value", "conflict", conflict)
			// This is already handled in merge, just log it

		case ConflictResolutionLog:
			// Log the conflict but don't fail
			m.log.InfoS("Merge conflict detected", "conflict", conflict)

		case ConflictResolutionInteractive:
			// In a non-interactive context, fall back to logging
			m.log.InfoS("Interactive resolution not available, logging conflict", "conflict", conflict)
			resolvedReport.Conflicts = append(resolvedReport.Conflicts, conflict)

		default:
			// Unknown resolution, keep conflict
			resolvedReport.Conflicts = append(resolvedReport.Conflicts, conflict)
		}
	}

	m.log.InfoS("Resolved conflicts",
		"original", len(report.Conflicts),
		"remaining", len(resolvedReport.Conflicts),
		"resolution", resolution)

	return resolvedReport
}
