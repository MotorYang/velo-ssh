package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vssh",
	Short: "VeloSSH is a TUI SSH manager and SFTP file transfer tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(app.StateServerList)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTUI(start app.AppState) error {
	dir, err := config.DefaultDir()
	if err != nil {
		return err
	}
	store := config.NewStore(dir)
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	m := tui.NewModel(start, store, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
