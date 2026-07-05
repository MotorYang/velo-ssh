package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export --output backup.json",
	Short: "Export server configuration (v1.1)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("vssh export is not implemented in MVP v1.0")
	},
}

func init() {
	exportCmd.Flags().String("output", "", "backup output path")
	exportCmd.Flags().Bool("include-secrets", false, "include encrypted secrets")
	rootCmd.AddCommand(exportCmd)
}
