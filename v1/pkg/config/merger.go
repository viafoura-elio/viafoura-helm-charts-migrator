package config

import (
	"fmt"
	"reflect"

	"helm-charts-migrator/v1/pkg/logger"
)

// ConfigurationMerger handles merging of global and service-specific configurations
type ConfigurationMerger struct {
	log *logger.NamedLogger
}

// NewConfigMerger creates a new configuration merger
func NewConfigMerger() *ConfigurationMerger {
	return &ConfigurationMerger{
		log: logger.WithName("config-merger"),
	}
}

// MergeServiceConfig merges global configuration with service-specific configuration
func (cm *ConfigurationMerger) MergeServiceConfig(global Globals, service Service) (*Service, []string) {
	var diffs []string
	merged := service // Create a copy to avoid modifying the original

	// Merge AutoInject
	diffs = append(diffs, cm.mergeAutoInject(&merged, global.AutoInject)...)

	// Merge Mappings
	diffs = append(diffs, cm.mergeMappings(&merged, global.Mappings)...)

	// Merge Secrets
	diffs = append(diffs, cm.mergeSecrets(&merged, global.Secrets)...)

	// Merge Migration config
	diffs = append(diffs, cm.mergeMigration(&merged, global.Migration)...)

	if len(diffs) > 0 {
		cm.log.InfoS("Configuration differences found for service",
			"service", service.Name, "differences", len(diffs))
		for _, diff := range diffs {
			cm.log.V(2).InfoS("Configuration difference", "service", service.Name, "diff", diff)
		}
	} else {
		cm.log.V(2).InfoS("No configuration differences", "service", service.Name)
	}

	return &merged, diffs
}

// mergeAutoInject merges global auto-inject configuration with service-specific
func (cm *ConfigurationMerger) mergeAutoInject(service *Service, globalAutoInject map[string]AutoInjectFile) []string {
	var diffs []string

	if len(globalAutoInject) == 0 {
		return diffs
	}

	if service.AutoInject == nil {
		service.AutoInject = make(map[string]AutoInjectFile)
	}

	for path, globalInject := range globalAutoInject {
		if _, exists := service.AutoInject[path]; !exists {
			service.AutoInject[path] = globalInject
			diffs = append(diffs, fmt.Sprintf("autoInject.%s: added from global", path))
		} else {
			// Merge keys from global if not present in service
			for _, globalKey := range globalInject.Keys {
				found := false
				for _, serviceKey := range service.AutoInject[path].Keys {
					if serviceKey.Key == globalKey.Key {
						found = true
						break
					}
				}
				if !found {
					file := service.AutoInject[path]
					file.Keys = append(file.Keys, globalKey)
					service.AutoInject[path] = file
					diffs = append(diffs, fmt.Sprintf("autoInject.%s.%s: added from global", path, globalKey.Key))
				}
			}
		}
	}

	return diffs
}

// mergeMappings merges global mappings with service-specific mappings
func (cm *ConfigurationMerger) mergeMappings(service *Service, globalMappings *Mappings) []string {
	var diffs []string

	if globalMappings == nil {
		return diffs
	}

	if service.Mappings == nil {
		service.Mappings = globalMappings
		diffs = append(diffs, "mappings: using global mappings")
	} else {
		// Deep merge mappings - service takes precedence
		mergedMappings := mergeMappingsDeep(globalMappings, service.Mappings)
		if !reflect.DeepEqual(mergedMappings, service.Mappings) {
			diffs = append(diffs, "mappings: merged with global mappings")
		}
		service.Mappings = mergedMappings
	}

	return diffs
}

// mergeSecrets merges global secrets configuration with service-specific
func (cm *ConfigurationMerger) mergeSecrets(service *Service, globalSecrets *Secrets) []string {
	var diffs []string

	if globalSecrets == nil {
		return diffs
	}

	if service.Secrets == nil {
		service.Secrets = &Secrets{}
	}

	// Merge patterns from global secrets
	for _, pattern := range globalSecrets.Patterns {
		if !contains(service.Secrets.Patterns, pattern) {
			service.Secrets.Patterns = append(service.Secrets.Patterns, pattern)
			diffs = append(diffs, fmt.Sprintf("secrets.patterns: added %s from global", pattern))
		}
	}

	// Merge keys from global secrets
	for _, key := range globalSecrets.Keys {
		if !contains(service.Secrets.Keys, key) {
			service.Secrets.Keys = append(service.Secrets.Keys, key)
			diffs = append(diffs, fmt.Sprintf("secrets.keys: added %s from global", key))
		}
	}

	// Merge UUIDs from global secrets
	for _, uuid := range globalSecrets.UUIDs {
		// Check if UUID pattern already exists
		found := false
		for _, existing := range service.Secrets.UUIDs {
			if existing.Pattern == uuid.Pattern {
				found = true
				break
			}
		}
		if !found {
			service.Secrets.UUIDs = append(service.Secrets.UUIDs, uuid)
			diffs = append(diffs, fmt.Sprintf("secrets.uuids: added pattern %s from global", uuid.Pattern))
		}
	}

	// Merge Values from global secrets
	for _, value := range globalSecrets.Values {
		// Check if Value pattern already exists
		found := false
		for _, existing := range service.Secrets.Values {
			if existing.Pattern == value.Pattern {
				found = true
				break
			}
		}
		if !found {
			service.Secrets.Values = append(service.Secrets.Values, value)
			diffs = append(diffs, fmt.Sprintf("secrets.values: added pattern %s from global", value.Pattern))
		}
	}

	return diffs
}

// mergeMigration merges global migration config with service-specific
func (cm *ConfigurationMerger) mergeMigration(service *Service, globalMigration Migration) []string {
	var diffs []string

	if !reflect.DeepEqual(globalMigration, Migration{}) {
		serviceDiff := compareMigrationConfig(globalMigration, service.Migration)
		diffs = append(diffs, serviceDiff...)
		service.Migration = mergeMigrationConfig(globalMigration, service.Migration)
	}

	return diffs
}

// Helper functions

// mergeMappingsDeep performs a deep merge of mappings (service takes precedence)
func mergeMappingsDeep(global, service *Mappings) *Mappings {
	if global == nil {
		return service
	}
	if service == nil {
		return global
	}

	merged := &Mappings{}

	// Merge each field - service takes precedence
	merged.Locations = mergeOrDefault(service.Locations, global.Locations)
	merged.Normalizer = mergeOrDefault(service.Normalizer, global.Normalizer)
	merged.Transform = mergeOrDefault(service.Transform, global.Transform)
	merged.Extract = mergeOrDefault(service.Extract, global.Extract)
	merged.Cleaner = mergeOrDefault(service.Cleaner, global.Cleaner)

	return merged
}

// mergeOrDefault returns service value if not nil, otherwise returns global value
func mergeOrDefault[T any](service, global *T) *T {
	if service != nil {
		return service
	}
	return global
}

// compareMigrationConfig compares global and service migration configs
func compareMigrationConfig(global, service Migration) []string {
	var diffs []string

	if global.BaseValuesPath != "" && global.BaseValuesPath != service.BaseValuesPath {
		diffs = append(diffs, fmt.Sprintf("migration.baseValuesPath: global=%s, service=%s",
			global.BaseValuesPath, service.BaseValuesPath))
	}

	if global.EnvValuesPattern != "" && global.EnvValuesPattern != service.EnvValuesPattern {
		diffs = append(diffs, fmt.Sprintf("migration.envValuesPattern: global=%s, service=%s",
			global.EnvValuesPattern, service.EnvValuesPattern))
	}

	if global.LegacyValuesFilename != "" && global.LegacyValuesFilename != service.LegacyValuesFilename {
		diffs = append(diffs, fmt.Sprintf("migration.legacyValuesFilename: global=%s, service=%s",
			global.LegacyValuesFilename, service.LegacyValuesFilename))
	}

	if global.HelmValuesFilename != "" && global.HelmValuesFilename != service.HelmValuesFilename {
		diffs = append(diffs, fmt.Sprintf("migration.helmValuesFilename: global=%s, service=%s",
			global.HelmValuesFilename, service.HelmValuesFilename))
	}

	return diffs
}

// mergeMigrationConfig merges global and service migration configs (service takes precedence)
func mergeMigrationConfig(global, service Migration) Migration {
	merged := service

	if merged.BaseValuesPath == "" && global.BaseValuesPath != "" {
		merged.BaseValuesPath = global.BaseValuesPath
	}

	if merged.EnvValuesPattern == "" && global.EnvValuesPattern != "" {
		merged.EnvValuesPattern = global.EnvValuesPattern
	}

	if merged.LegacyValuesFilename == "" && global.LegacyValuesFilename != "" {
		merged.LegacyValuesFilename = global.LegacyValuesFilename
	}

	if merged.HelmValuesFilename == "" && global.HelmValuesFilename != "" {
		merged.HelmValuesFilename = global.HelmValuesFilename
	}

	return merged
}

// contains checks if a string exists in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
