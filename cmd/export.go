package cmd

import (
	"fmt"

	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export --output backup.json",
	Short: "Export server configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		includeSecrets, err := cmd.Flags().GetBool("include-secrets")
		if err != nil {
			return err
		}
		dir, err := config.DefaultDir()
		if err != nil {
			return err
		}
		store := config.NewStore(dir)
		secrets := config.NewSecretStore(store.SecretsPath())
		if err := config.ExportBackup(store, secrets, output, includeSecrets); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Exported VeloSSH backup to %s\n", config.BackupPath(output))
		if includeSecrets {
			fmt.Fprintln(cmd.OutOrStdout(), "Warning: exported secret values are stored in plaintext in the backup file.")
		}
		return nil
	},
}

func init() {
	exportCmd.Flags().String("output", "", "backup output path")
	exportCmd.Flags().Bool("include-secrets", false, "include secret values in plaintext backup")
	rootCmd.AddCommand(exportCmd)
}
