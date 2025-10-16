package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"helm-charts-migrator/v1/pkg/logger"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

var (
	baseFile     string
	overrideFile string
	outputFile   string
)

func YamlMergeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yaml-merge",
		Short: "Merge two YAML files using the internal yaml package",
		Long: `Merge a base YAML file with an override YAML file.
The override values take precedence over base values.
Arrays are replaced entirely, not merged.`,
		RunE: runYamlMerge,
	}

	cmd.Flags().StringVar(&baseFile, "base-file", "", "Path to the base YAML file (required)")
	cmd.Flags().StringVar(&overrideFile, "override-file", "", "Path to the override YAML file (required)")
	cmd.Flags().StringVar(&outputFile, "output", "", "Path to the output merged YAML file (required)")

	cmd.MarkFlagRequired("base-file")
	cmd.MarkFlagRequired("override-file")
	cmd.MarkFlagRequired("output")

	return cmd
}

func runYamlMerge(cmd *cobra.Command, args []string) error {
	log := logger.WithName("yaml-merge")
	log.InfoS("Starting YAML merge operation",
		"base", baseFile,
		"override", overrideFile,
		"output", outputFile)

	// Read base file
	baseData, err := os.ReadFile(baseFile)
	if err != nil {
		return fmt.Errorf("failed to read base file %s: %w", baseFile, err)
	}

	// Read override file
	overrideData, err := os.ReadFile(overrideFile)
	if err != nil {
		return fmt.Errorf("failed to read override file %s: %w", overrideFile, err)
	}

	// Parse base YAML using the advanced tree structure
	baseTree, err := yaml.UnmarshalYAML(baseData)
	if err != nil {
		return fmt.Errorf("failed to parse base YAML: %w", err)
	}

	// Parse override YAML using the advanced tree structure
	overrideTree, err := yaml.UnmarshalYAML(overrideData)
	if err != nil {
		return fmt.Errorf("failed to parse override YAML: %w", err)
	}

	// Merge the trees (override takes precedence)
	mergedTree := yaml.MergeTrees(baseTree, overrideTree)

	// Convert merged tree to YAML
	mergedYAML, err := mergedTree.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to convert merged tree to YAML: %w", err)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write merged result to output file
	if err := os.WriteFile(outputFile, mergedYAML, 0644); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputFile, err)
	}

	log.InfoS("YAML merge completed successfully",
		"output", outputFile,
		"size", len(mergedYAML))

	// Also show a diff summary if there were changes
	if len(baseTree.Documents) > 0 && len(overrideTree.Documents) > 0 {
		diffs := yaml.DiffTrees(baseTree, mergedTree)
		if len(diffs) > 0 {
			log.InfoS("Changes applied during merge",
				"total_changes", len(diffs))

			// Log first few changes as examples
			maxChangesToShow := 5
			if len(diffs) < maxChangesToShow {
				maxChangesToShow = len(diffs)
			}

			for i := 0; i < maxChangesToShow; i++ {
				diff := diffs[i]
				log.V(2).InfoS("Change detected",
					"type", string(diff.Type),
					"path", diff.Path)
			}

			if len(diffs) > maxChangesToShow {
				log.InfoS("Additional changes omitted",
					"omitted", len(diffs)-maxChangesToShow)
			}
		} else {
			log.InfoS("No differences detected between base and merged result")
		}
	}

	return nil
}
