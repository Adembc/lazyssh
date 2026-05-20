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
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	SetMasterKey("test-master-key-32-bytes-long!!")

	plaintext := "my-secret-password-123"
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == "" {
		t.Fatal("expected non-empty ciphertext")
	}
	if encrypted == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	SetMasterKey("test-master-key-32-bytes-long!!")

	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("encrypt empty failed: %v", err)
	}
	if encrypted != "" {
		t.Fatal("expected empty ciphertext for empty plaintext")
	}

	decrypted, err := Decrypt("")
	if err != nil {
		t.Fatalf("decrypt empty failed: %v", err)
	}
	if decrypted != "" {
		t.Fatal("expected empty plaintext for empty ciphertext")
	}
}

func TestEncryptWithoutKey(t *testing.T) {
	masterKey = nil

	_, err := Encrypt("test")
	if err == nil {
		t.Fatal("expected error when master key is not set")
	}
}

func TestDecryptWithoutKey(t *testing.T) {
	masterKey = nil

	_, err := Decrypt("test")
	if err == nil {
		t.Fatal("expected error when master key is not set")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	SetMasterKey("test-master-key-32-bytes-long!!")

	_, err := Decrypt("invalid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptShortCiphertext(t *testing.T) {
	SetMasterKey("test-master-key-32-bytes-long!!")

	// Valid base64 but too short to contain nonce
	_, err := Decrypt("YWJj")
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestIsEncryptionAvailable(t *testing.T) {
	masterKey = nil
	if IsEncryptionAvailable() {
		t.Fatal("expected encryption to be unavailable when key is nil")
	}

	SetMasterKey("test-key")
	if !IsEncryptionAvailable() {
		t.Fatal("expected encryption to be available when key is set")
	}
}

func TestNormalizeKey(t *testing.T) {
	// Short key should be padded
	shortKey := normalizeKey([]byte("short"))
	if len(shortKey) != 32 {
		t.Fatalf("expected key length 32, got %d", len(shortKey))
	}

	// Long key should be truncated
	longKey := normalizeKey([]byte("this-is-a-very-long-key-that-exceeds-thirty-two-characters-total"))
	if len(longKey) != 32 {
		t.Fatalf("expected key length 32, got %d", len(longKey))
	}

	// Exactly 32 bytes should remain unchanged
	exactKey := normalizeKey([]byte("exactly-32-bytes-long-key-1234"))
	if len(exactKey) != 32 {
		t.Fatalf("expected key length 32, got %d", len(exactKey))
	}
}
