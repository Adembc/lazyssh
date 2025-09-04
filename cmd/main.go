// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Adembc/lazyssh/internal/adapters/data/file"
	"github.com/Adembc/lazyssh/internal/adapters/flags"

	"github.com/Adembc/lazyssh/internal/adapters/config"
	"github.com/Adembc/lazyssh/internal/adapters/logger"
	"github.com/Adembc/lazyssh/internal/adapters/ui"
	"github.com/Adembc/lazyssh/internal/core/services"
	"github.com/spf13/cobra"
)

var (
	version   = "develop"
	gitCommit = "unknown"
)

func main() {
	cfg := config.NewOSConfig()
	sshConfigFile := filepath.Join(cfg.HomeDir(), ".ssh", "config")
	metaDataFile := cfg.ConfigPath("metadata.json")

	serverRepo := file.NewServerRepo(sshConfigFile, metaDataFile)
	serverService := services.NewServerService(serverRepo)
	tui := ui.NewTUI(serverService, version, gitCommit)

	rootCmd := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}

	flagsAdapter := flags.NewCobraFlags(rootCmd)
	// configure logger before running any command
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		logger.ConfigureLogger(cfg.LogPath("lazyssh.log"), flagsAdapter.IsDebug())
		slog.Info("Running lazyssh", "version", version, "commit", gitCommit)
	}
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
