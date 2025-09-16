package config

// Service represents a service configuration with all its settings
type Service struct {
	Enabled     bool                      `yaml:"enabled"`
	Name        string                    `yaml:"name"`
	Alias       string                    `yaml:"alias,omitempty"`
	Capitalized string                    `yaml:"capitalized"`
	AutoInject  map[string]AutoInjectFile `yaml:"autoInject,omitempty"`
	Mappings    *Mappings                 `yaml:"mappings,omitempty"`
	Migration   Migration                 `yaml:"migration,omitempty"`
	Secrets     *Secrets                  `yaml:"secrets,omitempty"`
}

// Migration represents migration-specific configuration
type Migration struct {
	BaseValuesPath       string `yaml:"baseValuesPath"`
	EnvValuesPattern     string `yaml:"envValuesPattern"`
	LegacyValuesFilename string `yaml:"legacyValuesFilename"`
	HelmValuesFilename   string `yaml:"helmValuesFilename"`
}

// GetEnabledServices returns only enabled services from a map
func GetEnabledServices(services map[string]Service) []string {
	var enabled []string
	for name, service := range services {
		if service.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// HasSecrets checks if the service has secret configuration
func (s *Service) HasSecrets() bool {
	return s.Secrets != nil && (len(s.Secrets.Keys) > 0 || len(s.Secrets.Patterns) > 0)
}

// HasMappings checks if the service has mapping configuration
func (s *Service) HasMappings() bool {
	return s.Mappings != nil
}

// HasAutoInject checks if the service has auto-injection configuration
func (s *Service) HasAutoInject() bool {
	return len(s.AutoInject) > 0
}
