package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
)

// ChartCopier handles copying base chart with template replacements
type ChartCopier interface {
	CopyBaseChart(src, dst, serviceName, capitalizedName string) error
	SetSourcePath(path string)
}

// chartCopier implements ChartCopier
type chartCopier struct {
	config     *config.Config
	file       services.FileService
	sourcePath string
	log        *logger.NamedLogger
}

// NewChartCopier creates a new ChartCopier
func NewChartCopier(cfg *config.Config, file services.FileService) ChartCopier {
	return &chartCopier{
		config: cfg,
		file:   file,
		log:    logger.WithName("chart-copier"),
	}
}

// SetSourcePath sets the source path for copying additional files
func (c *chartCopier) SetSourcePath(path string) {
	c.sourcePath = path
}

// CopyBaseChart copies the base chart and replaces template placeholders
func (c *chartCopier) CopyBaseChart(src, dst, serviceName, capitalizedName string) error {
	// Copy directory structure
	if err := c.file.CopyDirectory(src, dst); err != nil {
		return fmt.Errorf("failed to copy base chart: %w", err)
	}

	// Replace placeholders in all files
	err := filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Replace placeholders
		newContent := string(content)
		newContent = strings.ReplaceAll(newContent, "base-chart", serviceName)
		newContent = strings.ReplaceAll(newContent, "Base-Chart", capitalizedName)

		// Write back if changed
		if newContent != string(content) {
			if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}
			c.log.V(3).InfoS("Replaced placeholders in file", "path", path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to replace placeholders: %w", err)
	}

	c.log.InfoS("Copied base chart", "service", serviceName, "dst", dst)
	return nil
}
