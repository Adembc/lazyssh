package main

import (
	"fmt"
	"github.com/Adembc/lazyssh/internal/adapters/data/memory"
	"github.com/Adembc/lazyssh/internal/adapters/ui"
	"github.com/Adembc/lazyssh/internal/core/services"
	"github.com/spf13/cobra"
	"os"
)

func main() {

	serverInMemoryRepo := memory.NewServerRepository()
	serverService := services.NewServerService(serverInMemoryRepo)
	tui := ui.NewTUI(serverService)

	rootCmd := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
