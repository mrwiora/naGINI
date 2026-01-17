package script

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"nagini/crypto"
	"nagini/system"
	"nagini/ui"
)

// ScriptMetadata holds script metadata information
type ScriptMetadata struct {
	NaGINIVersion   string
	Author          string
	TemplateVersion string
	Info            string
}

// ParseMetadata extracts metadata from script content
func ParseMetadata(scriptContent []byte) ScriptMetadata {
	metadata := ScriptMetadata{}
	scanner := bufio.NewScanner(strings.NewReader(string(scriptContent)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip non-metadata lines
		if !strings.HasPrefix(line, "#@") {
			continue
		}

		// Remove "#@" prefix and split by ":"
		metaLine := strings.TrimSpace(line[2:])
		parts := strings.SplitN(metaLine, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "naGINI":
			metadata.NaGINIVersion = value
		case "Author":
			metadata.Author = value
		case "Template-Version":
			metadata.TemplateVersion = value
		case "Info":
			metadata.Info = value
		}
	}

	return metadata
}

// DecryptIfNeeded checks if content is encrypted and decrypts it if passphrase is provided
func DecryptIfNeeded(content []byte, passphrase string) ([]byte, bool, error) {
	// Check if content appears to be encrypted (base64 encoded with proper length)
	if !crypto.IsEncrypted(content) {
		// Not encrypted, return as-is
		return content, false, nil
	}

	// Content appears to be encrypted
	if passphrase == "" {
		return nil, true, fmt.Errorf("script appears to be encrypted but no passphrase provided (use -passphrase flag)")
	}

	// Attempt to decrypt
	fmt.Printf("%sDetected encrypted script, attempting to decrypt...%s\n", ui.Yellow, ui.Reset)
	decrypted, err := crypto.Decrypt(string(content), passphrase)
	if err != nil {
		return nil, true, fmt.Errorf("failed to decrypt script: %v", err)
	}

	fmt.Printf("%sScript decrypted successfully%s\n", ui.Green, ui.Reset)
	return decrypted, true, nil
}

// DownloadScript downloads the script and returns its content, SHA256 hash, and metadata
func DownloadScript(url string) ([]byte, string, ScriptMetadata, error) {
	fmt.Printf("%sDownloading script from %s...%s\n", ui.Yellow, url, ui.Reset)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Download the script
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", ScriptMetadata{}, fmt.Errorf("failed to download script: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", ScriptMetadata{}, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Read the script content
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", ScriptMetadata{}, fmt.Errorf("failed to read script content: %v", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(content)
	hashString := hex.EncodeToString(hash[:])

	// Parse metadata from script content
	metadata := ParseMetadata(content)

	fmt.Printf("%sScript downloaded successfully (%d bytes)%s\n", ui.Green, len(content), ui.Reset)
	return content, hashString, metadata, nil
}

// ExecuteScript executes the downloaded script content
func ExecuteScript(scriptContent []byte, disk, iface, interfaceMac, password string) error {
	fmt.Printf("%sExecuting installation script...%s\n", ui.Yellow, ui.Reset)

	// Create a temporary file for the script
	tmpFile, err := ioutil.TempFile("", "arch-install-*.sh")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write script content to temporary file
	if _, err := tmpFile.Write(scriptContent); err != nil {
		return fmt.Errorf("failed to write script to temporary file: %v", err)
	}

	// Make the script executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %v", err)
	}

	// Close the file before executing
	tmpFile.Close()

	// Execute the script with bash -e (exit on first error)
	cmd := exec.Command("bash", "-e", tmpFile.Name())

	// Capture stderr while still showing stdout in real-time
	var stderrBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Stdin = os.Stdin

	// Get partition suffixes based on disk type
	partition1, partition2 := system.GetPartitionSuffix(disk)

	// Set environment variables for the script
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DISK=%s", disk),
		fmt.Sprintf("INTERFACE=%s", iface),
		fmt.Sprintf("INTERFACEMAC=%s", interfaceMac),
		fmt.Sprintf("PASSWORD=%s", password),
		fmt.Sprintf("PARTITION1=%s", partition1),
		fmt.Sprintf("PARTITION2=%s", partition2),
	)

	if err := cmd.Run(); err != nil {
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("script execution failed: %v\nError output:\n%s", err, stderrBuf.String())
		}
		return fmt.Errorf("script execution failed: %v", err)
	}

	fmt.Printf("%sScript executed successfully%s\n", ui.Green, ui.Reset)
	return nil
}
