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

import "time"

// ConnectionType represents the type of connection (SSH or AWS SSM)
type ConnectionType string

const (
	ConnectionTypeSSH ConnectionType = "ssh"
	ConnectionTypeAWS ConnectionType = "aws"
)

type Server struct {
	// Common fields for all connection types
	Alias    string
	Host     string
	User     string
	Port     int
	Key      string
	Tags     []string
	LastSeen time.Time
	PinnedAt time.Time
	SSHCount int

	// Connection type and source tracking
	ConnectionType ConnectionType
	Source         string // "ssh_config", "aws_func", etc.

	// AWS-specific fields (only used when ConnectionType == ConnectionTypeAWS)
	AWSProfile   string
	AWSRegion    string
	EC2TagFilter string
	SSMDocument  string
	SSMCommand   string
}
