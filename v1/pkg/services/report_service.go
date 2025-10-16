package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// reportService implements ReportService
type reportService struct {
	config          *config.Config
	log             *logger.NamedLogger
	currentService  string
	startTime       time.Time
	transformations []Transformation
	extractions     []Extraction
	mu              sync.Mutex
}

// NewReportService creates a new ReportService
func NewReportService(cfg *config.Config) ReportService {
	return &reportService{
		config:          cfg,
		log:             logger.WithName("report-service"),
		transformations: []Transformation{},
		extractions:     []Extraction{},
	}
}

// StartReport initializes a new report for a service
func (r *reportService) StartReport(serviceName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.currentService = serviceName
	r.startTime = time.Now()
	r.transformations = []Transformation{}
	r.extractions = []Extraction{}

	r.log.InfoS("Started report", "service", serviceName)
}

// RecordTransformation records a transformation operation
func (r *reportService) RecordTransformation(file string, transformation Transformation) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.transformations = append(r.transformations, transformation)

	if transformation.Error != nil {
		r.log.V(2).InfoS("Transformation failed",
			"file", file,
			"type", transformation.Type,
			"error", transformation.Error)
	} else {
		r.log.V(3).InfoS("Transformation recorded",
			"file", file,
			"type", transformation.Type,
			"applied", transformation.Applied)
	}
}

// RecordExtraction records an extraction operation
func (r *reportService) RecordExtraction(file string, extraction Extraction) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.extractions = append(r.extractions, extraction)

	if extraction.Error != nil {
		r.log.V(2).InfoS("Extraction failed",
			"file", file,
			"type", extraction.Type,
			"error", extraction.Error)
	} else {
		r.log.V(3).InfoS("Extraction recorded",
			"file", file,
			"type", extraction.Type,
			"items", extraction.ItemsCount)
	}
}

// GenerateReport generates the final transformation report
func (r *reportService) GenerateReport() (*TransformationReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	endTime := time.Now()
	duration := endTime.Sub(r.startTime)

	// Calculate summary statistics
	summary := r.calculateSummary(duration)

	report := &TransformationReport{
		ServiceName:     r.currentService,
		StartTime:       r.startTime.Format(time.RFC3339),
		EndTime:         endTime.Format(time.RFC3339),
		Transformations: r.transformations,
		Extractions:     r.extractions,
		Summary:         summary,
	}

	r.log.InfoS("Generated report",
		"service", r.currentService,
		"transformations", summary.TotalTransformations,
		"extractions", summary.TotalExtractions,
		"duration", duration)

	return report, nil
}

// SaveReport saves the report to a file
func (r *reportService) SaveReport(path string) error {
	report, err := r.GenerateReport()
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Determine format based on file extension
	ext := filepath.Ext(path)
	var data []byte

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report to JSON: %w", err)
		}

	case ".yaml", ".yml":
		data, err = yaml.Marshal(report)
		if err != nil {
			return fmt.Errorf("failed to marshal report to YAML: %w", err)
		}

	case ".md", ".markdown":
		data = []byte(r.formatMarkdownReport(report))

	default:
		// Default to text format
		data = []byte(r.formatTextReport(report))
	}

	// Write report to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	r.log.InfoS("Saved report", "path", path, "format", ext)
	return nil
}

// calculateSummary calculates summary statistics
func (r *reportService) calculateSummary(duration time.Duration) ReportSummary {
	summary := ReportSummary{
		TotalTransformations: len(r.transformations),
		TotalExtractions:     len(r.extractions),
		Duration:             duration.String(),
	}

	// Count successful and failed transformations
	for _, t := range r.transformations {
		if t.Error == nil && t.Applied {
			summary.SuccessfulTransforms++
		} else if t.Error != nil {
			summary.FailedTransforms++
		}
	}

	// Count successful and failed extractions
	for _, e := range r.extractions {
		if e.Success {
			summary.SuccessfulExtracts++
		} else {
			summary.FailedExtracts++
		}
	}

	return summary
}

// formatTextReport formats the report as plain text
func (r *reportService) formatTextReport(report *TransformationReport) string {
	output := fmt.Sprintf(`
================================================================================
MIGRATION REPORT - %s
================================================================================

Service: %s
Start Time: %s
End Time: %s
Duration: %s

SUMMARY
-------
Total Transformations: %d (Success: %d, Failed: %d)
Total Extractions: %d (Success: %d, Failed: %d)

`,
		report.ServiceName,
		report.ServiceName,
		report.StartTime,
		report.EndTime,
		report.Summary.Duration,
		report.Summary.TotalTransformations,
		report.Summary.SuccessfulTransforms,
		report.Summary.FailedTransforms,
		report.Summary.TotalExtractions,
		report.Summary.SuccessfulExtracts,
		report.Summary.FailedExtracts)

	// Add transformations section
	if len(report.Transformations) > 0 {
		output += "TRANSFORMATIONS\n"
		output += "---------------\n"
		for i, t := range report.Transformations {
			status := "✓"
			if t.Error != nil {
				status = "✗"
			} else if !t.Applied {
				status = "○"
			}
			output += fmt.Sprintf("%d. [%s] %s - %s\n", i+1, status, t.Type, t.Description)
			if t.Error != nil {
				output += fmt.Sprintf("   Error: %v\n", t.Error)
			}
		}
		output += "\n"
	}

	// Add extractions section
	if len(report.Extractions) > 0 {
		output += "EXTRACTIONS\n"
		output += "-----------\n"
		for i, e := range report.Extractions {
			status := "✓"
			if !e.Success {
				status = "✗"
			}
			output += fmt.Sprintf("%d. [%s] %s\n", i+1, status, e.Type)
			output += fmt.Sprintf("   Source: %s\n", e.Source)
			output += fmt.Sprintf("   Destination: %s\n", e.Destination)
			output += fmt.Sprintf("   Items: %d\n", e.ItemsCount)
			if e.Error != nil {
				output += fmt.Sprintf("   Error: %v\n", e.Error)
			}
		}
		output += "\n"
	}

	output += "================================================================================\n"
	return output
}

// formatMarkdownReport formats the report as Markdown
func (r *reportService) formatMarkdownReport(report *TransformationReport) string {
	output := fmt.Sprintf(`# Migration Report - %s

## Summary

- **Service**: %s
- **Start Time**: %s
- **End Time**: %s
- **Duration**: %s

### Statistics

| Metric | Total | Success | Failed |
|--------|-------|---------|--------|
| Transformations | %d | %d | %d |
| Extractions | %d | %d | %d |

`,
		report.ServiceName,
		report.ServiceName,
		report.StartTime,
		report.EndTime,
		report.Summary.Duration,
		report.Summary.TotalTransformations,
		report.Summary.SuccessfulTransforms,
		report.Summary.FailedTransforms,
		report.Summary.TotalExtractions,
		report.Summary.SuccessfulExtracts,
		report.Summary.FailedExtracts)

	// Add transformations section
	if len(report.Transformations) > 0 {
		output += "## Transformations\n\n"
		for i, t := range report.Transformations {
			status := "✅"
			if t.Error != nil {
				status = "❌"
			} else if !t.Applied {
				status = "⚪"
			}
			output += fmt.Sprintf("%d. %s **%s** - %s\n", i+1, status, t.Type, t.Description)
			if t.Error != nil {
				output += fmt.Sprintf("   - Error: `%v`\n", t.Error)
			}
			if t.Before != nil && t.After != nil {
				output += "   - Changes applied\n"
			}
		}
		output += "\n"
	}

	// Add extractions section
	if len(report.Extractions) > 0 {
		output += "## Extractions\n\n"
		for i, e := range report.Extractions {
			status := "✅"
			if !e.Success {
				status = "❌"
			}
			output += fmt.Sprintf("%d. %s **%s**\n", i+1, status, e.Type)
			output += fmt.Sprintf("   - Source: `%s`\n", e.Source)
			output += fmt.Sprintf("   - Destination: `%s`\n", e.Destination)
			output += fmt.Sprintf("   - Items extracted: %d\n", e.ItemsCount)
			if e.Error != nil {
				output += fmt.Sprintf("   - Error: `%v`\n", e.Error)
			}
		}
		output += "\n"
	}

	// Add legend
	output += `## Legend

- ✅ Successful
- ❌ Failed
- ⚪ Skipped/Not Applied
`

	return output
}

// Helper method to create a transformation record
func CreateTransformation(transformType, description string, before, after interface{}, applied bool, err error) Transformation {
	return Transformation{
		Type:        transformType,
		Description: description,
		Before:      before,
		After:       after,
		Applied:     applied,
		Error:       err,
	}
}

// Helper method to create an extraction record
func CreateExtraction(extractType, source, destination string, itemsCount int, success bool, err error) Extraction {
	return Extraction{
		Type:        extractType,
		Source:      source,
		Destination: destination,
		ItemsCount:  itemsCount,
		Success:     success,
		Error:       err,
	}
}
