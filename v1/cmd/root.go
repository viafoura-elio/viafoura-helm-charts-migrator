package cmd

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"

	"helm-charts-migrator/v1/pkg/logger"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "helm-charts-migrator",
	Short: "A tool for migrating Helm charts",
	Long: `Helm Charts Migrator is a CLI tool that helps you migrate
and manage Helm charts efficiently.

Get started:
  1. Initialize a configuration file:
     $ helm-charts-migrator init
  
  2. Edit the configuration to match your environment:
     $ vi config.yaml
  
  3. Run the migration:
     $ helm-charts-migrator migrate

For more information, use --help with any command.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	defer logger.Flush()
	err := rootCmd.Execute()
	if err != nil {
		logger.Error(err, "Failed to execute command")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().
		StringVar(&cfgFile,
			"config",
			"./config.yaml",
			"config file (default is ./config.yaml)")

	// Add klog flags to the command
	fs := flag.NewFlagSet("klog", flag.ExitOnError)
	logger.InitFlags(fs)
	rootCmd.PersistentFlags().AddGoFlagSet(fs)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for config in current directory first
		cwd, err := os.Getwd()
		if err == nil {
			viper.AddConfigPath(cwd)
		}
		
		// Also look in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
			viper.AddConfigPath(home + "/.config/helm-charts-migrator")
		}
		
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("MIGRATOR") // Allow MIGRATOR_* environment variables

	if err := viper.ReadInConfig(); err == nil {
		logger.V(1).InfoS("Using config file", "file", viper.ConfigFileUsed())
	} else {
		// Only show this for commands that need config (not init, version, help)
		if rootCmd.Name() != "init" && rootCmd.Name() != "version" && rootCmd.Name() != "help" {
			logger.V(3).InfoS("Config file not found, using defaults", "error", err)
		}
	}

	// Flush logs before the program exits
	klog.Flush()
}
