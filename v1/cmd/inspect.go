package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
)

var (
	inspectService   string
	inspectCluster   string
	inspectNamespace string
	inspectVerbose   bool
	inspectFormat    string
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect and report on service configuration hierarchy",
	Long: `Inspect loads the hierarchical configuration structure and reports
on service configurations in a human-readable format.

This command helps you understand how configuration values are inherited
and merged across different levels (global, account, cluster, namespace).

Examples:
  # Inspect all services
  helm-charts-migrator inspect

  # Inspect a specific service
  helm-charts-migrator inspect --service heimdall

  # Inspect service configuration for a specific cluster
  helm-charts-migrator inspect --service heimdall --cluster prod01

  # Inspect with verbose output showing all configuration layers
  helm-charts-migrator inspect --service heimdall --verbose

  # Output in different formats
  helm-charts-migrator inspect --service heimdall --format json`,
	RunE: runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)

	inspectCmd.Flags().StringVarP(&inspectService, "service", "s", "", "Service name to inspect")
	inspectCmd.Flags().StringVarP(&inspectCluster, "cluster", "c", "", "Cluster name for context")
	inspectCmd.Flags().StringVarP(&inspectNamespace, "namespace", "n", "", "Namespace for context")
	inspectCmd.Flags().BoolVar(&inspectVerbose, "verbose", false, "Show detailed configuration hierarchy")
	inspectCmd.Flags().StringVarP(&inspectFormat, "format", "f", "text", "Output format (text, json, yaml)")
}

func runInspect(cmd *cobra.Command, args []string) error {
	log := logger.WithName("inspect")

	// Load configuration
	log.InfoS("Loading configuration", "file", cfgFile)
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If specific service requested, inspect that service
	if inspectService != "" {
		return inspectServiceConfig(cfg, inspectService, inspectCluster, inspectNamespace)
	}

	// Otherwise, show overview of all services
	return inspectAllServices(cfg)
}

func inspectAllServices(cfg *config.Config) error {
	fmt.Println("=== Service Configuration Overview ===")
	fmt.Println()

	// Count clusters and namespaces
	totalClusters := 0
	totalNamespaces := 0
	for accountName, account := range cfg.Accounts {
		clusterCount := len(account.Clusters)
		totalClusters += clusterCount

		fmt.Printf("Account: %s (%d cluster%s)\n", accountName, clusterCount, pluralize(clusterCount))

		for clusterName, cluster := range account.Clusters {
			nsCount := 0
			for _, ns := range cluster.Namespaces {
				if ns.Enabled {
					nsCount++
				}
			}
			totalNamespaces += nsCount

			status := "disabled"
			if cluster.Enabled {
				status = "enabled"
			}
			defaultMarker := ""
			if cluster.Default {
				defaultMarker = " (default)"
			}

			fmt.Printf("  ├─ %s: %s%s - %d namespace%s\n",
				clusterName, status, defaultMarker, nsCount, pluralize(nsCount))
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d cluster%s, %d namespace%s\n\n",
		totalClusters, pluralize(totalClusters),
		totalNamespaces, pluralize(totalNamespaces))

	// List services
	fmt.Println("=== Services ===")
	fmt.Println()

	if len(cfg.Services) == 0 {
		fmt.Println("No services configured")
		return nil
	}

	// Sort services by name
	serviceNames := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	enabledCount := 0
	for _, name := range serviceNames {
		service := cfg.Services[name]
		status := "✗ disabled"
		if service.Enabled {
			status = "✓ enabled"
			enabledCount++
		}

		aliasInfo := ""
		if service.Alias != "" {
			aliasInfo = fmt.Sprintf(" (alias: %s)", service.Alias)
		}

		fmt.Printf("  %s %s%s\n", status, name, aliasInfo)

		// Show service-specific overrides if verbose
		if inspectVerbose {
			if service.HasSecrets() {
				fmt.Printf("      • Custom secrets configuration\n")
			}
			if service.HasMappings() {
				fmt.Printf("      • Custom mappings configuration\n")
			}
			if service.HasAutoInject() {
				fmt.Printf("      • Auto-inject rules\n")
			}
		}
	}

	fmt.Printf("\nTotal: %d service%s (%d enabled, %d disabled)\n",
		len(serviceNames), pluralize(len(serviceNames)),
		enabledCount, len(serviceNames)-enabledCount)

	// Show global configuration summary
	fmt.Println("\n=== Global Configuration ===")
	fmt.Println()

	if cfg.Globals.Pipeline.Enabled {
		enabledSteps := 0
		for _, step := range cfg.Globals.Pipeline.Steps {
			if step.Enabled {
				enabledSteps++
			}
		}
		fmt.Printf("Pipeline: enabled (%d/%d steps active)\n", enabledSteps, len(cfg.Globals.Pipeline.Steps))
	} else {
		fmt.Println("Pipeline: disabled")
	}

	fmt.Printf("Converter: skipJavaProperties=%v, skipUppercaseKeys=%v, minUppercaseChars=%d\n",
		cfg.Globals.Converter.SkipJavaProperties,
		cfg.Globals.Converter.SkipUppercaseKeys,
		cfg.Globals.Converter.MinUppercaseChars)

	fmt.Printf("Performance: maxConcurrentServices=%d, showProgress=%v\n",
		cfg.Globals.Performance.MaxConcurrentServices,
		cfg.Globals.Performance.ShowProgress)

	if cfg.Globals.SOPS.Enabled {
		fmt.Printf("SOPS: enabled (profile=%s, workers=%d)\n",
			cfg.Globals.SOPS.AwsProfile,
			cfg.Globals.SOPS.ParallelWorkers)
	} else {
		fmt.Println("SOPS: disabled")
	}

	if cfg.Globals.Secrets != nil {
		status := cfg.Globals.Secrets.GetStatusString()
		if status == "disabled" {
			fmt.Printf("Secrets: %s\n", status)
		} else {
			fmt.Printf("Secrets: %s (%d patterns, %d UUIDs, %d values)\n",
				status,
				len(cfg.Globals.Secrets.Patterns),
				len(cfg.Globals.Secrets.UUIDs),
				len(cfg.Globals.Secrets.Values))
		}
	}

	fmt.Println()
	fmt.Println("💡 Tip: Use --service <name> to inspect specific service configuration")
	fmt.Println("💡 Tip: Use --verbose to see detailed configuration layers")

	return nil
}

func inspectServiceConfig(cfg *config.Config, serviceName, clusterName, namespaceName string) error {
	// Check if service exists
	service, exists := cfg.Services[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found in configuration", serviceName)
	}

	fmt.Printf("=== Service: %s ===\n\n", serviceName)

	// Basic service info
	status := "disabled"
	if service.Enabled {
		status = "enabled"
	}
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Name: %s\n", service.Name)
	if service.Alias != "" {
		fmt.Printf("Alias: %s\n", service.Alias)
	}
	if service.ServiceType != "" {
		fmt.Printf("Type: %s\n", service.ServiceType)
	}
	if service.GitRepo != "" {
		fmt.Printf("Repository: %s\n", service.GitRepo)
	}
	fmt.Println()

	// Get merged configuration
	mergedService, differences := cfg.GetMergedServiceConfig(serviceName)
	if mergedService == nil {
		return fmt.Errorf("failed to get merged configuration for service '%s'", serviceName)
	}

	// Show configuration hierarchy
	fmt.Println("=== Configuration Hierarchy ===")
	fmt.Println()

	// Show what's inherited vs overridden
	fmt.Println("📋 Configuration Sources:")
	fmt.Println("  1. Global defaults (baseline)")
	if len(differences) > 0 {
		fmt.Printf("  2. Service-specific overrides (%d difference%s)\n",
			len(differences), pluralize(len(differences)))
	}
	fmt.Println()

	if len(differences) > 0 && inspectVerbose {
		fmt.Println("🔍 Service Overrides:")
		for _, diff := range differences {
			fmt.Printf("  • %s\n", diff)
		}
		fmt.Println()
	}

	// Show converter configuration
	if mergedService.HasMappings() && mergedService.Mappings != nil {
		fmt.Println("🗺️  Mappings Configuration:")

		if mergedService.Mappings.Normalizer != nil && mergedService.Mappings.Normalizer.Enabled {
			fmt.Printf("  • Normalizer: %d pattern%s\n",
				len(mergedService.Mappings.Normalizer.Patterns),
				pluralize(len(mergedService.Mappings.Normalizer.Patterns)))
		}

		if mergedService.Mappings.Transform != nil && mergedService.Mappings.Transform.Enabled {
			fmt.Printf("  • Transform: %d rule%s\n",
				len(mergedService.Mappings.Transform.Rules),
				pluralize(len(mergedService.Mappings.Transform.Rules)))
		}

		if mergedService.Mappings.Extract != nil && mergedService.Mappings.Extract.Enabled {
			fmt.Printf("  • Extract: %d pattern%s\n",
				len(mergedService.Mappings.Extract.Patterns),
				pluralize(len(mergedService.Mappings.Extract.Patterns)))
		}

		if mergedService.Mappings.Cleaner != nil && mergedService.Mappings.Cleaner.Enabled {
			fmt.Printf("  • Cleaner: %d key pattern%s\n",
				len(mergedService.Mappings.Cleaner.KeyPatterns),
				pluralize(len(mergedService.Mappings.Cleaner.KeyPatterns)))
		}
		fmt.Println()
	}

	// Show secrets configuration
	if mergedService.HasSecrets() && mergedService.Secrets != nil {
		fmt.Println("🔐 Secrets Configuration:")

		status := mergedService.Secrets.GetStatusString()
		fmt.Printf("  Status: %s\n", status)

		if status == "enabled" {

			if len(mergedService.Secrets.Keys) > 0 {
				fmt.Printf("  • Specific keys: %d\n", len(mergedService.Secrets.Keys))
				if inspectVerbose {
					for _, key := range mergedService.Secrets.Keys {
						fmt.Printf("    - %s\n", key)
					}
				}
			}

			if len(mergedService.Secrets.Patterns) > 0 {
				fmt.Printf("  • Patterns: %d\n", len(mergedService.Secrets.Patterns))
				if inspectVerbose {
					for _, pattern := range mergedService.Secrets.Patterns {
						fmt.Printf("    - %s\n", pattern)
					}
				}
			}

			if len(mergedService.Secrets.UUIDs) > 0 {
				fmt.Printf("  • UUID patterns: %d\n", len(mergedService.Secrets.UUIDs))
			}

			if mergedService.Secrets.Locations != nil {
				fmt.Println("  • Locations:")
				if mergedService.Secrets.Locations.BasePath != "" {
					fmt.Printf("    - Base path: %s\n", mergedService.Secrets.Locations.BasePath)
				}
				if mergedService.Secrets.Locations.StorePath != "" {
					fmt.Printf("    - Store path: %s\n", mergedService.Secrets.Locations.StorePath)
				}
				if len(mergedService.Secrets.Locations.AdditionalPaths) > 0 {
					fmt.Printf("    - Additional paths: %d\n", len(mergedService.Secrets.Locations.AdditionalPaths))
				}
			}

			if mergedService.Secrets.Description != "" {
				fmt.Printf("  📝 %s\n", mergedService.Secrets.Description)
			}
		}
		fmt.Println()
	}

	// Show auto-inject configuration
	if mergedService.HasAutoInject() {
		fmt.Println("💉 Auto-Inject Rules:")
		for pattern, injectFile := range mergedService.AutoInject {
			fmt.Printf("  • Pattern: %s\n", pattern)
			fmt.Printf("    Rules: %d\n", len(injectFile.Keys))
			if inspectVerbose {
				for _, rule := range injectFile.Keys {
					fmt.Printf("    - %s: %v (condition: %s)\n",
						rule.Key, rule.Value, rule.Condition)
					if rule.Description != "" {
						fmt.Printf("      %s\n", rule.Description)
					}
				}
			}
		}
		fmt.Println()
	}

	// Show cluster deployment info
	if clusterName != "" {
		cluster := cfg.GetCluster(clusterName)
		if cluster == nil {
			return fmt.Errorf("cluster '%s' not found", clusterName)
		}

		fmt.Printf("=== Deployment Context: %s ===\n\n", clusterName)
		fmt.Printf("Source: %s\n", cluster.Source)
		fmt.Printf("Target: %s\n", cluster.Target)
		fmt.Printf("AWS Profile: %s\n", cluster.AWSProfile)
		fmt.Printf("AWS Region: %s\n", cluster.AWSRegion)
		fmt.Printf("Default Namespace: %s\n", cluster.DefaultNamespace)
		fmt.Println()

		// Show available namespaces
		enabledNS := []string{}
		for nsName, ns := range cluster.Namespaces {
			if ns.Enabled {
				enabledNS = append(enabledNS, nsName)
			}
		}
		if len(enabledNS) > 0 {
			sort.Strings(enabledNS)
			fmt.Printf("Enabled Namespaces (%d):\n", len(enabledNS))
			for _, ns := range enabledNS {
				marker := ""
				if ns == cluster.DefaultNamespace {
					marker = " (default)"
				}
				fmt.Printf("  • %s%s\n", ns, marker)
			}
		}
	}

	if !inspectVerbose {
		fmt.Println()
		fmt.Println("💡 Tip: Use --verbose to see detailed configuration values")
	}

	return nil
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}