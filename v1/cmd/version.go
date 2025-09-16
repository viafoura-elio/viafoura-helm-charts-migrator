package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of helm-charts-migrator",
	Long:  `All software has versions. This is helm-charts-migrator's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("helm-charts-migrator v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
