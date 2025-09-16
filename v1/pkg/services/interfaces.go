package services

import (
	"context"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
)

// KubernetesService handles all Kubernetes cluster operations
type KubernetesService interface {
	// GetClient returns a Kubernetes client for the given context
	GetClient(context string) (*kubernetes.Clientset, error)
	
	// ListReleases lists all Helm releases in a namespace
	ListReleases(ctx context.Context, kubeContext, namespace string) ([]*release.Release, error)
	
	// GetRelease gets a specific Helm release
	GetRelease(ctx context.Context, kubeContext, namespace, releaseName string) (*release.Release, error)
	
	// SwitchContext switches the kubectl context
	SwitchContext(context string) error
	
	// GetCurrentContext returns the current kubectl context
	GetCurrentContext() (string, error)
}

// HelmService handles Helm-specific operations
type HelmService interface {
	// GetReleaseByName finds a release by service name from a list of releases
	GetReleaseByName(serviceName string, releases []*release.Release) *release.Release
	
	// ExtractValues extracts values from a Helm release
	ExtractValues(release *release.Release) (map[string]interface{}, error)
	
	// ExtractManifest extracts and processes the manifest from a release
	ExtractManifest(release *release.Release) (string, error)
	
	// ValidateChart validates a Helm chart
	ValidateChart(chartPath string) error
}

// FileService handles all file I/O operations
type FileService interface {
	// ReadYAML reads a YAML file and returns it as a map
	ReadYAML(path string) (map[string]interface{}, error)
	
	// WriteYAML writes data to a YAML file
	WriteYAML(path string, data interface{}) error
	
	// CopyDirectory copies a directory recursively
	CopyDirectory(src, dst string) error
	
	// CopyFile copies a single file
	CopyFile(src, dst string) error
	
	// Exists checks if a file or directory exists
	Exists(path string) bool
	
	// EnsureDir creates a directory if it doesn't exist
	EnsureDir(path string) error
	
	// ListFiles lists files in a directory matching a pattern
	ListFiles(dir, pattern string) ([]string, error)
}

// TransformationService handles value transformations
type TransformationService interface {
	// Transform applies transformations to values based on config
	Transform(values map[string]interface{}, config TransformConfig) (map[string]interface{}, error)
	
	// NormalizeKeys normalizes keys in a values map
	NormalizeKeys(values map[string]interface{}) map[string]interface{}
	
	// ExtractSecrets extracts secrets from values
	ExtractSecrets(values map[string]interface{}) (secrets, cleaned map[string]interface{})
	
	// ConvertKeys converts keys based on configured rules (e.g., camelCase)
	ConvertKeys(values map[string]interface{}) map[string]interface{}
	
	// MergeValues merges multiple value sources
	MergeValues(base, override map[string]interface{}) map[string]interface{}
}

// CacheService handles caching of releases and values
type CacheService interface {
	// GetReleases returns cached releases for a cluster:namespace
	GetReleases(cluster, namespace string) []*release.Release
	
	// SetReleases caches releases for a cluster:namespace
	SetReleases(cluster, namespace string, releases []*release.Release) error
	
	// GetTempPath returns a temp path for storing resources
	GetTempPath(cluster, namespace, service, resourceType string) string
	
	// Clear clears the cache
	Clear() error
	
	// Cleanup removes temporary files
	Cleanup() error
}

// SOPSService handles SOPS encryption/decryption
type SOPSService interface {
	// Encrypt encrypts a file using SOPS
	Encrypt(filePath string) error
	
	// Decrypt decrypts a file using SOPS
	Decrypt(filePath string) ([]byte, error)
	
	// EncryptBatch encrypts multiple files in parallel
	EncryptBatch(filePaths []string, workers int) error
	
	// IsEncrypted checks if a file is SOPS encrypted
	IsEncrypted(filePath string) bool
}

// TransformConfig holds transformation configuration
type TransformConfig struct {
	ServiceName      string
	ClusterName      string
	Namespace        string
	GlobalConfig     interface{}
	ServiceConfig    interface{}
}

// MergeService handles merging values with comment preservation
type MergeService interface {
	// MergeWithComments merges YAML nodes while preserving comments
	MergeWithComments(base, override []byte) ([]byte, *MergeReport, error)
	
	// TrackChanges tracks changes between before and after states
	TrackChanges(before, after map[string]interface{}) *ChangeSet
}

// MergeReport contains information about the merge operation
type MergeReport struct {
	AddedKeys   []string
	UpdatedKeys []string
	DeletedKeys []string
	Conflicts   []string
}

// ChangeSet contains information about changes
type ChangeSet struct {
	Added   map[string]interface{}
	Updated map[string]interface{}
	Deleted map[string]interface{}
}

// ReportService handles transformation reporting
type ReportService interface {
	// StartReport initializes a new report for a service
	StartReport(serviceName string)
	
	// RecordTransformation records a transformation operation
	RecordTransformation(file string, transformation Transformation)
	
	// RecordExtraction records an extraction operation
	RecordExtraction(file string, extraction Extraction)
	
	// GenerateReport generates the final transformation report
	GenerateReport() (*TransformationReport, error)
	
	// SaveReport saves the report to a file
	SaveReport(path string) error
}

// Transformation represents a transformation operation
type Transformation struct {
	Type        string
	Description string
	Before      interface{}
	After       interface{}
	Applied     bool
	Error       error
}

// Extraction represents an extraction operation
type Extraction struct {
	Type        string
	Source      string
	Destination string
	ItemsCount  int
	Success     bool
	Error       error
}

// TransformationReport contains the complete transformation report
type TransformationReport struct {
	ServiceName      string
	StartTime        string
	EndTime          string
	Transformations  []Transformation
	Extractions      []Extraction
	Summary          ReportSummary
}

// ReportSummary contains summary statistics
type ReportSummary struct {
	TotalTransformations int
	SuccessfulTransforms int
	FailedTransforms     int
	TotalExtractions     int
	SuccessfulExtracts   int
	FailedExtracts       int
	Duration             string
}