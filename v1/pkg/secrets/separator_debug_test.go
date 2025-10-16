package secrets

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"helm-charts-migrator/v1/pkg/config"
)

func TestDebugSeparator(t *testing.T) {
	// Create simple test config
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{},
		},
	}
	cfg.Globals.Secrets.Patterns = []string{
		".*secret.*",
		".*3f4beddd.*",
	}

	// Create extractor
	extractor, err := New(cfg)
	require.NoError(t, err)

	// Test YAML with secrets mixed in configMap
	testYAML := `
configMap:
  root.properties:
    3f4beddd-2061-49b0-ae80-6f1f2ed65b37: "secret-value-1"
    com.viafoura.heimdall.jwt.secret_1: "this is my secret"
    com.viafoura.heimdall.cookie.access.domain: "vf-dev2.org"
`

	// First, let's see what the extractor finds
	extraction, err := extractor.ExtractSecrets([]byte(testYAML), "test-service")
	require.NoError(t, err)

	fmt.Printf("Found %d secrets:\n", len(extraction.Secrets))
	for _, secret := range extraction.Secrets {
		fmt.Printf("  Path: %s, Key: %s\n", secret.Path, secret.Key)
	}

	// Now test the separator
	separator := NewSeparator(extractor)
	result, separationResult, err := separator.SeparateSecrets([]byte(testYAML), "test-service")
	require.NoError(t, err)

	fmt.Printf("\nSeparation result: %d secrets moved\n", separationResult.MovedCount)
	for _, extracted := range separationResult.ExtractedSecrets {
		fmt.Printf("  From: %s -> To: %s\n", extracted.OriginalPath, extracted.NewPath)
	}

	// Print the result
	fmt.Printf("\nResult YAML:\n%s\n", string(result))
}
