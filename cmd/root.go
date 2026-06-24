package cmd

import (
	"os"

	"github.com/masudur-rahman/kazi-ancestry/configs"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kazi-ancestry",
	Short: "Kazi Ancestry — family tree server",
}

// Execute runs the root command. Called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(configs.Load)
	rootCmd.PersistentFlags().StringVar(&configs.CfgFile, "config", "", "path to YAML config file (default: $CONFIG_FILE or config.yaml)")
}
