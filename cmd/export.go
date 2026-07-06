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
		encrypt, err := cmd.Flags().GetBool("encrypt")
		if err != nil {
			return err
		}
		passphrase, err := cmd.Flags().GetString("passphrase")
		if err != nil {
			return err
		}
		if encrypt && passphrase == "" {
			return fmt.Errorf("export backup: --passphrase is required with --encrypt")
		}
		dir, err := config.DefaultDir()
		if err != nil {
			return err
		}
		store := config.NewStore(dir)
		secrets := config.NewSecretStore(store.SecretsPath())
		if !encrypt {
			passphrase = ""
		}
		if err := config.ExportBackupWithPassphrase(store, secrets, output, includeSecrets, passphrase); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Exported VeloSSH backup to %s\n", config.BackupPath(output))
		if includeSecrets {
			if encrypt {
				fmt.Fprintln(cmd.OutOrStdout(), "Secret values were encrypted with AES-256-GCM.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Warning: exported secret values are stored in plaintext in the backup file.")
			}
		}
		return nil
	},
}

func init() {
	exportCmd.Flags().String("output", "", "backup output path")
	exportCmd.Flags().Bool("include-secrets", false, "include secret values in plaintext backup")
	exportCmd.Flags().Bool("encrypt", false, "encrypt backup payload with AES-256-GCM")
	exportCmd.Flags().String("passphrase", "", "backup encryption/decryption passphrase")
	rootCmd.AddCommand(exportCmd)
}
