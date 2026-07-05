package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/sshnet"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect <server-id-or-name>",
	Short: "Connect directly to an SSH shell",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.DefaultDir()
		if err != nil {
			return err
		}
		store := config.NewStore(dir)
		cfg, err := store.Load()
		if err != nil {
			return err
		}
		srv, err := store.FindServer(args[0])
		if err != nil {
			return err
		}
		client := sshnet.NewClient(cfg.Settings, config.OSKeyring{})
		if err := connectWithHostKeyPrompt(client, srv); err != nil {
			return err
		}
		defer client.Close()
		return client.RunInteractiveShell(context.Background(), os.Stdin, os.Stdout, os.Stderr, func(action sshnet.EscapeResult) {
			fmt.Fprintf(os.Stderr, "\nlocal command %q is only available inside the TUI shell flow\n", action.Command)
		})
	},
}

func connectWithHostKeyPrompt(client *sshnet.Client, srv config.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := client.Connect(ctx, srv); err != nil {
		var unknown *sshnet.UnknownHostKeyError
		if !errors.As(err, &unknown) {
			return err
		}
		fmt.Fprintf(os.Stderr, "Unknown SSH host key for %s (%s:%d)\n", srv.Name, srv.Host, srv.Port)
		fmt.Fprintf(os.Stderr, "Fingerprint: %s\n", unknown.Fingerprint)
		fmt.Fprintf(os.Stderr, "Known hosts: %s\n", unknown.KnownHostsPath)
		fmt.Fprint(os.Stderr, "Type yes to trust this host key and continue: ")
		line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("read host key confirmation: %w", readErr)
		}
		if strings.TrimSpace(line) != "yes" {
			return fmt.Errorf("host key was not trusted; connection canceled")
		}
		if err := sshnet.AcceptHostKey(unknown); err != nil {
			return fmt.Errorf("accept host key: %w", err)
		}
		retryCtx, retryCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer retryCancel()
		return client.Connect(retryCtx, srv)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
