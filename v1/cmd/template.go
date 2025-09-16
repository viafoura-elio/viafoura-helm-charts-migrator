package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elioetibr/yaml"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"

	"helm-charts-migrator/v1/pkg/logger"
)

var (
	templateChartPath   string
	templateReleaseName string
	templateNamespace   string
	templateValuesFiles []string
	templateSetValues   []string
	templateOutputDir   string
	templateShowOnly    []string
	templateKubeVersion string
	templateAPIVersions []string
	templateIncludeCRDs bool
	templateSkipTests   bool
	templateValidate    bool
	templateIsUpgrade   bool
	templateDryRun      bool
	templateSplitFiles  bool
)

var templateCmd = &cobra.Command{
	Use:   "template [CHART_PATH]",
	Short: "Render chart templates locally",
	Long: `Render chart templates locally and display the output.

This command renders the Helm chart templates without installing the chart.
It's useful for debugging templates and validating the generated manifests.

Examples:
  # Render the base-chart with default values
  helm-charts-migrator template ./migration/base-chart

  # Render with a specific release name and namespace
  helm-charts-migrator template ./migration/base-chart --release myapp --namespace production

  # Render with custom values file
  helm-charts-migrator template ./migration/base-chart -f custom-values.yaml

  # Render and save to files
  helm-charts-migrator template ./migration/base-chart --output-dir ./manifests

  # Render only specific templates
  helm-charts-migrator template ./migration/base-chart --show-only templates/deployment.yaml

  # Render with specific values
  helm-charts-migrator template ./migration/base-chart --set image.tag=v2.0.0`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemplate,
}

func init() {
	rootCmd.AddCommand(templateCmd)

	templateCmd.Flags().StringVarP(&templateChartPath, "chart", "c", "./migration/base-chart", "Helm Chart path")
	templateCmd.Flags().StringVarP(&templateReleaseName, "release", "r", "release-name", "Release name")
	templateCmd.Flags().StringVarP(&templateNamespace, "namespace", "n", "default", "Namespace to render the manifest")
	templateCmd.Flags().StringArrayVarP(&templateValuesFiles, "values", "f", []string{}, "Specify values in a YAML file (can be specified multiple times)")
	templateCmd.Flags().StringArrayVar(&templateSetValues, "set", []string{}, "Set values on the command line (can be specified multiple times)")
	templateCmd.Flags().StringVar(&templateOutputDir, "output-dir", "", "Write the executed templates to files in output-dir instead of stdout")
	templateCmd.Flags().StringArrayVar(&templateShowOnly, "show-only", []string{}, "Only show manifests rendered from the given templates")
	templateCmd.Flags().StringVar(&templateKubeVersion, "kube-version", "", "Kubernetes version used for Capabilities.KubeVersion")
	templateCmd.Flags().StringArrayVar(&templateAPIVersions, "api-versions", []string{}, "Kubernetes api versions used for Capabilities.APIVersions")
	templateCmd.Flags().BoolVar(&templateIncludeCRDs, "include-crds", false, "Include CRDs in the templated output")
	templateCmd.Flags().BoolVar(&templateSkipTests, "skip-tests", false, "Skip tests from templated output")
	templateCmd.Flags().BoolVar(&templateValidate, "validate", false, "Validate your manifests against the Kubernetes cluster")
	templateCmd.Flags().BoolVar(&templateIsUpgrade, "is-upgrade", false, "Set .Release.IsUpgrade instead of .Release.IsInstall")
	templateCmd.Flags().BoolVar(&templateDryRun, "dry-run", false, "Simulate a template command")
	templateCmd.Flags().BoolVar(&templateSplitFiles, "split", false, "Split output into separate files per resource (requires --output-dir)")
}

func runTemplate(_ *cobra.Command, args []string) error {
	// Use chartPath from flags, override if provided as argument
	chartPath := templateChartPath
	if len(args) > 0 {
		chartPath = args[0]
	}

	// Split flag sets output-dir to default if not specified
	if templateSplitFiles && templateOutputDir == "" {
		templateOutputDir = ".cache/_helm-charts-template"
	}

	// Check if chart path exists
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		return fmt.Errorf("chart path does not exist: %s", chartPath)
	}

	// Check if Chart.yaml exists
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	if _, err := os.Stat(chartFile); os.IsNotExist(err) {
		return fmt.Errorf("chart.yaml not found in %s", chartPath)
	}

	logger.InfoS("Rendering Helm chart", "path", chartPath, "release", templateReleaseName, "namespace", templateNamespace)

	if templateDryRun {
		logger.InfoS("Dry run mode", "chart", chartPath, "release", templateReleaseName)
		return nil
	}

	// Initialize Helm configuration
	settings := cli.New()
	settings.SetNamespace(templateNamespace)

	// Initialize action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), templateNamespace, os.Getenv("HELM_DRIVER"), logger.Info); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	// Load the chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Create install action for template rendering
	client := action.NewInstall(actionConfig)
	client.DryRun = true // This makes it render templates without installing
	client.ReleaseName = templateReleaseName
	client.Namespace = templateNamespace
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = templateIncludeCRDs
	client.DisableHooks = templateSkipTests

	if templateIsUpgrade {
		client.IsUpgrade = true
	}

	// Set Kubernetes version if specified
	if templateKubeVersion != "" {
		client.KubeVersion = &chartutil.KubeVersion{
			Version: templateKubeVersion,
		}
	}

	// Set API versions if specified
	if len(templateAPIVersions) > 0 {
		client.APIVersions = chartutil.VersionSet(templateAPIVersions)
	}

	// Merge values
	valueOpts := &values.Options{
		ValueFiles: templateValuesFiles,
		Values:     templateSetValues,
	}

	vals, err := valueOpts.MergeValues(getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	// Set validation
	if templateValidate {
		client.DryRunOption = "server"
	} else {
		client.DryRunOption = "client"
	}

	// Run the template rendering
	release, err := client.Run(chart, vals)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Handle output
	if templateOutputDir != "" {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(templateOutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if templateSplitFiles {
			// Split manifests into separate files
			resources := parseManifestIntoResources(release.Manifest)
			for _, resource := range resources {
				// Apply show-only filter if specified
				if len(templateShowOnly) > 0 && !shouldShowManifest(resource.source, templateShowOnly) {
					continue
				}

				filename := generateResourceFilename(resource)
				outputPath := filepath.Join(templateOutputDir, filename)
				if err := os.WriteFile(outputPath, []byte(resource.content), 0644); err != nil {
					return fmt.Errorf("failed to write manifest %s: %w", outputPath, err)
				}
				logger.V(2).InfoS("Wrote resource", "file", filename, "kind", resource.kind, "name", resource.name)
			}
		} else if len(templateShowOnly) > 0 {
			// Filter and write specific manifests
			manifests := splitManifests(release.Manifest)
			for _, manifest := range manifests {
				if shouldShowManifest(manifest.source, templateShowOnly) {
					filename := filepath.Base(manifest.source)
					if filename == "" {
						filename = "manifest.yaml"
					}
					outputPath := filepath.Join(templateOutputDir, filename)
					if err := os.WriteFile(outputPath, []byte(manifest.content), 0644); err != nil {
						return fmt.Errorf("failed to write manifest %s: %w", outputPath, err)
					}
				}
			}
		} else {
			// Write all manifests to a single file
			mainPath := filepath.Join(templateOutputDir, "manifest.yaml")
			if err := os.WriteFile(mainPath, []byte(release.Manifest), 0644); err != nil {
				return fmt.Errorf("failed to write main manifest: %w", err)
			}
		}

		logger.InfoS("Templates rendered successfully", "output", templateOutputDir)
	} else {
		// Output to stdout
		if len(templateShowOnly) > 0 {
			// Filter manifests
			manifests := splitManifests(release.Manifest)
			for _, manifest := range manifests {
				if shouldShowManifest(manifest.source, templateShowOnly) {
					fmt.Print(manifest.content)
				}
			}
		} else {
			// Show all manifests
			fmt.Print(release.Manifest)
		}
	}

	return nil
}

func shouldShowManifest(path string, showOnly []string) bool {
	for _, pattern := range showOnly {
		if filepath.Base(path) == filepath.Base(pattern) {
			return true
		}
	}
	return false
}

type manifestInfo struct {
	source  string
	content string
}

type resourceInfo struct {
	source     string
	content    string
	kind       string
	name       string
	namespace  string
	apiVersion string
}

func splitManifests(fullManifest string) []manifestInfo {
	var manifests []manifestInfo
	var currentContent string
	var currentSource string

	lines := strings.Split(fullManifest, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# Source: ") {
			// Save previous manifest if exists
			if currentContent != "" {
				manifests = append(manifests, manifestInfo{
					source:  currentSource,
					content: currentContent,
				})
			}
			// Start new manifest
			currentSource = strings.TrimPrefix(line, "# Source: ")
			currentContent = line + "\n"
		} else {
			currentContent += line + "\n"
		}
	}

	// Add last manifest
	if currentContent != "" {
		manifests = append(manifests, manifestInfo{
			source:  currentSource,
			content: currentContent,
		})
	}

	return manifests
}

func parseManifestIntoResources(fullManifest string) []resourceInfo {
	var resources []resourceInfo

	// Split by YAML document separator
	docs := strings.Split(fullManifest, "\n---\n")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" || doc == "---" {
			continue
		}

		resource := resourceInfo{
			content: doc,
		}

		// Extract source from comment
		lines := strings.Split(doc, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# Source: ") {
				resource.source = strings.TrimPrefix(line, "# Source: ")
				break
			}
		}

		// Parse YAML to extract metadata
		var obj map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &obj); err == nil {
			// Extract apiVersion
			if apiVersion, ok := obj["apiVersion"].(string); ok {
				resource.apiVersion = apiVersion
			}

			// Extract kind
			if kind, ok := obj["kind"].(string); ok {
				resource.kind = kind
			}

			// Extract metadata
			if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
				if name, ok := metadata["name"].(string); ok {
					resource.name = name
				}
				if namespace, ok := metadata["namespace"].(string); ok {
					resource.namespace = namespace
				}
			}
		}

		resources = append(resources, resource)
	}

	return resources
}

func generateResourceFilename(resource resourceInfo) string {
	// Start with a base filename
	var parts []string

	// Add index for ordering (based on resource type priority)
	priority := getResourcePriority(resource.kind)
	parts = append(parts, fmt.Sprintf("%02d", priority))

	// Add resource kind (lowercase)
	if resource.kind != "" {
		parts = append(parts, strings.ToLower(resource.kind))
	}

	// Add resource name
	if resource.name != "" {
		parts = append(parts, resource.name)
	}

	// Join with hyphen and add .yaml extension
	filename := strings.Join(parts, "-") + ".yaml"

	// Sanitize filename (remove any invalid characters)
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, ":", "-")

	return filename
}

func getResourcePriority(kind string) int {
	// Order resources by dependency hierarchy
	priorities := map[string]int{
		"Namespace":                1,
		"ResourceQuota":            2,
		"LimitRange":               3,
		"PodSecurityPolicy":        4,
		"NetworkPolicy":            5,
		"ServiceAccount":           10,
		"Secret":                   11,
		"ConfigMap":                12,
		"StorageClass":             15,
		"PersistentVolume":         16,
		"PersistentVolumeClaim":    17,
		"CustomResourceDefinition": 20,
		"ClusterRole":              21,
		"ClusterRoleBinding":       22,
		"Role":                     23,
		"RoleBinding":              24,
		"Service":                  30,
		"DaemonSet":                40,
		"Deployment":               41,
		"ReplicaSet":               42,
		"StatefulSet":              43,
		"Rollout":                  44,
		"Job":                      50,
		"CronJob":                  51,
		"Ingress":                  60,
		"APIService":               70,
	}

	if priority, ok := priorities[kind]; ok {
		return priority
	}

	// Default priority for unknown resources
	return 99
}
