package config

// AutoInjectFile represents auto-injection configuration for a specific file pattern
type AutoInjectFile struct {
	Keys []AutoInjectKey `yaml:"keys"`
}

// AutoInjectKey represents a single key-value injection rule
type AutoInjectKey struct {
	Key         string `yaml:"key"`
	Value       string `yaml:"value"`
	Condition   string `yaml:"condition"` // ifExists, ifNotExists, always, disabled
	Description string `yaml:"description"`
}

// InjectionCondition represents when to apply an injection
type InjectionCondition string

const (
	ConditionIfExists    InjectionCondition = "ifExists"
	ConditionIfNotExists InjectionCondition = "ifNotExists"
	ConditionAlways      InjectionCondition = "always"
	ConditionDisabled    InjectionCondition = "disabled"
)

// IsValid checks if the injection condition is valid
func (c InjectionCondition) IsValid() bool {
	switch c {
	case ConditionIfExists, ConditionIfNotExists, ConditionAlways, ConditionDisabled:
		return true
	default:
		return false
	}
}

// ShouldApply determines if injection should be applied based on condition and key existence
func (c InjectionCondition) ShouldApply(keyExists bool) bool {
	switch c {
	case ConditionIfExists:
		return keyExists
	case ConditionIfNotExists:
		return !keyExists
	case ConditionAlways:
		return true
	case ConditionDisabled:
		return false
	default:
		return false
	}
}
