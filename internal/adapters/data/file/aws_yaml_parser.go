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
	"io"
	"strings"
	"time"

	"github.com/kjm0001/lazyssh/internal/core/domain"
	"gopkg.in/yaml.v3"
)

// AWSYamlConfig represents the complete YAML configuration structure
type AWSYamlConfig struct {
	AWSServers []AWSServerConfig `yaml:"aws_servers"`
	AWSConfig  AWSGlobalConfig   `yaml:"aws_config"`
}

// AWSServerConfig represents a single AWS server configuration
type AWSServerConfig struct {
	Alias           string                `yaml:"alias"`
	Profile         string                `yaml:"profile"`
	Region          string                `yaml:"region"`
	ConnectionType  string                `yaml:"connection_type"`
	TargetSelection AWSTargetSelection    `yaml:"target_selection"`
	SSM             AWSSSMConfig          `yaml:"ssm"`
	Tags            []string              `yaml:"tags,omitempty"`
	Description     string                `yaml:"description,omitempty"`
}

// AWSTargetSelection defines how to select the target EC2 instance
type AWSTargetSelection struct {
	Method     string         `yaml:"method"` // "ec2_tag_filter", "instance_id"
	InstanceID string         `yaml:"instance_id,omitempty"`
	Filters    []AWSTagFilter `yaml:"filters,omitempty"`
}

// AWSTagFilter represents EC2 tag filters
type AWSTagFilter struct {
	Name   string   `yaml:"name"`
	Values []string `yaml:"values"`
}

// AWSSSMConfig represents SSM session configuration
type AWSSSMConfig struct {
	Document   string `yaml:"document"`
	LocalPort  int    `yaml:"local_port,omitempty"`
	RemotePort int    `yaml:"remote_port,omitempty"`
	PortNumber int    `yaml:"port_number,omitempty"`
}

// AWSGlobalConfig represents global AWS configuration
type AWSGlobalConfig struct {
	DefaultRegion        string `yaml:"default_region"`
	DefaultSSMDocument   string `yaml:"default_ssm_document"`
	Timeout             int    `yaml:"timeout"`
	RetryAttempts       int    `yaml:"retry_attempts"`
}

// AWSYamlParser parses AWS server configurations from YAML format
type AWSYamlParser struct{}

// Parse reads YAML configuration and converts it to domain.Server objects
func (p *AWSYamlParser) Parse(reader io.Reader) ([]domain.Server, error) {
	var config AWSYamlConfig
	
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var servers []domain.Server
	for _, awsServer := range config.AWSServers {
		server, err := p.convertToServer(awsServer, &config.AWSConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to convert server '%s': %w", awsServer.Alias, err)
		}
		servers = append(servers, *server)
	}

	return servers, nil
}

// convertToServer converts AWSServerConfig to domain.Server
func (p *AWSYamlParser) convertToServer(config AWSServerConfig, globalConfig *AWSGlobalConfig) (*domain.Server, error) {
	if config.Alias == "" {
		return nil, fmt.Errorf("server alias is required")
	}
	if config.Profile == "" {
		return nil, fmt.Errorf("AWS profile is required for server '%s'", config.Alias)
	}

	server := &domain.Server{
		Alias:          config.Alias,
		ConnectionType: domain.ConnectionTypeAWS,
		Source:         "aws_yaml",
		AWSProfile:     config.Profile,
		Tags:           config.Tags,
		LastSeen:       time.Now(), // Will be updated from metadata
	}

	// Set region (use provided or global default)
	if config.Region != "" {
		server.AWSRegion = config.Region
	} else if globalConfig.DefaultRegion != "" {
		server.AWSRegion = globalConfig.DefaultRegion
	} else {
		server.AWSRegion = "us-east-1" // fallback default
	}

	// Handle target selection
	switch config.TargetSelection.Method {
	case "instance_id":
		if config.TargetSelection.InstanceID == "" {
			return nil, fmt.Errorf("instance_id is required when method is 'instance_id'")
		}
		server.Host = config.TargetSelection.InstanceID
		
	case "ec2_tag_filter":
		if len(config.TargetSelection.Filters) == 0 {
			return nil, fmt.Errorf("filters are required when method is 'ec2_tag_filter'")
		}
		server.EC2TagFilter = p.buildTagFilterString(config.TargetSelection.Filters)
		
	default:
		return nil, fmt.Errorf("unsupported target selection method: '%s'", config.TargetSelection.Method)
	}

	// Set SSM document
	if config.SSM.Document != "" {
		server.SSMDocument = config.SSM.Document
	} else if globalConfig.DefaultSSMDocument != "" {
		server.SSMDocument = globalConfig.DefaultSSMDocument
	} else {
		server.SSMDocument = "AWS-StartSSHSession" // default
	}

	// Build SSM command for port forwarding if specified
	if config.SSM.LocalPort > 0 && config.SSM.RemotePort > 0 {
		server.SSMCommand = fmt.Sprintf("localPortNumber=%d,remotePortNumber=%d", 
			config.SSM.LocalPort, config.SSM.RemotePort)
	} else if config.SSM.PortNumber > 0 {
		server.SSMCommand = fmt.Sprintf("portNumber=%d", config.SSM.PortNumber)
	} else if server.SSMDocument == "AWS-StartInteractiveCommand" {
		// Default parameters for interactive command
		server.SSMCommand = `command="sudo su - ubuntu"`
	}

	return server, nil
}

// buildTagFilterString converts tag filters to the format expected by AWS CLI
func (p *AWSYamlParser) buildTagFilterString(filters []AWSTagFilter) string {
	var filterStrings []string
	
	for _, filter := range filters {
		if len(filter.Values) > 0 {
			values := strings.Join(filter.Values, ",")
			filterStrings = append(filterStrings, fmt.Sprintf("Name=%s,Values=%s", filter.Name, values))
		}
	}
	
	return strings.Join(filterStrings, " ")
}