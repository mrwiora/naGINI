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

	"nagini/ui"
)

// SHA256ToBase32 converts a SHA256 hex string to base32 for STARTCODE generation
func SHA256ToBase32(sha256Hex string) (string, error) {
	// Treat hex string as ASCII bytes (matching user's encoding approach)
	hexAsBytes := []byte(sha256Hex)

	// Encode to base32 (standard encoding with padding for most authenticator apps)
	base32Secret := base32.StdEncoding.EncodeToString(hexAsBytes)
	return base32Secret, nil
}

// GenerateSTARTCODE generates a STARTCODE token for the given secret and timestamp
func GenerateSTARTCODE(secret string, timestamp int64) (string, error) {
	// Use hex string as ASCII bytes directly (matching user's encoding approach)
	secretBytes := []byte(secret)

	// STARTCODE uses 30-second time windows
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

// VerifySTARTCODE verifies a STARTCODE token against the secret, accepting current, previous, and next time windows
func VerifySTARTCODE(secret, token string) bool {
	currentTime := time.Now().Unix()

	// Check current, previous (-30s), and next (+30s) time windows
	timeWindows := []int64{
		currentTime - 30, // Previous window
		currentTime,      // Current window
		currentTime + 30, // Next window
	}

	for _, timestamp := range timeWindows {
		expectedToken, err := GenerateSTARTCODE(secret, timestamp)
		if err != nil {
			continue
		}
		if token == expectedToken {
			return true
		}
	}

	return false
}

// DisplayScriptContent displays the script content to the user
func DisplayScriptContent(scriptContent []byte) {
	fmt.Printf("\n%s=== Script Content ===%s\n", ui.Red, ui.Reset)
	fmt.Printf("%s%s%s", ui.Red, string(scriptContent), ui.Reset)
	fmt.Printf("\n%s=== End of Script ===%s\n\n", ui.Red, ui.Reset)
}

// PromptForSTARTCODE prompts user for STARTCODE token and verifies it
func PromptForSTARTCODE(scriptHash string, scriptContent []byte) error {
	// Convert SHA256 to base32 for user reference
	base32Secret, err := SHA256ToBase32(scriptHash)
	if err != nil {
		return fmt.Errorf("failed to generate base32 secret: %v", err)
	}

	// Debug output (only shown if DEBUG environment variable is set)
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("\n%s=== STARTCODE Debug Information ===%s\n", ui.Blue, ui.Reset)
		fmt.Printf("Script SHA256: %s%s%s\n", ui.Cyan, scriptHash, ui.Reset)
		fmt.Printf("Base32 Secret: %s%s%s\n", ui.Cyan, base32Secret, ui.Reset)
		fmt.Printf("Add this base32 secret to your authenticator app.\n")
		fmt.Printf("Generate a STARTCODE using your authenticator app.\n")
		fmt.Printf("%s========================================%s\n", ui.Blue, ui.Reset)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s=== STARTCODE Verification Required ===%s\n", ui.Yellow, ui.Reset)
		fmt.Printf("The script content is verified using SHA256: %s%s%s\n", ui.Cyan, scriptHash, ui.Reset)
		fmt.Print("Enter STARTCODE (6 digits) or press ENTER to view script content: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading STARTCODE: %v", err)
		}

		token := strings.TrimSpace(input)

		// If user just pressed enter, show script content and continue loop
		if token == "" {
			DisplayScriptContent(scriptContent)
			continue
		}

		// Validate token format (6 digits)
		if !regexp.MustCompile(`^\d{6}$`).MatchString(token) {
			fmt.Printf("%sSTARTCODE must be exactly 6 digits. Please try again.%s\n", ui.Red, ui.Reset)
			continue
		}

		// Verify STARTCODE
		if !VerifySTARTCODE(scriptHash, token) {
			fmt.Printf("%sInvalid STARTCODE. Please try again.%s\n", ui.Red, ui.Reset)
			continue
		}

		fmt.Printf("%s✓ STARTCODE verification successful!%s\n", ui.Green, ui.Reset)
		return nil
	}
}
