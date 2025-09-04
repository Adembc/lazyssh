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

package config

import (
	"os"
	"path/filepath"

	"github.com/Adembc/lazyssh/internal/core/ports"
)

type OSConfig struct {
	homeDir string
}

func NewOSConfig() ports.ConfigProvider {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return &OSConfig{homeDir: home}
}

func (c *OSConfig) HomeDir() string {
	return c.homeDir
}

func (c *OSConfig) ConfigPath(elems ...string) string {
	return filepath.Join(c.HomeDir(), ".lazyssh", filepath.Join(elems...))
}

func (c *OSConfig) LogPath(filename string) string {
	return c.ConfigPath("logs", filename)
}

func (c *OSConfig) GetEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}
