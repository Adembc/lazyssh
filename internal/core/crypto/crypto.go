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

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
)

var masterKey []byte

const masterKeyFile = "lazyssh.key"

func init() {
	key := os.Getenv("LAZYSSH_MASTER_KEY")
	if key != "" {
		masterKey = normalizeKey([]byte(key))
		return
	}
	// Try to load from config directory
	configDir := getConfigDir()
	if configDir != "" {
		keyPath := filepath.Join(configDir, masterKeyFile)
		if data, err := os.ReadFile(keyPath); err == nil {
			masterKey = normalizeKey(data)
		}
	}
}

func getConfigDir() string {
	// Prefer XDG_CONFIG_HOME
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "lazyssh")
	}
	// Fall back to ~/.config/lazyssh
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "lazyssh")
	}
	return ""
}

func normalizeKey(key []byte) []byte {
	if len(key) < 32 {
		key = append(key, make([]byte, 32-len(key))...)
	}
	if len(key) > 32 {
		key = key[:32]
	}
	return key
}

// Encrypt encrypts plaintext using AES-GCM and returns base64-encoded ciphertext.
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if masterKey == nil {
		// Auto-generate and save master key
		if err := EnsureMasterKey(); err != nil {
			return "", err
		}
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if masterKey == nil {
		return "", errors.New("master key not set: set LAZYSSH_MASTER_KEY env var")
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// IsEncryptionAvailable returns true if the master key is set.
func IsEncryptionAvailable() bool {
	return masterKey != nil
}

// EnsureMasterKey generates and saves a master key if not already set.
// Returns error if key generation or saving fails.
func EnsureMasterKey() error {
	if masterKey != nil {
		return nil
	}

	// Generate random 32-byte key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}

	// Save to config directory
	configDir := getConfigDir()
	if configDir == "" {
		return errors.New("cannot determine config directory")
	}

	// Create directory if needed
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	keyPath := filepath.Join(configDir, masterKeyFile)
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return err
	}

	masterKey = key
	return nil
}

// SetMasterKey allows setting the master key programmatically (useful for testing).
func SetMasterKey(key string) {
	masterKey = normalizeKey([]byte(key))
}
