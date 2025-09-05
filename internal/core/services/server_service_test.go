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

package services

import (
	"strings"
	"testing"

	"github.com/kjm0001/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestServerService_buildAWSCommand(t *testing.T) {
	logger := zap.NewNop().Sugar()
	service := &serverService{
		logger: logger,
	}

	tests := []struct {
		name        string
		server      domain.Server
		expectError bool
		expectAlias string
	}{
		{
			name: "Valid AWS server",
			server: domain.Server{
				Alias:          "aws-dev1-g",
				ConnectionType: domain.ConnectionTypeAWS,
				Source:         "aws_func",
				AWSProfile:     "exp-dev1",
				AWSRegion:      "us-east-1",
			},
			expectError: false,
			expectAlias: "aws-dev1-g",
		},
		{
			name: "SSH server should fail",
			server: domain.Server{
				Alias:          "ssh-server",
				ConnectionType: domain.ConnectionTypeSSH,
				Source:         "ssh_config",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := service.buildAWSCommand(tt.server)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cmd == nil {
				t.Errorf("Expected command but got nil")
				return
			}

			// Verify the command is bash with -c flag
			if cmd.Path != "/bin/bash" && cmd.Path != "bash" {
				// On some systems bash might be in different locations
				args := cmd.Args
				if len(args) == 0 || args[0] != "bash" {
					t.Errorf("Expected bash command, got %v", cmd.Args)
				}
			}

			// Verify the command contains the function alias
			cmdStr := strings.Join(cmd.Args, " ")
			if !strings.Contains(cmdStr, tt.expectAlias) {
				t.Errorf("Expected command to contain alias '%s', got: %s", tt.expectAlias, cmdStr)
			}

			// Verify the shell script contains expected elements
			if !strings.Contains(cmdStr, "source") {
				t.Errorf("Expected shell script to contain 'source', got: %s", cmdStr)
			}

			if !strings.Contains(cmdStr, "aws-ssh.func") {
				t.Errorf("Expected shell script to contain 'aws-ssh.func', got: %s", cmdStr)
			}

			if !strings.Contains(cmdStr, "declare -f") {
				t.Errorf("Expected shell script to contain 'declare -f', got: %s", cmdStr)
			}
		})
	}
}

func TestServerService_buildAWSCommand_InvalidConnectionType(t *testing.T) {
	logger := zap.NewNop().Sugar()
	service := &serverService{
		logger: logger,
	}

	server := domain.Server{
		Alias:          "test-server",
		ConnectionType: domain.ConnectionTypeSSH,
		Source:         "ssh_config",
	}

	_, err := service.buildAWSCommand(server)
	if err == nil {
		t.Errorf("Expected error for non-AWS connection type")
	}

	expectedError := "server is not an AWS connection type"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedError, err.Error())
	}
}
