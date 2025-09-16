package common

// MigratorOptions contains configuration options for migration
type MigratorOptions struct {
	// ConfigPath is the config.yaml Path
	ConfigPath string
	// SourcePath is the Legacy Helm Charts Path
	SourcePath string
	// SourcePath is the Legacy Helm Charts Path
	TargetPath   string
	BasePath     string
	CacheDir     string
	CleanupCache bool
	RefreshCache bool
	DryRun       bool

	// Override options from CLI
	Cluster    string
	Namespaces []string
	Services   []string
	AwsProfile string
	NoSOPS     bool // Skip SOPS encryption when true
}
