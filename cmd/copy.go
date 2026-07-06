package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/sshnet"
	"github.com/motoryang/velo-ssh/internal/transfer"
	"github.com/spf13/cobra"
)

type remoteSpec struct {
	Server string
	Path   string
}

var copyCmd = &cobra.Command{
	Use:   "copy <source-server>:<remote-path> <target-server>:<remote-path>",
	Short: "Copy one remote file between two configured servers",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceSpec, err := parseRemoteSpec(args[0])
		if err != nil {
			return err
		}
		targetSpec, err := parseRemoteSpec(args[1])
		if err != nil {
			return err
		}
		dir, err := config.DefaultDir()
		if err != nil {
			return err
		}
		store := config.NewStore(dir)
		cfg, err := store.Load()
		if err != nil {
			return err
		}
		sourceServer, err := store.FindServer(sourceSpec.Server)
		if err != nil {
			return err
		}
		targetServer, err := store.FindServer(targetSpec.Server)
		if err != nil {
			return err
		}
		secrets := config.NewSecretStore(store.SecretsPath())
		sourceClient := sshnet.NewClient(cfg.Settings, secrets)
		targetClient := sshnet.NewClient(cfg.Settings, secrets)
		if err := connectWithHostKeyPrompt(sourceClient, sourceServer); err != nil {
			return err
		}
		defer sourceClient.Close()
		if err := connectWithHostKeyPrompt(targetClient, targetServer); err != nil {
			return err
		}
		defer targetClient.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sourceSFTP, err := sourceClient.OpenSFTP(ctx)
		if err != nil {
			return err
		}
		targetSFTP, err := targetClient.OpenSFTP(ctx)
		if err != nil {
			return err
		}
		taskID := fmt.Sprintf("copy-%d", time.Now().UnixNano())
		if err := transfer.AtomicRemoteCopy(sourceSFTP, targetSFTP, sourceSpec.Path, targetSpec.Path, taskID, nil); err != nil {
			return fmt.Errorf("cross-server copy %s -> %s: %w", args[0], args[1], err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Copied %s to %s\n", args[0], args[1])
		return nil
	},
}

func parseRemoteSpec(value string) (remoteSpec, error) {
	idx := strings.Index(value, ":")
	if idx <= 0 || idx == len(value)-1 {
		return remoteSpec{}, fmt.Errorf("remote spec %q must be <server>:<remote-path>", value)
	}
	spec := remoteSpec{Server: strings.TrimSpace(value[:idx]), Path: strings.TrimSpace(value[idx+1:])}
	if spec.Server == "" || spec.Path == "" {
		return remoteSpec{}, fmt.Errorf("remote spec %q must include server and path", value)
	}
	return spec, nil
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
