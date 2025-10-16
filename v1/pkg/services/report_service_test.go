package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"helm-charts-migrator/v1/pkg/config"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

func TestReportService_BasicFlow(t *testing.T) {
	cfg := &config.Config{}
	svc := NewReportService(cfg)

	// Start report
	svc.StartReport("test-service")

	// Record some transformations
	svc.RecordTransformation("file1.yaml", Transformation{
		Type:        "camelCase",
		Description: "Convert keys to camelCase",
		Before:      map[string]interface{}{"test_key": "value"},
		After:       map[string]interface{}{"testKey": "value"},
		Applied:     true,
		Error:       nil,
	})

	svc.RecordTransformation("file2.yaml", Transformation{
		Type:        "normalize",
		Description: "Normalize structure",
		Applied:     false,
		Error:       errors.New("normalization failed"),
	})

	// Record some extractions
	svc.RecordExtraction("manifest.yaml", Extraction{
		Type:        "deployment",
		Source:      "k8s-cluster",
		Destination: "local/manifest.yaml",
		ItemsCount:  5,
		Success:     true,
		Error:       nil,
	})

	svc.RecordExtraction("values.yaml", Extraction{
		Type:        "values",
		Source:      "helm-release",
		Destination: "local/values.yaml",
		ItemsCount:  0,
		Success:     false,
		Error:       errors.New("no values found"),
	})

	// Generate report
	report, err := svc.GenerateReport()
	require.NoError(t, err)
	require.NotNil(t, report)

	// Validate report
	assert.Equal(t, "test-service", report.ServiceName)
	assert.Len(t, report.Transformations, 2)
	assert.Len(t, report.Extractions, 2)

	// Check summary
	assert.Equal(t, 2, report.Summary.TotalTransformations)
	assert.Equal(t, 1, report.Summary.SuccessfulTransforms)
	assert.Equal(t, 1, report.Summary.FailedTransforms)
	assert.Equal(t, 2, report.Summary.TotalExtractions)
	assert.Equal(t, 1, report.Summary.SuccessfulExtracts)
	assert.Equal(t, 1, report.Summary.FailedExtracts)
}

func TestReportService_SaveReport(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	svc := NewReportService(cfg)

	// Setup test data
	svc.StartReport("test-service")
	svc.RecordTransformation("test.yaml", Transformation{
		Type:        "test",
		Description: "Test transformation",
		Applied:     true,
	})
	svc.RecordExtraction("test.yaml", Extraction{
		Type:        "test",
		Source:      "source",
		Destination: "dest",
		ItemsCount:  1,
		Success:     true,
	})

	tests := []struct {
		name     string
		filename string
		validate func(t *testing.T, content []byte)
	}{
		{
			name:     "save as JSON",
			filename: "report.json",
			validate: func(t *testing.T, content []byte) {
				var report TransformationReport
				err := json.Unmarshal(content, &report)
				require.NoError(t, err)
				assert.Equal(t, "test-service", report.ServiceName)
				assert.Len(t, report.Transformations, 1)
				assert.Len(t, report.Extractions, 1)
			},
		},
		{
			name:     "save as YAML",
			filename: "report.yaml",
			validate: func(t *testing.T, content []byte) {
				var report TransformationReport
				err := yaml.Unmarshal(content, &report)
				require.NoError(t, err)
				assert.Equal(t, "test-service", report.ServiceName)
			},
		},
		{
			name:     "save as Markdown",
			filename: "report.md",
			validate: func(t *testing.T, content []byte) {
				contentStr := string(content)
				assert.Contains(t, contentStr, "# Migration Report - test-service")
				assert.Contains(t, contentStr, "## Summary")
				assert.Contains(t, contentStr, "## Transformations")
				assert.Contains(t, contentStr, "## Extractions")
				assert.Contains(t, contentStr, "✅")
			},
		},
		{
			name:     "save as text",
			filename: "report.txt",
			validate: func(t *testing.T, content []byte) {
				contentStr := string(content)
				assert.Contains(t, contentStr, "MIGRATION REPORT")
				assert.Contains(t, contentStr, "test-service")
				assert.Contains(t, contentStr, "SUMMARY")
				assert.Contains(t, contentStr, "TRANSFORMATIONS")
				assert.Contains(t, contentStr, "EXTRACTIONS")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.filename)
			err := svc.SaveReport(path)
			require.NoError(t, err)

			// Read and validate
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			tt.validate(t, content)
		})
	}
}

func TestReportService_ConcurrentRecording(t *testing.T) {
	cfg := &config.Config{}
	svc := NewReportService(cfg)

	svc.StartReport("concurrent-test")

	// Record transformations and extractions concurrently
	done := make(chan bool, 4)

	go func() {
		for i := 0; i < 10; i++ {
			svc.RecordTransformation("file.yaml", Transformation{
				Type:        "transform",
				Description: "Test",
				Applied:     true,
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			svc.RecordExtraction("file.yaml", Extraction{
				Type:        "extract",
				Source:      "source",
				Destination: "dest",
				Success:     true,
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			svc.RecordTransformation("error.yaml", Transformation{
				Type:    "error",
				Applied: false,
				Error:   errors.New("test error"),
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			svc.RecordExtraction("error.yaml", Extraction{
				Type:    "error",
				Success: false,
				Error:   errors.New("test error"),
			})
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Generate report
	report, err := svc.GenerateReport()
	require.NoError(t, err)

	// Validate counts
	assert.Equal(t, 15, report.Summary.TotalTransformations)
	assert.Equal(t, 10, report.Summary.SuccessfulTransforms)
	assert.Equal(t, 5, report.Summary.FailedTransforms)
	assert.Equal(t, 15, report.Summary.TotalExtractions)
	assert.Equal(t, 10, report.Summary.SuccessfulExtracts)
	assert.Equal(t, 5, report.Summary.FailedExtracts)
}

func TestReportService_FormatMarkdown(t *testing.T) {
	cfg := &config.Config{}
	svc := NewReportService(cfg)

	svc.StartReport("markdown-test")

	// Add various types of records
	svc.RecordTransformation("success.yaml", Transformation{
		Type:        "camelCase",
		Description: "Convert to camelCase",
		Before:      map[string]interface{}{"old_key": "value"},
		After:       map[string]interface{}{"oldKey": "value"},
		Applied:     true,
	})

	svc.RecordTransformation("skip.yaml", Transformation{
		Type:        "skip",
		Description: "Skipped transformation",
		Applied:     false,
	})

	svc.RecordTransformation("error.yaml", Transformation{
		Type:        "error",
		Description: "Failed transformation",
		Applied:     false,
		Error:       errors.New("transformation error"),
	})

	svc.RecordExtraction("success.yaml", Extraction{
		Type:        "values",
		Source:      "helm-release",
		Destination: "values.yaml",
		ItemsCount:  42,
		Success:     true,
	})

	svc.RecordExtraction("error.yaml", Extraction{
		Type:        "manifest",
		Source:      "k8s-cluster",
		Destination: "manifest.yaml",
		Success:     false,
		Error:       errors.New("extraction error"),
	})

	// Save as markdown
	tmpFile := filepath.Join(t.TempDir(), "report.md")
	err := svc.SaveReport(tmpFile)
	require.NoError(t, err)

	// Read and validate markdown content
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	contentStr := string(content)

	// Check markdown structure
	assert.Contains(t, contentStr, "# Migration Report")
	assert.Contains(t, contentStr, "## Summary")
	assert.Contains(t, contentStr, "## Transformations")
	assert.Contains(t, contentStr, "## Extractions")
	assert.Contains(t, contentStr, "## Legend")

	// Check status icons
	assert.Contains(t, contentStr, "✅") // Success
	assert.Contains(t, contentStr, "❌") // Error
	assert.Contains(t, contentStr, "⚪") // Skipped

	// Check table
	assert.Contains(t, contentStr, "| Metric |")
	assert.Contains(t, contentStr, "| Transformations |")
	assert.Contains(t, contentStr, "| Extractions |")

	// Check specific content
	assert.Contains(t, contentStr, "Convert to camelCase")
	assert.Contains(t, contentStr, "transformation error")
	assert.Contains(t, contentStr, "extraction error")
	assert.Contains(t, contentStr, "Items extracted: 42")
}

func TestReportService_Duration(t *testing.T) {
	cfg := &config.Config{}
	svc := NewReportService(cfg)

	svc.StartReport("duration-test")

	// Add a small delay
	time.Sleep(100 * time.Millisecond)

	svc.RecordTransformation("test.yaml", Transformation{
		Type:    "test",
		Applied: true,
	})

	report, err := svc.GenerateReport()
	require.NoError(t, err)

	// Check that duration is recorded
	assert.NotEmpty(t, report.Summary.Duration)
	assert.True(t, strings.Contains(report.Summary.Duration, "ms") ||
		strings.Contains(report.Summary.Duration, "s"))

	// Check timestamps
	assert.NotEmpty(t, report.StartTime)
	assert.NotEmpty(t, report.EndTime)

	// Parse and compare times
	startTime, err := time.Parse(time.RFC3339, report.StartTime)
	require.NoError(t, err)
	endTime, err := time.Parse(time.RFC3339, report.EndTime)
	require.NoError(t, err)

	assert.True(t, endTime.After(startTime))
}
