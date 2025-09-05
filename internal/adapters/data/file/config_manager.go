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

package file

import (
	"os"
	"path/filepath"

	"github.com/kjm0001/lazyssh/internal/core/domain"
	"gopkg.in/yaml.v3"
)

type configManager struct {
	filePath      string
	configDirPath string
}

func newConfigManager(filePath string) *configManager {
	return &configManager{
		filePath:      filePath,
		configDirPath: filepath.Dir(filePath),
	}
}

func (cm *configManager) load() (domain.Config, error) {
	// If config file doesn't exist, create default config and save it
	if _, err := os.Stat(cm.filePath); os.IsNotExist(err) {
		defaultConfig := domain.DefaultConfig(cm.configDirPath)
		// Save the default config to the file
		if saveErr := cm.save(defaultConfig); saveErr != nil {
			// If we can't save, still return the default config
			return defaultConfig, nil
		}
		return defaultConfig, nil
	}

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return domain.DefaultConfig(cm.configDirPath), err
	}

	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return domain.DefaultConfig(cm.configDirPath), err
	}

	return config, nil
}

func (cm *configManager) save(config domain.Config) error {
	if err := cm.ensureDirectory(); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Write YAML data to file
	if err := os.WriteFile(cm.filePath, data, 0o600); err != nil {
		return err
	}

	return nil
}

func (cm *configManager) ensureDirectory() error {
	dir := filepath.Dir(cm.filePath)
	return os.MkdirAll(dir, 0o700)
}
