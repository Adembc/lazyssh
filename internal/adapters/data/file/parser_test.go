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
	"strings"
	"testing"

	"github.com/kjm0001/lazyssh/internal/core/domain"
)

func TestAWSFunctionParser_Parse(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedCount   int
		expectedServers []domain.Server
	}{
		{
			name: "Parse basic AWS function",
			input: `function exp-dev-web-01 () {
   aws ssm start-session --target $(aws ec2 describe-instances --profile=exp-dev1 --region=us-east-1 --filters "Name=tag:Name,Values=web-01" --query "Reservations[0].Instances[0].InstanceId" --output text) --profile=exp-dev1 --region=us-east-1
}`,
			expectedCount: 1,
			expectedServers: []domain.Server{
				{
					Alias:          "exp-dev-web-01",
					ConnectionType: domain.ConnectionTypeAWS,
					Source:         "aws_func",
					AWSProfile:     "exp-dev1",
					AWSRegion:      "us-east-1",
					EC2TagFilter:   "Name=tag:Name,Values=web-01",
					SSMDocument:    "AWS-StartSSHSession",
				},
			},
		},
		{
			name: "Parse AWS function with custom SSM document",
			input: `function exp-prod-api-server () {
   aws ssm start-session --target i-1234567890abcdef0 --profile=exp-phi --region=us-west-2 --document-name AWS-StartPortForwardingSession
}`,
			expectedCount: 1,
			expectedServers: []domain.Server{
				{
					Alias:          "exp-prod-api-server",
					Host:           "i-1234567890abcdef0",
					ConnectionType: domain.ConnectionTypeAWS,
					Source:         "aws_func",
					AWSProfile:     "exp-phi",
					AWSRegion:      "us-west-2",
					SSMDocument:    "AWS-StartPortForwardingSession",
				},
			},
		},
		{
			name: "Skip non-AWS functions",
			input: `function regular-function () {
   echo "This is not an AWS function"
}

function exp-dev-db () {
   aws ssm start-session --target i-abcdef1234567890 --profile=exp-dev1 --region=us-east-1
}`,
			expectedCount: 1,
			expectedServers: []domain.Server{
				{
					Alias:          "exp-dev-db",
					Host:           "i-abcdef1234567890",
					ConnectionType: domain.ConnectionTypeAWS,
					Source:         "aws_func",
					AWSProfile:     "exp-dev1",
					AWSRegion:      "us-east-1",
					SSMDocument:    "AWS-StartSSHSession",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &AWSFunctionParser{}
			reader := strings.NewReader(tt.input)

			servers, err := parser.Parse(reader)
			if err != nil {
				t.Errorf("Parse() error = %v, want nil", err)
				return
			}

			if len(servers) != tt.expectedCount {
				t.Errorf("Parse() returned %d servers, want %d", len(servers), tt.expectedCount)
				return
			}

			for i, expected := range tt.expectedServers {
				if i >= len(servers) {
					t.Errorf("Expected server %d not found", i)
					continue
				}

				server := servers[i]

				if server.Alias != expected.Alias {
					t.Errorf("Server %d Alias = %q, want %q", i, server.Alias, expected.Alias)
				}
				if server.ConnectionType != expected.ConnectionType {
					t.Errorf("Server %d ConnectionType = %q, want %q", i, server.ConnectionType, expected.ConnectionType)
				}
				if server.Source != expected.Source {
					t.Errorf("Server %d Source = %q, want %q", i, server.Source, expected.Source)
				}
				if server.AWSProfile != expected.AWSProfile {
					t.Errorf("Server %d AWSProfile = %q, want %q", i, server.AWSProfile, expected.AWSProfile)
				}
				if server.AWSRegion != expected.AWSRegion {
					t.Errorf("Server %d AWSRegion = %q, want %q", i, server.AWSRegion, expected.AWSRegion)
				}
				if server.EC2TagFilter != expected.EC2TagFilter {
					t.Errorf("Server %d EC2TagFilter = %q, want %q", i, server.EC2TagFilter, expected.EC2TagFilter)
				}
				if server.SSMDocument != expected.SSMDocument {
					t.Errorf("Server %d SSMDocument = %q, want %q", i, server.SSMDocument, expected.SSMDocument)
				}
				if server.Host != expected.Host {
					t.Errorf("Server %d Host = %q, want %q", i, server.Host, expected.Host)
				}
			}
		})
	}
}

func TestAWSFunctionParser_ParseEmptyInput(t *testing.T) {
	parser := &AWSFunctionParser{}
	reader := strings.NewReader("")

	servers, err := parser.Parse(reader)
	if err != nil {
		t.Errorf("Parse() error = %v, want nil", err)
		return
	}

	if len(servers) != 0 {
		t.Errorf("Parse() returned %d servers, want 0", len(servers))
	}
}
