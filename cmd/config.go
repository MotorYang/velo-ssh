package cmd

import (
	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open VeloSSH interactive settings center",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(app.StateSettingsCenter)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
