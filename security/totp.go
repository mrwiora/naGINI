package security

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// SHA256ToBase32 converts a SHA256 hex string to base32 for TOTP apps
func SHA256ToBase32(sha256Hex string) (string, error) {
	// Treat hex string as ASCII bytes (matching user's encoding approach)
	hexAsBytes := []byte(sha256Hex)

	// Encode to base32 (standard encoding with padding for most TOTP apps)
	base32Secret := base32.StdEncoding.EncodeToString(hexAsBytes)
	return base32Secret, nil
}

// GenerateTOTP generates a TOTP token for the given secret and timestamp
func GenerateTOTP(secret string, timestamp int64) (string, error) {
	// Use hex string as ASCII bytes directly (matching user's encoding approach)
	secretBytes := []byte(secret)

	// TOTP uses 30-second time windows
	timeCounter := timestamp / 30

	// Convert time counter to 8-byte big-endian
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(timeCounter))

	// Generate HMAC-SHA1
	mac := hmac.New(sha1.New, secretBytes)
	mac.Write(counterBytes)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[19] & 0xf
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Generate 6-digit code
	return fmt.Sprintf("%06d", code%1000000), nil
}

// VerifyTOTP verifies a TOTP token against the secret, accepting current, previous, and next time windows
func VerifyTOTP(secret, token string) bool {
	currentTime := time.Now().Unix()

	// Check current, previous (-30s), and next (+30s) time windows
	timeWindows := []int64{
		currentTime - 30, // Previous window
		currentTime,      // Current window
		currentTime + 30, // Next window
	}

	for _, timestamp := range timeWindows {
		expectedToken, err := GenerateTOTP(secret, timestamp)
		if err != nil {
			continue
		}
		if token == expectedToken {
			return true
		}
	}

	return false
}

// PromptForTOTP prompts user for TOTP token and verifies it
func PromptForTOTP(scriptHash string) error {
	// Convert SHA256 to base32 for user reference
	base32Secret, err := SHA256ToBase32(scriptHash)
	if err != nil {
		return fmt.Errorf("failed to generate base32 secret: %v", err)
	}

	// Debug output (only shown if DEBUG environment variable is set)
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("\n=== TOTP Debug Information ===\n")
		fmt.Printf("Script SHA256: %s\n", scriptHash)
		fmt.Printf("Base32 Secret: %s\n", base32Secret)
		fmt.Printf("Add this base32 secret to your TOTP authenticator app.\n")
		fmt.Printf("Generate a TOTP token using your authenticator app.\n")
		fmt.Printf("==============================\n")
	}

	fmt.Printf("\n=== TOTP Verification Required ===\n")
	fmt.Print("Enter TOTP token (6 digits): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading TOTP token: %v", err)
	}

	token := strings.TrimSpace(input)

	// Validate token format (6 digits)
	if !regexp.MustCompile(`^\d{6}$`).MatchString(token) {
		return fmt.Errorf("TOTP token must be exactly 6 digits")
	}

	// Verify TOTP
	if !VerifyTOTP(scriptHash, token) {
		return fmt.Errorf("invalid TOTP token")
	}

	fmt.Println("✓ TOTP verification successful!")
	return nil
}
