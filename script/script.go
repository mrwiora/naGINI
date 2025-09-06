package script

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// DownloadScript downloads the script and returns its content and SHA256 hash
func DownloadScript(url string) ([]byte, string, error) {
	fmt.Printf("Downloading script from %s...\n", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Download the script
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download script: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Read the script content
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read script content: %v", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(content)
	hashString := hex.EncodeToString(hash[:])

	fmt.Printf("Script downloaded successfully (%d bytes)\n", len(content))
	return content, hashString, nil
}

// ExecuteScript executes the downloaded script content
func ExecuteScript(scriptContent []byte, disk, iface, password string) error {
	fmt.Println("Executing installation script...")

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

	// Execute the script with bash
	cmd := exec.Command("bash", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set environment variables for the script
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DISK=%s", disk),
		fmt.Sprintf("INTERFACE=%s", iface),
		fmt.Sprintf("PASSWORD=%s", password),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script execution failed: %v", err)
	}

	fmt.Println("Script executed successfully")
	return nil
}
