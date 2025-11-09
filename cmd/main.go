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
	"io"
	"os"
	"path/filepath"

	"github.com/Adembc/lazyssh/internal/adapters/data/ssh_config_file"
	"github.com/Adembc/lazyssh/internal/logger"

	"github.com/Adembc/lazyssh/internal/adapters/ui"
	"github.com/Adembc/lazyssh/internal/core/services"
	"github.com/spf13/cobra"
)

var (
	version       = "develop"
	gitCommit     = "unknown"
	sshConfigFile string

	rootCmd = &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				// nolint:gocritic // exitAfterDefer: ensure immediate exit on unrecoverable error
				os.Exit(1)
			}

			if sshConfigFile == "" {
				sshConfigFile = filepath.Join(home, ".ssh", "config")
			} else {
				stat, err := os.Stat(sshConfigFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
					os.Exit(1)
				}
				if stat.Mode()&os.ModeType != 0 {
					f, err := os.CreateTemp("", "tmpfile-")
					if err != nil {
						log.Fatal(err)
					}

					// close and remove the temporary file at the end of the program
					defer f.Close()
					defer os.Remove(f.Name())

					// write data to the temporary file
					fd, err := os.Open(sshConfigFile)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
						os.Exit(1)
					}
					defer fd.Close()

					// Read the entire contents at once
					content, err := io.ReadAll(fd)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
						os.Exit(1)
					}
					if _, err := f.WriteString(string(content)); err != nil {
						log.Fatal(err)
					}

					sshConfigFile = f.Name()
				}
			}

			metaDataFile := filepath.Join(home, ".lazyssh", "metadata.json")
			serverRepo := ssh_config_file.NewRepository(log, sshConfigFile, metaDataFile)
			serverService := services.NewServerService(log, serverRepo)
			tui := ui.NewTUI(log, serverService, version, gitCommit)

			return tui.Run()
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&sshConfigFile, "sshconfig", "", "path to ssh config file (default: ~/.ssh/config)")

	rootCmd.SilenceUsage = true
}
