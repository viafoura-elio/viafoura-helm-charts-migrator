package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/elioetibr/yaml"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
)

var (
	validateService           string
	validateCluster           string
	validateNamespace         string
	validateStrict            bool
	validateKubeconformConfig string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the migrated Helm chart templates",
	Long: `Validate the migrated Helm chart templates for correctness.

This command validates the Helm chart templates using various methods:
- Helm template with --validate flag
- Kubernetes dry-run (if cluster is accessible)
- Kubeconform for offline validation (if installed)

Examples:
  # Validate templates for auth-service in prod01 cluster
  helm-charts-migrator validate --service auth-service --cluster prod01

  # Validate with strict mode (fail on warnings)
  helm-charts-migrator validate --service auth-service --cluster prod01 --strict

  # Validate with kubeconform config
  helm-charts-migrator validate --service auth-service --cluster prod01 --kubeconform-config ./kubeconform.yaml`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&validateService, "service", "s", "", "Service name to validate (required)")
	validateCmd.Flags().StringVarP(&validateCluster, "cluster", "c", "", "Cluster name (required)")
	validateCmd.Flags().StringVarP(&validateNamespace, "namespace", "n", "", "Namespace (optional)")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "Fail on warnings")
	validateCmd.Flags().StringVar(&validateKubeconformConfig, "kubeconform-config", "", "Path to kubeconform config file")

	validateCmd.MarkFlagRequired("service")
	validateCmd.MarkFlagRequired("cluster")
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate cluster
	clusterConfig, exists := cfg.Clusters[validateCluster]
	if !exists {
		return fmt.Errorf("cluster %s not found in configuration", validateCluster)
	}

	// Validate service
	serviceConfig, exists := cfg.Services[validateService]
	if !exists {
		return fmt.Errorf("service %s not found in configuration", validateService)
	}

	if !serviceConfig.Enabled {
		logger.Warning("Service is not enabled", "service", validateService)
	}

	// Determine namespace
	ns := validateNamespace
	if ns == "" {
		// Look for first enabled namespace across all environments
		for _, env := range clusterConfig.Environments {
			if !env.Enabled {
				continue
			}
			for nsName, nsConfig := range env.Namespaces {
				if nsConfig.Enabled {
					ns = nsName
					break
				}
			}
			if ns != "" {
				break
			}
		}
		if ns == "" {
			return fmt.Errorf("no namespace specified and no enabled namespace found")
		}
	}

	// Build paths
	cacheDir := fmt.Sprintf(".cache/%s/%s/%s", validateCluster, ns, validateService)
	baseChartPath := filepath.Join(cacheDir, "base-chart")
	valuesPath := filepath.Join(baseChartPath, "values.yaml")

	// Check if base-chart exists
	if _, err := os.Stat(baseChartPath); os.IsNotExist(err) {
		return fmt.Errorf("base-chart not found at %s. Please run migrate command first", baseChartPath)
	}

	logger.InfoS("Validating Helm chart",
		"service", validateService,
		"cluster", validateCluster,
		"namespace", ns)

	var validationErrors []string

	// 1. Validate with Helm template --validate
	if err := validateWithHelm(baseChartPath, valuesPath, ns); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("Helm validation: %v", err))
		logger.Error(err, "Helm template validation failed")
	} else {
		logger.InfoS("✓ Helm template validation passed")
	}

	// 2. Validate with kubeconform if available
	if isCommandAvailable("kubeconform") {
		if err := validateWithKubeconform(baseChartPath, valuesPath, ns); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Kubeconform validation: %v", err))
			logger.Error(err, "Kubeconform validation failed")
		} else {
			logger.InfoS("✓ Kubeconform validation passed")
		}
	} else {
		logger.V(2).InfoS("Kubeconform not found, skipping offline validation")
	}

	// 3. Check for deprecated API versions
	if warnings := checkDeprecatedAPIs(baseChartPath, valuesPath, ns); len(warnings) > 0 {
		for _, warning := range warnings {
			logger.Warning(warning)
			if validateStrict {
				validationErrors = append(validationErrors, warning)
			}
		}
	}

	// 4. Validate chart structure
	if err := validateChartStructure(baseChartPath); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("Chart structure: %v", err))
		logger.Error(err, "Chart structure validation failed")
	} else {
		logger.InfoS("✓ Chart structure validation passed")
	}

	// Report results
	if len(validationErrors) > 0 {
		fmt.Println("\n❌ Validation failed with the following errors:")
		for _, err := range validationErrors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("validation failed with %d errors", len(validationErrors))
	}

	fmt.Println("\n✅ All validations passed successfully!")
	return nil
}

func validateWithHelm(chartPath, valuesPath, namespace string) error {
	// Initialize Helm configuration
	settings := cli.New()
	settings.SetNamespace(namespace)

	// Initialize action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logger.Info); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	// Load the chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Load values
	values := make(map[string]interface{})
	if valuesPath != "" {
		valuesBytes, err := os.ReadFile(valuesPath)
		if err != nil {
			return fmt.Errorf("failed to read values file: %w", err)
		}
		if err := yaml.Unmarshal(valuesBytes, &values); err != nil {
			return fmt.Errorf("failed to parse values file: %w", err)
		}
	}

	// Create install action for validation
	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = "validation-test"
	client.Namespace = namespace
	client.ClientOnly = false      // This enables server-side validation
	client.DryRunOption = "server" // Validate against the cluster

	// Run validation
	_, err = client.Run(chart, values)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

func validateWithKubeconform(chartPath, valuesPath, namespace string) error {
	// Render templates using Helm SDK
	manifest, err := renderTemplates(chartPath, valuesPath, namespace)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Then validate with kubeconform
	kubeconformArgs := []string{
		"-summary",
		"-exit-on-error",
	}

	if validateKubeconformConfig != "" {
		kubeconformArgs = append(kubeconformArgs, "-config", validateKubeconformConfig)
	}

	kubeconformCmd := exec.Command("kubeconform", kubeconformArgs...)
	kubeconformCmd.Stdin = bytes.NewReader([]byte(manifest))

	output, err := kubeconformCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}

	return nil
}

func checkDeprecatedAPIs(chartPath, valuesPath, namespace string) []string {
	var warnings []string

	// Render templates using Helm SDK
	content, err := renderTemplates(chartPath, valuesPath, namespace)
	if err != nil {
		logger.V(2).InfoS("Failed to render templates for deprecated API check", "error", err)
		return warnings
	}

	// Check for deprecated API versions
	deprecatedAPIs := map[string]string{
		"extensions/v1beta1":                "apps/v1",
		"apps/v1beta1":                      "apps/v1",
		"apps/v1beta2":                      "apps/v1",
		"networking.k8s.io/v1beta1/Ingress": "networking.k8s.io/v1",
		"rbac.authorization.k8s.io/v1beta1": "rbac.authorization.k8s.io/v1",
		"batch/v1beta1/CronJob":             "batch/v1",
	}

	for deprecated, replacement := range deprecatedAPIs {
		if strings.Contains(content, deprecated) {
			warnings = append(warnings,
				fmt.Sprintf("Deprecated API version '%s' found, should use '%s'", deprecated, replacement))
		}
	}

	return warnings
}

func validateChartStructure(chartPath string) error {
	// Check required files
	requiredFiles := []string{
		"Chart.yaml",
		"values.yaml",
	}

	for _, file := range requiredFiles {
		path := filepath.Join(chartPath, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("required file %s not found", file)
		}
	}

	// Check templates directory
	templatesDir := filepath.Join(chartPath, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return fmt.Errorf("templates directory not found")
	}

	// Check if templates directory has at least one file
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	hasTemplates := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			hasTemplates = true
			break
		}
	}

	if !hasTemplates {
		return fmt.Errorf("no template files found in templates directory")
	}

	return nil
}

func isCommandAvailable(command string) bool {
	cmd := exec.Command("which", command)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// renderTemplates renders Helm chart templates using the Helm SDK
func renderTemplates(chartPath, valuesPath, namespace string) (string, error) {
	// Initialize Helm configuration
	settings := cli.New()
	settings.SetNamespace(namespace)

	// Initialize action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logger.Info); err != nil {
		return "", fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	// Load the chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart: %w", err)
	}

	// Load values
	values := make(map[string]interface{})
	if valuesPath != "" {
		valuesBytes, err := os.ReadFile(valuesPath)
		if err != nil {
			return "", fmt.Errorf("failed to read values file: %w", err)
		}
		if err := yaml.Unmarshal(valuesBytes, &values); err != nil {
			return "", fmt.Errorf("failed to parse values file: %w", err)
		}
	}

	// Create install action for template rendering
	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = "validation-test"
	client.Namespace = namespace
	client.ClientOnly = true // Just render templates, don't validate against cluster

	// Run template rendering
	release, err := client.Run(chart, values)
	if err != nil {
		return "", fmt.Errorf("failed to render templates: %w", err)
	}

	return release.Manifest, nil
}
