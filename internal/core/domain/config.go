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

package domain

import "path/filepath"

// Config represents the application configuration
type Config struct {
	// EnableAWSSource controls whether to load servers from AWS YAML configuration
	EnableAWSSource bool `yaml:"enable_aws_source"`

	// AWSServersPath specifies the path to the AWS servers YAML configuration file
	AWSServersPath string `yaml:"aws_servers_path"`
}

// DefaultConfig returns the default configuration with the provided config directory
func DefaultConfig(configDirPath string) Config {
	var defaultAWSPath string
	if configDirPath != "" {
		defaultAWSPath = filepath.Join(configDirPath, "aws-servers.yaml")
	} else {
		// Fallback to standard config directory if no config directory provided
		defaultAWSPath = "~/.config/lazyssh/aws-servers.yaml"
	}
	
	return Config{
		EnableAWSSource: true, // Enable by default
		AWSServersPath:  defaultAWSPath,
	}
}
