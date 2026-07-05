package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import backup.json",
	Short: "Import server configuration (v1.1)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("vssh import is not implemented in MVP v1.0")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
