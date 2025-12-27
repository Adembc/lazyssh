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

// SSHKey represents an SSH key that can be used for authentication.
type SSHKey struct {
	// Path is the absolute path to the private key file
	Path string

	// Name is the display name (typically filename without path)
	Name string

	// Comment is the key comment (from public key or ssh-add)
	Comment string

	// Type is the key algorithm (rsa, ed25519, ecdsa, etc.)
	Type string

	// Size is the key size in bits (e.g., 4096 for RSA)
	Size int

	// Fingerprint is the key fingerprint (e.g., "SHA256:5NUhY...")
	Fingerprint string

	// LoadedInAgent indicates if the key is currently loaded in ssh-agent
	LoadedInAgent bool

	// IsEncrypted indicates if the private key is encrypted with a passphrase
	IsEncrypted bool

	// HasPublicKey indicates if a corresponding .pub file exists
	HasPublicKey bool

	// ModTime is the last modification time of the private key file
	ModTime time.Time

	// Source indicates where this key was discovered ("config" or "agent")
	Source string

	// PublicKeyLine is the full public key line from ssh-add -L output.
	// Used for unloading keys from ssh-agent. Format: "ssh-rsa AAAAB3... comment"
	PublicKeyLine string
}
