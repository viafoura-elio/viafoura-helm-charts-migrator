package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
)

// ChartCopier handles copying base chart with template replacements
type ChartCopier interface {
	CopyBaseChart(src, dst, serviceName, capitalizedName string) error
	CopyBaseChartWithService(src, dst string, service *config.Service) error
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

// CopyBaseChart copies the base chart and replaces template placeholders (legacy method)
func (c *chartCopier) CopyBaseChart(src, dst, serviceName, capitalizedName string) error {
	// Create a minimal service config for backward compatibility
	service := &config.Service{
		Name:        serviceName,
		Capitalized: capitalizedName,
	}
	return c.CopyBaseChartWithService(src, dst, service)
}

// CopyBaseChartWithService copies the base chart and replaces all template placeholders
func (c *chartCopier) CopyBaseChartWithService(src, dst string, service *config.Service) error {
	// Copy directory structure
	if err := c.file.CopyDirectory(src, dst); err != nil {
		return fmt.Errorf("failed to copy base chart: %w", err)
	}

	// Prepare replacement patterns with word boundaries
	replacements := c.prepareReplacements(service)

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

		// Apply replacements
		newContent := string(content)
		changed := false

		for _, r := range replacements {
			if r.regex != nil {
				// Use regex replacement for word boundary matching
				replaced := r.regex.ReplaceAllString(newContent, r.replacement)
				if replaced != newContent {
					newContent = replaced
					changed = true
				}
			} else {
				// Use simple string replacement for URLs
				if strings.Contains(newContent, r.pattern) {
					newContent = strings.ReplaceAll(newContent, r.pattern, r.replacement)
					changed = true
				}
			}
		}

		// Write back if changed
		if changed {
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

	c.log.InfoS("Copied base chart", "service", service.Name, "dst", dst)
	return nil
}

// replacement holds a pattern and its replacement
type replacement struct {
	pattern     string
	replacement string
	regex       *regexp.Regexp
}

// prepareReplacements creates all replacement patterns for a service
func (c *chartCopier) prepareReplacements(service *config.Service) []replacement {
	var replacements []replacement

	// IMPORTANT: Process full URL first before word-based replacements
	// to avoid partial replacements in URLs

	// 1. Git repo URL replacement (exact string match) - MUST BE FIRST
	if service.GitRepo != "" {
		replacements = append(replacements, replacement{
			pattern:     "https://github.com/viafoura/base-chart-github-repo",
			replacement: service.GitRepo,
			regex:       nil, // Use exact string matching for URLs
		})
	}

	// 2. Base-Chart-Type -> serviceTypeCapitalized (with word boundaries)
	if service.ServiceTypeCapitalized != "" {
		replacements = append(replacements, replacement{
			pattern:     "Base-Chart-Type",
			replacement: service.ServiceTypeCapitalized,
			regex:       regexp.MustCompile(`\bBase-Chart-Type\b`),
		})
	}

	// 3. Base-Chart -> capitalized (with word boundaries)
	if service.Capitalized != "" {
		replacements = append(replacements, replacement{
			pattern:     "Base-Chart",
			replacement: service.Capitalized,
			regex:       regexp.MustCompile(`\bBase-Chart\b`),
		})
	}

	// 4. base-chart-type -> serviceType (with word boundaries)
	if service.ServiceType != "" {
		replacements = append(replacements, replacement{
			pattern:     "base-chart-type",
			replacement: service.ServiceType,
			regex:       regexp.MustCompile(`\bbase-chart-type\b`),
		})
	}

	// 5. base-chart -> name (with word boundaries)
	if service.Name != "" {
		replacements = append(replacements, replacement{
			pattern:     "base-chart",
			replacement: service.Name,
			regex:       regexp.MustCompile(`\bbase-chart\b`),
		})
	}

	return replacements
}