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
	"strings"

	"github.com/Adembc/lazyssh/internal/adapters/data/file"
	"github.com/Adembc/lazyssh/internal/logger"

	"github.com/Adembc/lazyssh/internal/adapters/ui"
	"github.com/Adembc/lazyssh/internal/core/services"
	"github.com/spf13/cobra"
)

var (
	version   = "develop"
	gitCommit = "unknown"

	// Command-line flags
	enableAWS bool
	serverAlias string
	configDir string
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
	
	// Use provided config directory or default to ~/config/lazyssh
	var configDirPath string
	if configDir != "" {
		// Use absolute path if provided, or relative to home if starts with ~/
		if filepath.IsAbs(configDir) {
			configDirPath = configDir
		} else if strings.HasPrefix(configDir, "~/") {
			configDirPath = filepath.Join(home, configDir[2:])
		} else {
			configDirPath = configDir
		}
	} else {
		configDirPath = filepath.Join(home, ".config", "lazyssh")
	}
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDirPath, 0755); err != nil {
		log.Errorw("failed to create config directory", "path", configDirPath, "error", err)
		os.Exit(1)
	}
	
	sshConfigFile := filepath.Join(home, ".ssh", "config")
	metaDataFile := filepath.Join(configDirPath, "metadata.json")
	configFile := filepath.Join(configDirPath, "lazyssh.yaml")

	serverRepo := file.NewServerRepoWithConfigDir(log, sshConfigFile, metaDataFile, configFile, configDirPath)
	serverService := services.NewServerService(log, serverRepo)
	tui := ui.NewTUI(log, serverService, version, gitCommit)

	rootCmd := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --server flag is provided, connect directly without TUI
			if serverAlias != "" {
				log.Infow("Direct server connection requested", "alias", serverAlias)
				fmt.Printf("Connecting to server: %s\n", serverAlias)
				
				// Connect directly to the specified server
				if err := serverService.SSH(serverAlias); err != nil {
					return fmt.Errorf("failed to connect to server %s: %w", serverAlias, err)
				}
				return nil
			}
			
			// Default behavior: launch TUI
			return tui.Run()
		},
	}
	rootCmd.SilenceUsage = true

	// Add command-line flags
	rootCmd.Flags().StringVar(&serverAlias, "server", "", "Connect directly to specified server alias (bypasses TUI)")
	rootCmd.Flags().StringVar(&configDir, "config-dir", "", "Config directory path (default: ~/.config/lazyssh)")
	rootCmd.Flags().BoolVar(&enableAWS, "enable-aws", true,
		"Enable loading servers from AWS function definitions (default: true)")
	rootCmd.Flags().Bool("disable-aws", false, "Disable loading servers from AWS function definitions")

	// Handle the disable-aws flag and apply configuration overrides
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Check if disable-aws flag was provided
		if disableAWS, _ := cmd.Flags().GetBool("disable-aws"); disableAWS {
			enableAWS = false
		}

		// Apply command-line override to configuration if enable-aws was explicitly set
		if cmd.Flags().Changed("enable-aws") || cmd.Flags().Changed("disable-aws") {
			if err := serverRepo.SetAWSSourceEnabled(enableAWS); err != nil {
				log.Warnf("Failed to save AWS source configuration: %v", err)
			}
		}

		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
