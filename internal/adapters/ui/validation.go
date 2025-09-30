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

package ui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// fieldValidator contains validation rules for SSH configuration fields
type fieldValidator struct {
	Required bool
	Pattern  *regexp.Regexp
	Validate func(string) error
	Message  string
}

// ValidationState tracks validation errors for each field
type ValidationState struct {
	errors map[string]string
	mu     sync.RWMutex
}

// NewValidationState creates a new validation state
func NewValidationState() *ValidationState {
	return &ValidationState{
		errors: make(map[string]string),
	}
}

// SetError sets or clears an error for a field
func (v *ValidationState) SetError(field, errMsg string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if errMsg == "" {
		delete(v.errors, field)
	} else {
		v.errors[field] = errMsg
	}
}

// GetError gets the error for a specific field
func (v *ValidationState) GetError(field string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.errors[field]
}

// HasErrors checks if there are any validation errors
func (v *ValidationState) HasErrors() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.errors) > 0
}

// GetErrorCount returns the number of validation errors
func (v *ValidationState) GetErrorCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.errors)
}

// GetAllErrors returns all validation errors in field order
func (v *ValidationState) GetAllErrors() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Define field order for consistent error display
	fieldOrder := []string{
		"Alias", "Host", "Port", "User", "Keys", "Tags",
		"ConnectTimeout", "ConnectionAttempts", "ServerAliveInterval", "ServerAliveCountMax",
		"IPQoS", "BindAddress", "LocalForward", "RemoteForward", "DynamicForward",
		"NumberOfPasswordPrompts", "CanonicalizeMaxDots", "EscapeChar",
	}

	// Create a set for O(1) lookups
	fieldOrderSet := make(map[string]bool, len(fieldOrder))
	for _, field := range fieldOrder {
		fieldOrderSet[field] = true
	}

	errors := make([]string, 0, len(v.errors))

	// Add errors in defined order
	for _, field := range fieldOrder {
		if err, exists := v.errors[field]; exists {
			errors = append(errors, fmt.Sprintf("%s: %s", field, err))
		}
	}

	// Add any other errors not in the defined order
	for field, err := range v.errors {
		if !fieldOrderSet[field] {
			errors = append(errors, fmt.Sprintf("%s: %s", field, err))
		}
	}

	return errors
}

// Clear removes all validation errors
func (v *ValidationState) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.errors = make(map[string]string)
}

// invalidHostChars contains characters that are not allowed in hostnames
const invalidHostChars = "@#$%^&*()=+[]{}|\\;:'\"<>,?/"

// invalidAddressChars contains characters that are not allowed in bind addresses
const invalidAddressChars = "@#$%^&()=+{}|\\;:'\"<>,?/"

// GetFieldValidators returns validation rules for SSH configuration fields
func GetFieldValidators() map[string]fieldValidator {
	return GetFieldValidatorsWithContext("", nil)
}

// GetFieldValidatorsWithContext returns validation rules with context-aware validators
// originalAlias is used to exclude the current alias when editing
// existingAliases is used to check for duplicates
func GetFieldValidatorsWithContext(originalAlias string, existingAliases []string) map[string]fieldValidator {
	validators := make(map[string]fieldValidator)

	// Basic fields
	validators["Alias"] = fieldValidator{
		Required: true,
		Pattern:  regexp.MustCompile(`^[a-zA-Z0-9._-]+$`),
		Validate: createAliasValidator(originalAlias, existingAliases),
		Message:  "Alias is required and can only contain letters, numbers, dots, hyphens, and underscores",
	}
	validators["Host"] = fieldValidator{
		Required: true,
		Validate: validateHost,
		Message:  "Host is required and must be a valid hostname or IP address",
	}
	validators["Port"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^([1-9]\d{0,4})$`),
		Validate: validatePort,
		Message:  "Port must be between 1 and 65535",
	}
	validators["User"] = fieldValidator{
		Pattern: regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`),
		Message: "User must start with a letter and contain only letters, numbers, dots, hyphens, and underscores",
	}
	validators["Keys"] = fieldValidator{
		Validate: validateKeyPaths,
		Message:  "Key file not found or not accessible",
	}

	// Connection fields
	validators["ConnectTimeout"] = fieldValidator{
		Validate: validateConnectTimeout,
		Message:  "ConnectTimeout must be a positive number or 'none'",
	}
	validators["ConnectionAttempts"] = fieldValidator{
		Pattern: regexp.MustCompile(`^[1-9]\d*$`),
		Message: "ConnectionAttempts must be a positive number",
	}
	validators["ServerAliveInterval"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validateNonNegativeNumber,
		Message:  "ServerAliveInterval must be a non-negative number",
	}
	validators["ServerAliveCountMax"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validateNonNegativeNumber,
		Message:  "ServerAliveCountMax must be a non-negative number",
	}
	validators["IPQoS"] = fieldValidator{
		Validate: validateIPQoS,
		Message:  "IPQoS must be valid QoS values (e.g., 'af21 cs1', 'lowdelay', 'ef')",
	}

	// Address and forwarding fields
	validators["BindAddress"] = fieldValidator{
		Validate: validateBindAddress,
		Message:  "BindAddress must be a valid IP address, hostname, or '*'",
	}
	validators["LocalForward"] = fieldValidator{
		Validate: validatePortForward,
		Message:  "LocalForward must be in format '[bind_address:]port:host:hostport'",
	}
	validators["RemoteForward"] = fieldValidator{
		Validate: validatePortForward,
		Message:  "RemoteForward must be in format '[bind_address:]port:host:hostport'",
	}
	validators["DynamicForward"] = fieldValidator{
		Validate: validateDynamicForward,
		Message:  "DynamicForward must be in format '[bind_address:]port'",
	}

	// Authentication fields
	validators["NumberOfPasswordPrompts"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validatePasswordPrompts,
		Message:  "NumberOfPasswordPrompts must be between 0 and 10",
	}

	// Advanced fields
	validators["CanonicalizeMaxDots"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validateNonNegativeNumber,
		Message:  "CanonicalizeMaxDots must be a non-negative number",
	}
	validators["EscapeChar"] = fieldValidator{
		Validate: validateEscapeChar,
		Message:  "EscapeChar must be a single character, 'none', or ^X format (e.g., ^A)",
	}

	// Security fields
	validators["UserKnownHostsFile"] = fieldValidator{
		Validate: validateKnownHostsFiles,
		Message:  "Known hosts file not found or not accessible",
	}

	return validators
}

// validatePort validates port number
func validatePort(value string) error {
	if value == "" {
		return nil // Port is optional
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid port number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// validateConnectTimeout validates connection timeout
func validateConnectTimeout(value string) error {
	if value == "" || value == "none" {
		return nil
	}
	timeout, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid timeout value")
	}
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive or 'none'")
	}
	return nil
}

// validateNonNegativeNumber validates that a value is a non-negative number
func validateNonNegativeNumber(value string) error {
	if value == "" {
		return nil
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid number")
	}
	if num < 0 {
		return fmt.Errorf("must be non-negative")
	}
	return nil
}

// validatePasswordPrompts validates NumberOfPasswordPrompts
func validatePasswordPrompts(value string) error {
	if value == "" {
		return nil
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid number")
	}
	if num < 0 || num > 10 {
		return fmt.Errorf("must be between 0 and 10")
	}
	return nil
}

// validateEscapeChar validates escape character format
func validateEscapeChar(value string) error {
	if value == "" || value == "none" || value == "~" {
		return nil
	}
	// Support ^X format (Ctrl+X)
	if len(value) == 2 && value[0] == '^' {
		char := value[1]
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
			return nil
		}
	}
	// Single printable character
	if len(value) == 1 && value[0] >= 32 && value[0] <= 126 {
		return nil
	}
	return fmt.Errorf("invalid escape character format")
}

// validateIPQoS validates IPQoS values
func validateIPQoS(value string) error {
	if value == "" {
		return nil
	}
	validValues := map[string]bool{
		"af11": true, "af12": true, "af13": true,
		"af21": true, "af22": true, "af23": true,
		"af31": true, "af32": true, "af33": true,
		"af41": true, "af42": true, "af43": true,
		"cs0": true, "cs1": true, "cs2": true, "cs3": true,
		"cs4": true, "cs5": true, "cs6": true, "cs7": true,
		"ef": true, "le": true,
		"lowdelay": true, "throughput": true, "reliability": true, "none": true,
	}
	// Can be single value or two space-separated values
	parts := strings.Fields(value)
	if len(parts) > 2 {
		return fmt.Errorf("IPQoS accepts at most 2 values")
	}
	for _, part := range parts {
		if !validValues[strings.ToLower(part)] {
			return fmt.Errorf("invalid IPQoS value: %s", part)
		}
	}
	return nil
}

// validateFilePath validates a single file path for existence and readability
func validateFilePath(path string) (exists bool, accessible bool, isDir bool) {
	// Get home directory for tilde expansion
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	// Expand tilde notation
	expandedPath := path
	if strings.HasPrefix(path, "~/") && homeDir != "" {
		expandedPath = filepath.Join(homeDir, path[2:])
	} else if strings.HasPrefix(path, "~") && homeDir != "" {
		// Handle ~ alone
		expandedPath = homeDir
	}

	// Check if file exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, false // File doesn't exist
		}
		// Permission denied or other error
		return true, false, false // File exists but not accessible
	}

	// Check if it's a directory
	if info.IsDir() {
		return true, true, true
	}

	// Check if file is readable
	// #nosec G304 - expandedPath is validated user input
	file, err := os.Open(expandedPath)
	if err != nil {
		return true, false, false // File exists but not readable
	}
	_ = file.Close()

	return true, true, false // File exists and is readable
}

// buildFileValidationError builds an error message from invalid and inaccessible file paths
func buildFileValidationError(invalidPaths, inaccessiblePaths []string) error {
	var errors []string
	if len(invalidPaths) > 0 {
		errors = append(errors, fmt.Sprintf("file(s) not found: %s", strings.Join(invalidPaths, ", ")))
	}
	if len(inaccessiblePaths) > 0 {
		errors = append(errors, fmt.Sprintf("file(s) not accessible: %s", strings.Join(inaccessiblePaths, ", ")))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// validateFilePaths validates multiple file paths with a custom separator
func validateFilePaths(files string, separator string) error {
	if files == "" {
		return nil
	}
	// Check for invalid characters first, before trimming
	if strings.ContainsAny(files, "\n\r\t") {
		return fmt.Errorf("file path contains invalid characters")
	}

	var paths []string
	if separator == " " {
		// For space separator, use Fields to handle multiple spaces
		paths = strings.Fields(files)
	} else {
		// For other separators like comma
		paths = strings.Split(files, separator)
	}

	var invalidPaths []string
	var inaccessiblePaths []string

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		exists, accessible, isDir := validateFilePath(path)

		switch {
		case !exists:
			invalidPaths = append(invalidPaths, path)
		case isDir:
			invalidPaths = append(invalidPaths, fmt.Sprintf("%s (is a directory)", path))
		case !accessible:
			inaccessiblePaths = append(inaccessiblePaths, path)
		}
	}

	return buildFileValidationError(invalidPaths, inaccessiblePaths)
}

// validateKeyPaths validates SSH key file paths (comma-separated)
func validateKeyPaths(keys string) error {
	return validateFilePaths(keys, ",")
}

// validateKnownHostsFiles validates known_hosts file paths (space-separated)
func validateKnownHostsFiles(files string) error {
	// Empty is valid - SSH will use default
	return validateFilePaths(files, " ")
}

// createAliasValidator creates a validator for alias uniqueness
func createAliasValidator(originalAlias string, existingAliases []string) func(string) error {
	return func(alias string) error {
		// Skip duplicate check if no existing aliases provided
		if len(existingAliases) == 0 {
			return nil
		}

		// Skip check if alias hasn't changed (edit mode)
		if originalAlias != "" && alias == originalAlias {
			return nil
		}

		// Check for duplicates
		for _, existing := range existingAliases {
			if existing == alias {
				return fmt.Errorf("alias '%s' already exists", alias)
			}
		}

		return nil
	}
}

// IsIPAddress checks if the given string is a valid IP address (IPv4 or IPv6)
func IsIPAddress(host string) bool {
	return net.ParseIP(host) != nil
}

// validateHost validates a hostname or IP address
func validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host is required")
	}

	// Check for spaces
	if strings.Contains(host, " ") {
		return fmt.Errorf("host cannot contain spaces")
	}

	// Try to parse as IP address first
	if IsIPAddress(host) {
		return nil
	}

	// Validate as hostname
	return validateHostname(host)
}

// validateHostname validates a hostname (not IP)
func validateHostname(host string) error {
	if len(host) > 253 {
		return fmt.Errorf("hostname too long")
	}

	// Check for invalid characters using a single check
	if strings.ContainsAny(host, invalidHostChars) {
		return fmt.Errorf("host contains invalid characters")
	}

	// Check hostname format
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return fmt.Errorf("hostname cannot start or end with a dot")
	}

	if strings.Contains(host, "..") {
		return fmt.Errorf("hostname cannot contain consecutive dots")
	}

	// Validate each label
	return validateHostLabels(host)
}

// validateHostLabels validates each label in a hostname
func validateHostLabels(host string) error {
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if err := validateHostLabel(label); err != nil {
			return err
		}
	}
	return nil
}

// validateHostLabel validates a single hostname label
func validateHostLabel(label string) error {
	if label == "" {
		return fmt.Errorf("hostname has empty label")
	}
	if len(label) > 63 {
		return fmt.Errorf("hostname label too long")
	}
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return fmt.Errorf("hostname label cannot start or end with hyphen")
	}
	return nil
}

// validatePortForward validates port forwarding specification
func validatePortForward(forward string) error {
	if forward == "" {
		return nil // Port forwarding is optional
	}

	// Support multiple forwards separated by comma
	forwards := strings.Split(forward, ",")
	for _, fwd := range forwards {
		fwd = strings.TrimSpace(fwd)
		if fwd == "" {
			continue
		}

		// Format: [bind_address:]port:host:hostport
		parts := strings.Split(fwd, ":")
		if len(parts) < 3 || len(parts) > 4 {
			return fmt.Errorf("invalid format, expected [bind_address:]port:host:hostport")
		}

		// Validate ports
		var portIdx, hostPortIdx int
		if len(parts) == 3 {
			// port:host:hostport
			portIdx = 0
			hostPortIdx = 2
		} else {
			// bind_address:port:host:hostport
			portIdx = 1
			hostPortIdx = 3

			// Validate bind address
			if parts[0] != "" && parts[0] != "*" {
				if err := validateBindAddress(parts[0]); err != nil {
					return fmt.Errorf("invalid bind address: %w", err)
				}
			}
		}

		// Validate port numbers
		port, err := strconv.Atoi(parts[portIdx])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port number: %s", parts[portIdx])
		}

		hostPort, err := strconv.Atoi(parts[hostPortIdx])
		if err != nil || hostPort < 1 || hostPort > 65535 {
			return fmt.Errorf("invalid host port number: %s", parts[hostPortIdx])
		}
	}

	return nil
}

// validateDynamicForward validates dynamic port forwarding specification
func validateDynamicForward(forward string) error {
	if forward == "" {
		return nil // Dynamic forwarding is optional
	}

	// Support multiple forwards separated by comma
	forwards := strings.Split(forward, ",")
	for _, fwd := range forwards {
		fwd = strings.TrimSpace(fwd)
		if fwd == "" {
			continue
		}

		// Format: [bind_address:]port
		parts := strings.Split(fwd, ":")
		if len(parts) > 2 {
			return fmt.Errorf("invalid format, expected [bind_address:]port")
		}

		var portStr string
		if len(parts) == 1 {
			// Just port
			portStr = parts[0]
		} else {
			// bind_address:port
			if parts[0] != "" && parts[0] != "*" {
				if err := validateBindAddress(parts[0]); err != nil {
					return fmt.Errorf("invalid bind address: %w", err)
				}
			}
			portStr = parts[1]
		}

		// Validate port number
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port number: %s", portStr)
		}
	}

	return nil
}

// validateBindAddress validates a bind address (IP, hostname, or *)
func validateBindAddress(address string) error {
	if address == "" || address == "*" {
		return nil // Empty or wildcard is valid
	}

	// Check for spaces
	if strings.Contains(address, " ") {
		return fmt.Errorf("address cannot contain spaces")
	}

	// Try to parse as IP address first (including IPv6)
	if net.ParseIP(address) != nil {
		return nil
	}

	// Validate as hostname with relaxed rules
	return validateBindHostname(address)
}

// isNumericDottedFormat checks if the address looks like an IP address (contains only dots and digits)
func isNumericDottedFormat(address string) bool {
	for _, ch := range address {
		if ch != '.' && (ch < '0' || ch > '9') {
			return false
		}
	}
	return strings.Contains(address, ".")
}

// validateBindHostname validates a hostname for bind address (more permissive than regular hostname)
func validateBindHostname(address string) error {
	// Check for invalid characters using a single check
	if strings.ContainsAny(address, invalidAddressChars) {
		return fmt.Errorf("address contains invalid characters")
	}

	// Check hostname format
	if strings.HasPrefix(address, ".") || strings.HasSuffix(address, ".") {
		return fmt.Errorf("address cannot start or end with a dot")
	}

	if strings.HasPrefix(address, "-") || strings.HasSuffix(address, "-") {
		return fmt.Errorf("address cannot start or end with hyphen")
	}

	// Check for consecutive dots
	if strings.Contains(address, "..") {
		return fmt.Errorf("address cannot contain consecutive dots")
	}

	// If it looks like an IP address (contains only dots and digits), validate it more strictly
	if isNumericDottedFormat(address) {
		// Check if all segments are valid numbers
		segments := strings.Split(address, ".")
		// IPv4 should have exactly 4 segments
		if len(segments) == 4 {
			for _, seg := range segments {
				if seg == "" {
					return fmt.Errorf("invalid IP address format")
				}
				num, err := strconv.Atoi(seg)
				if err != nil || num < 0 || num > 255 {
					return fmt.Errorf("invalid IP address format")
				}
			}
			return nil // Valid IPv4
		}
		// If it's not 4 segments but looks numeric, it's invalid
		return fmt.Errorf("invalid address format")
	}

	// Check each label for hyphens at start/end
	if strings.Contains(address, ".") {
		return validateAddressLabels(address)
	}

	return nil
}

// validateAddressLabels validates labels in a bind address
func validateAddressLabels(address string) error {
	labels := strings.Split(address, ".")
	for _, label := range labels {
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("address label cannot start or end with hyphen")
		}
	}
	return nil
}

// stripColorTags removes tview color tags from a string
func stripColorTags(s string) string {
	// Remove all tview color tags like [red], [-], [yellow], etc.
	colorTagRegex := regexp.MustCompile(`\[[^\]]*\]`)
	return colorTagRegex.ReplaceAllString(s, "")
}
