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
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

type serverService struct {
	serverRepository ports.ServerRepository
	logger           *zap.SugaredLogger
}

// NewServerService creates a new instance of serverService.
func NewServerService(logger *zap.SugaredLogger, sr ports.ServerRepository) *serverService {
	return &serverService{
		logger:           logger,
		serverRepository: sr,
	}
}

// ListServers returns a list of servers sorted with pinned on top.
func (s *serverService) ListServers(query string) ([]domain.Server, error) {
	servers, err := s.serverRepository.ListServers(query)
	if err != nil {
		s.logger.Errorw("failed to list servers", "error", err)
		return nil, err
	}

	// Sort: pinned first (PinnedAt non-zero), then by PinnedAt desc, then by Alias asc.
	sort.SliceStable(servers, func(i, j int) bool {
		pi := !servers[i].PinnedAt.IsZero()
		pj := !servers[j].PinnedAt.IsZero()
		if pi != pj {
			return pi
		}
		if pi && pj {
			return servers[i].PinnedAt.After(servers[j].PinnedAt)
		}
		return servers[i].Alias < servers[j].Alias
	})

	return servers, nil
}

// validateServer performs core validation of server fields.
func validateServer(srv domain.Server) error {
	if strings.TrimSpace(srv.Alias) == "" {
		return fmt.Errorf("alias is required")
	}
	if ok, _ := regexp.MatchString(`^[A-Za-z0-9_.-]+$`, srv.Alias); !ok {
		return fmt.Errorf("alias may contain letters, digits, dot, dash, underscore")
	}
	if strings.TrimSpace(srv.Host) == "" {
		return fmt.Errorf("Host/IP is required")
	}
	if ip := net.ParseIP(srv.Host); ip == nil {
		if strings.Contains(srv.Host, " ") {
			return fmt.Errorf("host must not contain spaces")
		}
		if ok, _ := regexp.MatchString(`^[A-Za-z0-9.-]+$`, srv.Host); !ok {
			return fmt.Errorf("host contains invalid characters")
		}
		if strings.HasPrefix(srv.Host, ".") || strings.HasSuffix(srv.Host, ".") {
			return fmt.Errorf("host must not start or end with a dot")
		}
		for _, lbl := range strings.Split(srv.Host, ".") {
			if lbl == "" {
				return fmt.Errorf("host must not contain empty labels")
			}
			if strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
				return fmt.Errorf("hostname labels must not start or end with a hyphen")
			}
		}
	}
	if srv.Port != 0 && (srv.Port < 1 || srv.Port > 65535) {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}
	return nil
}

// UpdateServer updates an existing server with new details.
func (s *serverService) UpdateServer(server domain.Server, newServer domain.Server) error {
	if err := validateServer(newServer); err != nil {
		s.logger.Warnw("validation failed on update", "error", err, "server", newServer)
		return err
	}
	err := s.serverRepository.UpdateServer(server, newServer)
	if err != nil {
		s.logger.Errorw("failed to update server", "error", err, "server", server)
	}
	return err
}

// AddServer adds a new server to the repository.
func (s *serverService) AddServer(server domain.Server) error {
	if err := validateServer(server); err != nil {
		s.logger.Warnw("validation failed on add", "error", err, "server", server)
		return err
	}
	err := s.serverRepository.AddServer(server)
	if err != nil {
		s.logger.Errorw("failed to add server", "error", err, "server", server)
	}
	return err
}

// DeleteServer removes a server from the repository.
func (s *serverService) DeleteServer(server domain.Server) error {
	err := s.serverRepository.DeleteServer(server)
	if err != nil {
		s.logger.Errorw("failed to delete server", "error", err, "server", server)
	}
	return err
}

// SetPinned sets or clears a pin timestamp for the server alias.
func (s *serverService) SetPinned(alias string, pinned bool) error {
	err := s.serverRepository.SetPinned(alias, pinned)
	if err != nil {
		s.logger.Errorw("failed to set pin state", "error", err, "alias", alias, "pinned", pinned)
	}
	return err
}

// SSH starts an interactive session to the given server using appropriate connection method.
func (s *serverService) SSH(alias string) error {
	// Get server details to determine connection type
	servers, err := s.serverRepository.ListServers("")
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	var server *domain.Server
	for _, srv := range servers {
		if srv.Alias == alias {
			server = &srv
			break
		}
	}

	if server == nil {
		return fmt.Errorf("server with alias '%s' not found", alias)
	}

	s.logger.Infow("connection start", "alias", alias, "type", server.ConnectionType)

	var cmd *exec.Cmd
	switch server.ConnectionType {
	case domain.ConnectionTypeAWS:
		cmd, err = s.buildAWSCommand(*server)
		if err != nil {
			return fmt.Errorf("failed to build AWS command: %w", err)
		}
	case domain.ConnectionTypeSSH:
		cmd = exec.Command("ssh", alias)
	default:
		// Fallback to SSH connection for unknown types
		cmd = exec.Command("ssh", alias)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		s.logger.Errorw("connection command failed", "alias", alias, "type", server.ConnectionType, "error", err)
		return err
	}

	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record connection metadata", "alias", alias, "error", err)
	}

	s.logger.Infow("connection end", "alias", alias, "type", server.ConnectionType)
	return nil
}

// Ping checks if the server is reachable on its SSH port.
func (s *serverService) Ping(server domain.Server) (bool, time.Duration, error) {
	start := time.Now()

	host, port, ok := resolveSSHDestination(server.Alias)
	if !ok {

		host = strings.TrimSpace(server.Host)
		if host == "" {
			host = server.Alias
		}
		if server.Port > 0 {
			port = server.Port
		} else {
			port = 22
		}
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return false, time.Since(start), err
	}
	_ = conn.Close()
	return true, time.Since(start), nil
}

// resolveSSHDestination uses `ssh -G <alias>` to extract HostName and Port from the user's SSH config.
// Returns host, port, ok where ok=false if resolution failed.
func resolveSSHDestination(alias string) (string, int, bool) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", 0, false
	}
	cmd := exec.Command("ssh", "-G", alias)
	out, err := cmd.Output()
	if err != nil {
		return "", 0, false
	}
	host := ""
	port := 0
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "hostname ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				host = parts[1]
			}
		}
		if strings.HasPrefix(line, "port ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if p, err := strconv.Atoi(parts[1]); err == nil {
					port = p
				}
			}
		}
	}
	if host == "" {
		host = alias
	}
	if port == 0 {
		port = 22
	}
	return host, port, true
}

// mapServerToAWSProfile maps server aliases to their corresponding AWS profiles
func mapServerToAWSProfile(serverAlias string) string {
	profileMap := map[string]string{
		"aws-dev1-g": "exp-dev1",
		"aws-dev1-b": "exp-dev1",
		"aws-prod-g": "exp-phi",
		"aws-prod-b": "exp-phi", 
		"aws-st-g":   "exp-st",
		"aws-st-b":   "exp-st",
		"aws-st2-g":  "exp-st2",
		"aws-st2-b":  "exp-st2",
		"aws-sandbox-b": "exp-sandbox",
		"cw-prod-g":  "clockwisemd",
		"cw-prod-b":  "clockwisemd",
		"cw-dev-g":   "clockwisemd",
		"cw-dev-b":   "clockwisemd",
		"cw-st-g":    "clockwisemd",
		"cw-st-b":    "clockwisemd",
		"cw-st2-g":   "clockwisemd",
		"cw-pe-dev-g": "clockwisemd",
	}
	
	if profile, exists := profileMap[serverAlias]; exists {
		return profile
	}
	
	// Default fallback - try to extract profile from server name pattern
	if strings.HasPrefix(serverAlias, "aws-dev1") {
		return "exp-dev1"
	} else if strings.HasPrefix(serverAlias, "aws-prod") {
		return "exp-phi"
	} else if strings.HasPrefix(serverAlias, "aws-st2") {
		return "exp-st2"
	} else if strings.HasPrefix(serverAlias, "aws-st") {
		return "exp-st"
	} else if strings.HasPrefix(serverAlias, "aws-sandbox") {
		return "exp-sandbox"
	} else if strings.HasPrefix(serverAlias, "cw-") {
		return "clockwisemd"
	}
	
	return "default" // fallback
}

// mapServerToInstanceName maps server aliases to their corresponding EC2 instance tag names
func mapServerToInstanceName(serverAlias string) string {
	// ClockwiseMD servers have different naming: cw-xxx-y -> clockwise-xxx-y-eks-bastion
	if strings.HasPrefix(serverAlias, "cw-") {
		// Convert cw-dev-g -> clockwise-dev-g-eks-bastion
		instanceName := strings.Replace(serverAlias, "cw-", "clockwise-", 1)
		return instanceName + "-eks-bastion"
	}
	
	// AWS servers use direct mapping: aws-xxx-y -> aws-xxx-y-eks-bastion
	return serverAlias + "-eks-bastion"
}

// mapServerToAWSRegion maps server aliases to their corresponding AWS regions
func mapServerToAWSRegion(serverAlias string) string {
	// Most servers use us-east-1, but st2 servers use us-east-2
	if strings.Contains(serverAlias, "st2") {
		return "us-east-2"
	}
	return "us-east-1" // default for most environments
}

// buildAWSCommand constructs a command to execute AWS SSM session using profiles instead of assume
func (s *serverService) buildAWSCommand(server domain.Server) (*exec.Cmd, error) {
	if server.ConnectionType != domain.ConnectionTypeAWS {
		return nil, fmt.Errorf("server is not an AWS connection type")
	}

	// Use server configuration from YAML (preferred) or fallback to legacy mappings
	awsProfile := server.AWSProfile
	awsRegion := server.AWSRegion
	
	// Fallback to legacy mappings if YAML config is incomplete
	if awsProfile == "" {
		awsProfile = mapServerToAWSProfile(server.Alias)
	}
	if awsRegion == "" {
		awsRegion = mapServerToAWSRegion(server.Alias)
	}
	
	s.logger.Infow("building AWS SSM command",
		"server_alias", server.Alias,
		"aws_profile", awsProfile,
		"aws_region", awsRegion,
		"host", server.Host,
		"ec2_tag_filter", server.EC2TagFilter,
		"ssm_document", server.SSMDocument,
		"source", server.Source)

	// Determine instance ID or EC2 filters based on configuration
	var instanceLookupScript string
	var instanceIdentifier string
	
	if server.Host != "" {
		// Direct instance ID connection
		instanceLookupScript = fmt.Sprintf(`INSTANCE_ID="%s"`, server.Host)
		instanceIdentifier = server.Host
	} else if server.EC2TagFilter != "" {
		// EC2 tag filter connection
		instanceLookupScript = fmt.Sprintf(`
echo "🔍 Finding EC2 instance using filters: %s"
INSTANCE_ID=$(aws --profile %s --region %s ec2 describe-instances \
    --filters %s "Name=instance-state-name,Values=running" \
    --query "Reservations[0].Instances[0].InstanceId" \
    --output text 2>/dev/null)

if [[ "$INSTANCE_ID" == "None" || "$INSTANCE_ID" == "" || "$INSTANCE_ID" == "null" ]]; then
    echo "✗ Could not find running instance with filters: %s"
    echo ""
    echo "🔍 Available instances matching filters:"
    aws --profile %s --region %s ec2 describe-instances \
        --filters %s \
        --query "Reservations[].Instances[].[Tags[?Key=='Name'].Value|[0], State.Name, InstanceId]" \
        --output table 2>/dev/null || echo "No instances found"
    exit 1
fi`, server.EC2TagFilter, awsProfile, awsRegion, server.EC2TagFilter, server.EC2TagFilter, awsProfile, awsRegion, server.EC2TagFilter)
		instanceIdentifier = server.EC2TagFilter
	} else {
		// Fallback to legacy instance name mapping
		instanceName := mapServerToInstanceName(server.Alias)
		instanceLookupScript = fmt.Sprintf(`
echo "🔍 Finding EC2 instance: %s (legacy mode)"
INSTANCE_ID=$(aws --profile %s --region %s ec2 describe-instances \
    --filters "Name=tag:Name,Values=%s" "Name=instance-state-name,Values=running" \
    --query "Reservations[0].Instances[0].InstanceId" \
    --output text 2>/dev/null)

if [[ "$INSTANCE_ID" == "None" || "$INSTANCE_ID" == "" || "$INSTANCE_ID" == "null" ]]; then
    echo "✗ Could not find running instance: %s"
    echo ""
    echo "🔍 Available instances with this name pattern:"
    aws --profile %s --region %s ec2 describe-instances \
        --filters "Name=tag:Name,Values=%s" \
        --query "Reservations[].Instances[].[Tags[?Key=='Name'].Value|[0], State.Name, InstanceId]" \
        --output table 2>/dev/null || echo "No instances found"
    exit 1
fi`, instanceName, awsProfile, awsRegion, instanceName, instanceName, awsProfile, awsRegion, instanceName)
		instanceIdentifier = instanceName
	}

	// Set SSM document and parameters
	ssmDocument := server.SSMDocument
	if ssmDocument == "" {
		ssmDocument = "AWS-StartInteractiveCommand" // default for backward compatibility
	}
	
	// Build SSM command parameters
	var ssmParameters string
	if server.SSMCommand != "" {
		ssmParameters = fmt.Sprintf(` --parameters %s`, server.SSMCommand)
	} else if ssmDocument == "AWS-StartInteractiveCommand" {
		ssmParameters = ` --parameters command="sudo su - ubuntu"`
	}

	// Create a shell script that uses YAML configuration
	shellScript := fmt.Sprintf(`
#!/bin/bash

echo "=================== LazySsh AWS Connection ==================="
echo "Server: %s"
echo "AWS Profile: %s" 
echo "AWS Region: %s"
echo "Target: %s"
echo "SSM Document: %s"
echo "Source: %s"
echo "=============================================================="

# Step 1: Verify AWS credentials
echo "🔍 Verifying AWS credentials..."
if ! aws --profile %s --region %s sts get-caller-identity >/dev/null 2>&1; then
    echo "✗ AWS credentials verification failed for profile: %s"
    echo "Please check your AWS profile configuration:"
    echo "  aws configure list --profile %s"
    exit 1
fi
echo "✓ AWS credentials verified for profile: %s"

# Step 2: Determine instance ID
echo ""
%s

echo "✓ Found instance: $INSTANCE_ID"

# Step 3: Start SSM session
echo ""
echo "🚀 Starting SSM session..."
echo "Command: aws --profile %s --region %s ssm start-session --target $INSTANCE_ID --document-name %s%s"
echo ""

exec aws --profile %s --region %s ssm start-session \
    --target "$INSTANCE_ID" \
    --document-name %s%s
`, server.Alias, awsProfile, awsRegion, instanceIdentifier, ssmDocument, server.Source, 
   awsProfile, awsRegion, awsProfile, awsProfile, awsProfile,
   instanceLookupScript,
   awsProfile, awsRegion, ssmDocument, ssmParameters,
   awsProfile, awsRegion, ssmDocument, ssmParameters)

	// Create command to execute the shell script
	// #nosec G204 - shellScript is constructed from controlled inputs (validated server alias and mapped AWS profile)
	cmd := exec.Command("bash", "-c", shellScript)

	// Preserve current environment (including any existing AWS variables)
	cmd.Env = os.Environ()

	return cmd, nil
}
