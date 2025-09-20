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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"Valid IP", "192.168.1.1", false},
		{"Valid hostname", "example.com", false},
		{"Valid subdomain", "api.example.com", false},
		{"Empty host", "", true},
		{"Host with spaces", "example .com", true},
		{"Host with invalid chars", "example@com", true},
		{"Host starting with dot", ".example.com", true},
		{"Host ending with dot", "example.com.", true},
		{"Host with empty label", "example..com", true},
		{"Label starting with hyphen", "-example.com", true},
		{"Label ending with hyphen", "example-.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHost(%s) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePortForward(t *testing.T) {
	tests := []struct {
		name    string
		forward string
		wantErr bool
	}{
		{"Valid simple forward", "8080:localhost:80", false},
		{"Valid with bind address", "127.0.0.1:8080:localhost:80", false},
		{"Multiple forwards", "8080:localhost:80, 3000:localhost:3000", false},
		{"Empty forward", "", false},
		{"Invalid format - too few parts", "8080:localhost", true},
		{"Invalid format - too many parts", "127.0.0.1:8080:localhost:80:extra", true},
		{"Invalid port number", "abc:localhost:80", true},
		{"Port out of range", "70000:localhost:80", true},
		{"Invalid bind address - malformed IP", "127.0.0.0.0.0.1:8080:localhost:80", true},
		{"Invalid bind address - IP out of range", "192.168.1.256:8080:localhost:80", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePortForward(tt.forward)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePortForward(%s) error = %v, wantErr %v", tt.forward, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDynamicForward(t *testing.T) {
	tests := []struct {
		name    string
		forward string
		wantErr bool
	}{
		{"Valid port only", "1080", false},
		{"Valid with bind address", "127.0.0.1:1080", false},
		{"Multiple forwards", "1080, 1081", false},
		{"Empty forward", "", false},
		{"Invalid format - too many parts", "127.0.0.1:1080:extra", true},
		{"Invalid port number", "abc", true},
		{"Port out of range", "70000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDynamicForward(tt.forward)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDynamicForward(%s) error = %v, wantErr %v", tt.forward, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBindAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"Valid IP", "192.168.1.1", false},
		{"Valid IPv6", "::1", false},
		{"Valid hostname", "example.com", false},
		{"Wildcard", "*", false},
		{"Localhost", "localhost", false},
		{"Empty address", "", false},
		{"Address with spaces", "example .com", true},
		{"Address with invalid chars", "example@com", true},
		{"Address starting with dot", ".example.com", true},
		{"Address ending with dot", "example.com.", true},
		{"Address starting with hyphen", "-example.com", true},
		{"Address ending with hyphen", "example-.com", true},
		{"Invalid IP-like address", "127.0.0.0.0.0.1", true},
		{"Invalid numeric hostname", "192.168.1.256", true},
		{"Multiple dots", "example..com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBindAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBindAddress(%s) error = %v, wantErr %v", tt.address, err, tt.wantErr)
			}
		})
	}
}

func TestValidateKeyPaths(t *testing.T) {
	// Prepare an isolated HOME with a mock .ssh folder and key files
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})

	tempHome := t.TempDir()
	sshDir := filepath.Join(tempHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatalf("failed to create temp .ssh dir: %v", err)
	}

	shouldExistFiles := []string{"id_rsa", "id_ed25519"}
	for _, name := range shouldExistFiles {
		p := filepath.Join(sshDir, name)
		if err := os.WriteFile(p, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create mock key file %s: %v", p, err)
		}
	}
	if err := os.Setenv("HOME", tempHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	tests := []struct {
		name    string
		keys    string
		wantErr bool
	}{
		{"Valid single path", "~/.ssh/id_rsa", false},
		{"Valid multiple paths", "~/.ssh/id_rsa, ~/.ssh/id_ed25519", false},
		{"Empty keys", "", false},
		{"Path with newline", "~/.ssh/id_rsa\n", true},
		{"Path with tab", "~/.ssh/id_rsa\t", true},
		{"Path with carriage return", "~/.ssh/id_rsa\r", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyPaths(tt.keys)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateKeyPaths(%s) error = %v, wantErr %v", tt.keys, err, tt.wantErr)
			}
		})
	}
}

func TestFieldValidatorPatterns(t *testing.T) {
	fieldValidators := GetFieldValidators()

	tests := []struct {
		field   string
		value   string
		wantErr bool
	}{
		// Alias field
		{"Alias", "server-01", false},
		{"Alias", "server_01", false},
		{"Alias", "server.01", false},
		{"Alias", "server@01", true},
		{"Alias", "", true}, // Required field

		// Port field
		{"Port", "22", false},
		{"Port", "65535", false},
		{"Port", "0", true},
		{"Port", "65536", true},
		{"Port", "abc", true},

		// User field
		{"User", "root", false},
		{"User", "user_name", false},
		{"User", "user-name", false},
		{"User", "1user", true}, // Can't start with number

		// ConnectTimeout field
		{"ConnectTimeout", "none", false},
		{"ConnectTimeout", "30", false},
		{"ConnectTimeout", "0", true},
		{"ConnectTimeout", "-10", true},

		// IPQoS field
		{"IPQoS", "af21 cs1", false},
		{"IPQoS", "ef", false},
		{"IPQoS", "lowdelay", false},
		{"IPQoS", "invalid", true},

		// EscapeChar field
		{"EscapeChar", "~", false},
		{"EscapeChar", "none", false},
		{"EscapeChar", "^A", false},
		{"EscapeChar", "^z", false},
		{"EscapeChar", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.field+"_"+tt.value, func(t *testing.T) {
			validator, exists := fieldValidators[tt.field]
			if !exists {
				if tt.wantErr {
					t.Errorf("Expected validator for field %s but none found", tt.field)
				}
				return
			}

			var err error
			// Check required fields
			switch {
			case validator.Required && tt.value == "":
				err = &testError{msg: "required field is empty"}
			case validator.Pattern != nil && !validator.Pattern.MatchString(tt.value):
				err = &testError{msg: "pattern mismatch"}
			case validator.Validate != nil:
				err = validator.Validate(tt.value)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("validateField(%s, %s) error = %v, wantErr %v", tt.field, tt.value, err, tt.wantErr)
			}
		})
	}
}

// testError is a helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestValidationState_MultipleErrors(t *testing.T) {
	state := NewValidationState()

	// Set multiple errors
	state.SetError("Alias", "Alias is required")
	state.SetError("Host", "Host is required")
	state.SetError("Port", "Port must be between 1 and 65535")
	state.SetError("User", "Invalid username")

	// Check that we have errors
	if !state.HasErrors() {
		t.Error("Expected HasErrors to return true")
	}

	// Get all errors
	errors := state.GetAllErrors()

	// Should have 4 errors
	if len(errors) != 4 {
		t.Errorf("Expected 4 errors, got %d", len(errors))
	}

	// Print errors for debugging
	t.Logf("Found %d errors:", len(errors))
	for i, err := range errors {
		t.Logf("  %d. %s", i+1, err)
	}

	// Check that errors are in the expected order
	expectedOrder := []string{"Alias", "Host", "Port", "User"}
	for i, expectedField := range expectedOrder {
		if i >= len(errors) {
			break
		}
		// Check if the error message starts with the expected field name
		if len(errors[i]) < len(expectedField) || errors[i][:len(expectedField)] != expectedField {
			t.Errorf("Expected error %d to be for field %s, but got: %s", i, expectedField, errors[i])
		}
	}
}

func TestValidationState_Clear(t *testing.T) {
	state := NewValidationState()

	// Add some errors
	state.SetError("Alias", "Error 1")
	state.SetError("Host", "Error 2")

	// Clear all errors
	state.Clear()

	// Should have no errors
	if state.HasErrors() {
		t.Error("Expected no errors after Clear()")
	}

	if state.GetErrorCount() != 0 {
		t.Errorf("Expected error count to be 0, got %d", state.GetErrorCount())
	}
}

func TestAliasValidation(t *testing.T) {
	tests := []struct {
		name            string
		alias           string
		originalAlias   string
		existingAliases []string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "valid new alias",
			alias:           "newserver",
			originalAlias:   "",
			existingAliases: []string{"server1", "server2"},
			wantErr:         false,
		},
		{
			name:            "duplicate alias",
			alias:           "server1",
			originalAlias:   "",
			existingAliases: []string{"server1", "server2"},
			wantErr:         true,
			errContains:     "already exists",
		},
		{
			name:            "edit mode - same alias allowed",
			alias:           "server1",
			originalAlias:   "server1",
			existingAliases: []string{"server1", "server2"},
			wantErr:         false,
		},
		{
			name:            "edit mode - changed to duplicate",
			alias:           "server2",
			originalAlias:   "server1",
			existingAliases: []string{"server1", "server2"},
			wantErr:         true,
			errContains:     "already exists",
		},
		{
			name:            "no existing aliases",
			alias:           "anyserver",
			originalAlias:   "",
			existingAliases: nil,
			wantErr:         false,
		},
		{
			name:            "empty existing aliases",
			alias:           "anyserver",
			originalAlias:   "",
			existingAliases: []string{},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := GetFieldValidatorsWithContext(tt.originalAlias, tt.existingAliases)
			aliasValidator := validators["Alias"]

			var err error
			if aliasValidator.Validate != nil {
				err = aliasValidator.Validate(tt.alias)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("alias validation error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}
