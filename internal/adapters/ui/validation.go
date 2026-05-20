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
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Adembc/lazyssh/internal/i18n"
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
	validators := make(map[string]fieldValidator)

	// Basic fields
	validators["Alias"] = fieldValidator{
		Required: true,
		Pattern:  regexp.MustCompile(`^[a-zA-Z0-9._-]+$`),
		Message:  i18n.T("validation.alias_format"),
	}
	validators["Host"] = fieldValidator{
		Required: true,
		Validate: validateHost,
		Message:  i18n.T("validation.host_format"),
	}
	validators["Port"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^([1-9]\d{0,4})$`),
		Validate: validatePort,
		Message:  i18n.T("validation.port_range"),
	}
	validators["User"] = fieldValidator{
		Pattern: regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`),
		Message: i18n.T("validation.user_format"),
	}
	validators["Keys"] = fieldValidator{
		Validate: validateKeyPaths,
		Message:  i18n.T("validation.keys_access"),
	}

	// Connection fields
	validators["ConnectTimeout"] = fieldValidator{
		Validate: validateConnectTimeout,
		Message:  i18n.T("validation.timeout_format"),
	}
	validators["ConnectionAttempts"] = fieldValidator{
		Pattern: regexp.MustCompile(`^[1-9]\d*$`),
		Message: i18n.T("validation.attempts_format"),
	}
	validators["ServerAliveInterval"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validateNonNegativeNumber,
		Message:  i18n.T("validation.alive_interval_format"),
	}
	validators["ServerAliveCountMax"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validateNonNegativeNumber,
		Message:  i18n.T("validation.alive_count_format"),
	}
	validators["IPQoS"] = fieldValidator{
		Validate: validateIPQoS,
		Message:  i18n.T("validation.ipqos_format"),
	}

	// Address and forwarding fields
	validators["BindAddress"] = fieldValidator{
		Validate: validateBindAddress,
		Message:  i18n.T("validation.bind_format"),
	}
	validators["LocalForward"] = fieldValidator{
		Validate: validatePortForward,
		Message:  i18n.T("validation.forward_format"),
	}
	validators["RemoteForward"] = fieldValidator{
		Validate: validatePortForward,
		Message:  i18n.T("validation.forward_format"),
	}
	validators["DynamicForward"] = fieldValidator{
		Validate: validateDynamicForward,
		Message:  i18n.T("validation.dynamic_forward_format"),
	}

	// Authentication fields
	validators["NumberOfPasswordPrompts"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validatePasswordPrompts,
		Message:  i18n.T("validation.password_prompts_range"),
	}

	// Advanced fields
	validators["CanonicalizeMaxDots"] = fieldValidator{
		Pattern:  regexp.MustCompile(`^\d+$`),
		Validate: validateNonNegativeNumber,
		Message:  i18n.T("validation.max_dots_format"),
	}
	validators["EscapeChar"] = fieldValidator{
		Validate: validateEscapeChar,
		Message:  i18n.T("validation.escape_char_format"),
	}

	// Security fields
	validators["UserKnownHostsFile"] = fieldValidator{
		Validate: validateKnownHostsFiles,
		Message:  i18n.T("validation.known_hosts_access"),
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
		return errors.New(i18n.T("validation.port_invalid"))
	}
	if port < 1 || port > 65535 {
		return errors.New(i18n.T("validation.port_range"))
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
		return errors.New(i18n.T("validation.timeout_invalid"))
	}
	if timeout <= 0 {
		return errors.New(i18n.T("validation.timeout_positive"))
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
		return errors.New(i18n.T("validation.invalid_number"))
	}
	if num < 0 {
		return errors.New(i18n.T("validation.non_negative"))
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
		return errors.New(i18n.T("validation.invalid_number"))
	}
	if num < 0 || num > 10 {
		return errors.New(i18n.T("validation.password_prompts_range"))
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
	return errors.New(i18n.T("validation.escape_char_format"))
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
		return errors.New(i18n.T("validation.ipqos_max"))
	}
	for _, part := range parts {
		if !validValues[strings.ToLower(part)] {
			return fmt.Errorf(i18n.T("validation.ipqos_value_invalid"), part)
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
		errors = append(errors, fmt.Sprintf(i18n.T("validation.files_not_found"), strings.Join(invalidPaths, ", ")))
	}
	if len(inaccessiblePaths) > 0 {
		errors = append(errors, fmt.Sprintf(i18n.T("validation.files_not_accessible"), strings.Join(inaccessiblePaths, ", ")))
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
		return errors.New(i18n.T("validation.file_invalid_chars"))
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

// validateHost validates a hostname or IP address
func validateHost(host string) error {
	if host == "" {
		return errors.New(i18n.T("validation.host_required"))
	}

	// Check for spaces
	if strings.Contains(host, " ") {
		return errors.New(i18n.T("validation.host_no_spaces"))
	}

	// Try to parse as IP address first
	if net.ParseIP(host) != nil {
		return nil
	}

	// Validate as hostname
	return validateHostname(host)
}

// validateHostname validates a hostname (not IP)
func validateHostname(host string) error {
	if len(host) > 253 {
		return errors.New(i18n.T("validation.hostname_too_long"))
	}

	// Check for invalid characters using a single check
	if strings.ContainsAny(host, invalidHostChars) {
		return errors.New(i18n.T("validation.host_invalid_chars"))
	}

	// Check hostname format
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return errors.New(i18n.T("validation.hostname_dots"))
	}

	if strings.Contains(host, "..") {
		return errors.New(i18n.T("validation.hostname_consecutive_dots"))
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
		return errors.New(i18n.T("validation.hostname_empty_label"))
	}
	if len(label) > 63 {
		return errors.New(i18n.T("validation.hostname_label_too_long"))
	}
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return errors.New(i18n.T("validation.hostname_label_hyphen"))
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
			return errors.New(i18n.T("validation.forward_format_detail"))
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
					return fmt.Errorf(i18n.T("validation.bind_address_invalid")+": %w", err)
				}
			}
		}

		// Validate port numbers
		port, err := strconv.Atoi(parts[portIdx])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf(i18n.T("validation.port_number_invalid"), parts[portIdx])
		}

		hostPort, err := strconv.Atoi(parts[hostPortIdx])
		if err != nil || hostPort < 1 || hostPort > 65535 {
			return fmt.Errorf(i18n.T("validation.host_port_invalid"), parts[hostPortIdx])
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
			return errors.New(i18n.T("validation.dynamic_forward_format_detail"))
		}

		var portStr string
		if len(parts) == 1 {
			// Just port
			portStr = parts[0]
		} else {
			// bind_address:port
			if parts[0] != "" && parts[0] != "*" {
				if err := validateBindAddress(parts[0]); err != nil {
					return fmt.Errorf(i18n.T("validation.bind_address_invalid")+": %w", err)
				}
			}
			portStr = parts[1]
		}

		// Validate port number
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf(i18n.T("validation.port_number_invalid"), portStr)
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
		return errors.New(i18n.T("validation.address_no_spaces"))
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
		return errors.New(i18n.T("validation.address_invalid_chars"))
	}

	// Check hostname format
	if strings.HasPrefix(address, ".") || strings.HasSuffix(address, ".") {
		return errors.New(i18n.T("validation.address_dots"))
	}

	if strings.HasPrefix(address, "-") || strings.HasSuffix(address, "-") {
		return errors.New(i18n.T("validation.address_hyphen"))
	}

	// Check for consecutive dots
	if strings.Contains(address, "..") {
		return errors.New(i18n.T("validation.address_consecutive_dots"))
	}

	// If it looks like an IP address (contains only dots and digits), validate it more strictly
	if isNumericDottedFormat(address) {
		// Check if all segments are valid numbers
		segments := strings.Split(address, ".")
		// IPv4 should have exactly 4 segments
		if len(segments) == 4 {
			for _, seg := range segments {
				if seg == "" {
					return errors.New(i18n.T("validation.ip_invalid"))
				}
				num, err := strconv.Atoi(seg)
				if err != nil || num < 0 || num > 255 {
					return errors.New(i18n.T("validation.ip_invalid"))
				}
			}
			return nil // Valid IPv4
		}
		// If it's not 4 segments but looks numeric, it's invalid
		return errors.New(i18n.T("validation.address_format"))
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
			return errors.New(i18n.T("validation.address_hyphen"))
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
