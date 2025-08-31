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
	"os"
	"path/filepath"

	"github.com/Adembc/lazyssh/internal/adapters/data/file"
	"github.com/Adembc/lazyssh/internal/logger"

	"github.com/Adembc/lazyssh/internal/adapters/ui"
	"github.com/Adembc/lazyssh/internal/core/services"
	"github.com/spf13/cobra"
)

var (
	version   = "develop"
	gitCommit = "unknown"
)

func main() {
	log, err := logger.New("LAZYSSH")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//nolint:errcheck // log.Sync may return an error which is safe to ignore here
	defer log.Sync()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorw("failed to get user home directory", "error", err)
		//nolint:gocritic // exitAfterDefer: ensure immediate exit on unrecoverable error
		os.Exit(1)
	}
	sshConfigFile := filepath.Join(home, ".ssh", "config")
	metaDataFile := filepath.Join(home, ".lazyssh", "metadata.json")

	serverRepo := file.NewServerRepo(log, sshConfigFile, metaDataFile)
	serverService := services.NewServerService(log, serverRepo)
	tui := ui.NewTUI(log, serverService, version, gitCommit)

	rootCmd := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
