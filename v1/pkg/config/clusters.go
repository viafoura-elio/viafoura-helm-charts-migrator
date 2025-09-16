package config

// Cluster represents a Kubernetes cluster configuration
type Cluster struct {
	Default          bool                   `yaml:"default"`
	DefaultNamespace string                 `yaml:"default_namespace"`
	Enabled          bool                   `yaml:"enabled"`
	Target           string                 `yaml:"target"`
	Source           string                 `yaml:"source"`
	AWSProfile       string                 `yaml:"aws_profile"`
	AWSRegion        string                 `yaml:"aws_region"`
	Environments     map[string]Environment `yaml:"environments"`
}

// Environment represents an environment within a cluster
type Environment struct {
	Enabled    bool                 `yaml:"enabled"`
	Namespaces map[string]Namespace `yaml:"namespaces"`
}

// Namespace represents a Kubernetes namespace configuration
type Namespace struct {
	Enabled bool   `yaml:"enabled"`
	Name    string `yaml:"name"`
}
