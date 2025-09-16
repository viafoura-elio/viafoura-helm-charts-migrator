package cmd

import (
	"github.com/spf13/cobra"

	"helm-charts-migrator/v1/pkg/common"
	"helm-charts-migrator/v1/pkg/migration"
)

var (
	baseHelmChart     string
	sourcePath        string
	targetPath        string
	cacheDir          string
	cleanupCache      bool
	noRefreshCache    bool
	dryRun            bool
	cluster           string
	namespaces        []string
	services          []string
	migrateAwsProfile string
	noSOPS            bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate Helm charts",
	Long:  `Migrate Helm charts from a source to a target location with various options.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return migration.RunMigrationWithFactory(common.MigratorOptions{
			ConfigPath:   cfgFile,
			SourcePath:   sourcePath,
			TargetPath:   targetPath,
			BasePath:     baseHelmChart,
			CacheDir:     cacheDir,
			CleanupCache: cleanupCache,
			RefreshCache: !noRefreshCache,
			DryRun:       dryRun,
			Cluster:      cluster,
			Namespaces:   namespaces,
			Services:     services,
			AwsProfile:   migrateAwsProfile,
			NoSOPS:       noSOPS,
		})
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().StringVarP(&baseHelmChart, "base", "b", "migration/base-chart", "Base path for Helm charts")
	migrateCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Perform a dry run without making changes")
	migrateCmd.Flags().StringVar(&sourcePath, "source", "/Volumes/Development/clients/viafoura/repos/_viafoura-elio/kubernetes-ops/viafoura/charts", "Source path for Helm charts")
	migrateCmd.Flags().StringVar(&targetPath, "target", "apps/", "Target path for migrated charts")
	migrateCmd.Flags().StringVar(&cacheDir, "cache-dir", ".cache", "Directory to store cached resources")
	migrateCmd.Flags().BoolVar(&cleanupCache, "cleanup-cache", false, "Clean up cache directory before migration")
	migrateCmd.Flags().BoolVar(&noRefreshCache, "no-refresh-cache", false, "Skip checking if cache is outdated (use existing cache as-is)")
	migrateCmd.Flags().StringVar(&migrateAwsProfile, "aws-profile", "cicd-sre", "AWS profile to use for SOPS encryption during secrets extraction")
	migrateCmd.Flags().BoolVar(&noSOPS, "no-sops", false, "Skip SOPS encryption of secrets files")

	// New flags for selective migration
	migrateCmd.Flags().StringVarP(&cluster, "cluster", "c", "", "Specific cluster to migrate (optional)")
	migrateCmd.Flags().StringSliceVarP(&namespaces, "namespaces", "n", []string{}, "Specific namespaces to migrate (can be specified multiple times)")
	migrateCmd.Flags().StringSliceVarP(&services, "services", "s", []string{}, "Specific services to migrate (can be specified multiple times)")
}
