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
	"fmt"
	"os"
	"path/filepath"

	"github.com/kjm0001/lazyssh/internal/core/domain"
)

// AWSConfigManager handles loading AWS server configurations from YAML
type AWSConfigManager struct {
	yamlConfigPath string
}

// newAWSConfigManager creates a new AWS config manager
func newAWSConfigManager(yamlConfigPath string) *AWSConfigManager {
	return &AWSConfigManager{
		yamlConfigPath: yamlConfigPath,
	}
}

// parseServers loads AWS servers from YAML configuration
func (m *AWSConfigManager) parseServers() ([]domain.Server, error) {
	return m.parseYAMLConfig()
}

// parseYAMLConfig loads servers from YAML configuration
func (m *AWSConfigManager) parseYAMLConfig() ([]domain.Server, error) {
	file, err := os.Open(m.yamlConfigPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	parser := &AWSYamlParser{}
	return parser.Parse(file)
}

// getConfigStatus returns information about YAML configuration file availability
func (m *AWSConfigManager) getConfigStatus() map[string]bool {
	status := make(map[string]bool)
	
	// Check YAML config
	if _, err := os.Stat(m.yamlConfigPath); err == nil {
		status["yaml_config"] = true
	} else {
		status["yaml_config"] = false
	}
	
	return status
}

// initializeYAMLConfig creates a sample YAML configuration file if it doesn't exist
func (m *AWSConfigManager) initializeYAMLConfig() error {
	// Check if YAML config already exists
	if _, err := os.Stat(m.yamlConfigPath); err == nil {
		return nil // Already exists
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(m.yamlConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create sample configuration
	sampleConfig := `# AWS Servers Configuration for lazyssh
# This file defines AWS servers that can be accessed via AWS SSM

aws_servers:
  # Example server configuration
  # - alias: my-server
  #   profile: my-aws-profile
  #   region: us-east-1
  #   connection_type: ssm
  #   target_selection:
  #     method: ec2_tag_filter
  #     filters:
  #       - name: "tag:Name"
  #         values: ["my-server-name"]
  #   ssm:
  #     document: AWS-StartSSHSession
  #     port_number: 22
  #   tags: ["environment", "role"]
  #   description: "Description of the server"

# Global AWS Configuration
aws_config:
  default_region: us-east-1
  default_ssm_document: AWS-StartSSHSession
  timeout: 30
  retry_attempts: 3`

	if err := os.WriteFile(m.yamlConfigPath, []byte(sampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to create sample YAML config: %w", err)
	}
	
	return nil
}

