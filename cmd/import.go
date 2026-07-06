package cmd

import (
	"fmt"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import backup.json",
	Short: "Import server configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		passphrase, err := cmd.Flags().GetString("passphrase")
		if err != nil {
			return err
		}
		dir, err := config.DefaultDir()
		if err != nil {
			return err
		}
		store := config.NewStore(dir)
		secrets := config.NewSecretStore(store.SecretsPath())
		if err := config.ImportBackupWithPassphrase(store, secrets, args[0], passphrase); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Imported VeloSSH backup from %s\n", config.BackupPath(args[0]))
		return nil
	},
}

func init() {
	importCmd.Flags().String("passphrase", "", "backup decryption passphrase")
	rootCmd.AddCommand(importCmd)
}
