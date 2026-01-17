package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltLength = 16  // 16 bytes for salt
	ivLength   = 12  // 12 bytes for GCM IV
	iterations = 100000 // PBKDF2 iterations
	keyLength  = 32  // 256 bits for AES-256
)

// Decrypt decrypts base64-encoded AES-256-GCM encrypted content using the provided passphrase
// The encrypted data format is: [16 bytes: Salt][12 bytes: IV][Remaining: Encrypted Data + Auth Tag]
func Decrypt(encryptedBase64, passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase cannot be empty")
	}

	// Decode from base64
	combined, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %v", err)
	}

	// Verify minimum length (salt + iv + at least some encrypted data)
	minLength := saltLength + ivLength + 16 // 16 for auth tag
	if len(combined) < minLength {
		return nil, fmt.Errorf("encrypted data too short (expected at least %d bytes, got %d)", minLength, len(combined))
	}

	// Extract components
	salt := combined[0:saltLength]
	iv := combined[saltLength : saltLength+ivLength]
	encryptedData := combined[saltLength+ivLength:]

	// Derive key using PBKDF2 with SHA256
	key := pbkdf2.Key([]byte(passphrase), salt, iterations, keyLength, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Decrypt and verify
	decrypted, err := gcm.Open(nil, iv, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (invalid passphrase or corrupted data): %v", err)
	}

	return decrypted, nil
}

// IsEncrypted attempts to detect if content is base64-encoded encrypted data
// This is a heuristic check - it verifies if the content is valid base64 and has the expected minimum length
func IsEncrypted(content []byte) bool {
	// Check if content looks like base64
	contentStr := string(content)

	// Try to decode as base64
	decoded, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		return false
	}

	// Check if decoded data has expected minimum length for encrypted format
	minLength := saltLength + ivLength + 16 // salt + iv + auth tag
	return len(decoded) >= minLength
}
