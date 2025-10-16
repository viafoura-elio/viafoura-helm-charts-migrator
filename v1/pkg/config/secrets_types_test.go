package config

import (
	"testing"
)

func TestSecretsIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		secrets  *Secrets
		expected bool
	}{
		{
			name:     "nil secrets returns true (inherit)",
			secrets:  nil,
			expected: true,
		},
		{
			name:     "empty secrets returns true (inherit)",
			secrets:  &Secrets{},
			expected: true,
		},
		{
			name: "explicitly enabled",
			secrets: &Secrets{
				Enabled: boolPtr(true),
			},
			expected: true,
		},
		{
			name: "explicitly disabled",
			secrets: &Secrets{
				Enabled: boolPtr(false),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.secrets.IsEnabled()
			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSecretsIsExplicitlyDisabled(t *testing.T) {
	tests := []struct {
		name     string
		secrets  *Secrets
		expected bool
	}{
		{
			name:     "nil secrets is not explicitly disabled",
			secrets:  nil,
			expected: false,
		},
		{
			name:     "empty secrets is not explicitly disabled",
			secrets:  &Secrets{},
			expected: false,
		},
		{
			name: "explicitly enabled is not explicitly disabled",
			secrets: &Secrets{
				Enabled: boolPtr(true),
			},
			expected: false,
		},
		{
			name: "explicitly disabled",
			secrets: &Secrets{
				Enabled: boolPtr(false),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.secrets.IsExplicitlyDisabled()
			if result != tt.expected {
				t.Errorf("IsExplicitlyDisabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSecretsGetStatusString(t *testing.T) {
	tests := []struct {
		name     string
		secrets  *Secrets
		expected string
	}{
		{
			name:     "nil secrets shows enabled (inherit)",
			secrets:  nil,
			expected: "enabled",
		},
		{
			name:     "empty secrets shows enabled (inherit)",
			secrets:  &Secrets{},
			expected: "enabled",
		},
		{
			name: "explicitly enabled shows enabled",
			secrets: &Secrets{
				Enabled: boolPtr(true),
			},
			expected: "enabled",
		},
		{
			name: "explicitly disabled shows disabled",
			secrets: &Secrets{
				Enabled: boolPtr(false),
			},
			expected: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.secrets.GetStatusString()
			if result != tt.expected {
				t.Errorf("GetStatusString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}