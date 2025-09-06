package ui

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"nagini/config"
)

// ConfirmWithUser asks user to confirm the detected values
func ConfirmWithUser(info config.SystemInfo) bool {
	fmt.Printf("\n=== Arch Linux Auto Setup ===\n")
	fmt.Printf("Detected DISK: %s\n", info.Disk)
	fmt.Printf("Detected INTERFACE: %s\n", info.Interface)
	fmt.Print("\nAre these values correct? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// GetScriptID prompts user for the script ID
func GetScriptID() (string, error) {
	fmt.Print("Enter the script ID (e.g., 1a2b3c4d): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading script ID: %v", err)
	}

	scriptID := strings.TrimSpace(input)
	if scriptID == "" {
		return "", fmt.Errorf("script ID cannot be empty")
	}

	// Basic validation - alphanumeric characters only
	validID := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !validID.MatchString(scriptID) {
		return "", fmt.Errorf("script ID can only contain alphanumeric characters")
	}

	return scriptID, nil
}

// GetPassword uses script ID as password
func GetPassword(scriptID string) string {
	// Use script ID as password
	password := scriptID
	fmt.Printf("Using script ID as password: %s\n", password)
	return password
}

// ConfirmExecution asks user to confirm script execution
func ConfirmExecution() bool {
	fmt.Print("\nTOTP verified. Do you want to proceed with executing this script? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// ShowAutoModeInfo displays information for automatic mode
func ShowAutoModeInfo(info config.SystemInfo, scriptID, baseURL string) {
	fmt.Println("=== Automatic Installation Mode ===")
	fmt.Printf("Using provided parameters:\n")
	fmt.Printf("  DISK: %s\n", info.Disk)
	fmt.Printf("  INTERFACE: %s\n", info.Interface)
	fmt.Printf("  SCRIPT: %s\n", scriptID)
	fmt.Printf("  Base URL: %s\n", baseURL)
}

// ShowInteractiveModeInfo displays information for interactive mode
func ShowInteractiveModeInfo() {
	fmt.Println("=== Interactive Installation Mode ===")
}

// ShowConfigurationSummary displays all configuration before execution
func ShowConfigurationSummary(info config.SystemInfo, password, scriptHash string) {
	fmt.Printf("=== Configuration Summary ===\n")
	fmt.Printf("DISK=%s\n", info.Disk)
	fmt.Printf("INTERFACE=%s\n", info.Interface)
	fmt.Printf("PASSWORD=%s\n", password)
	fmt.Printf("Script SHA256: %s\n", scriptHash)
}

// ShowProvidedParameter displays when a parameter is provided via command line
func ShowProvidedParameter(paramType, value string) {
	fmt.Printf("Using provided %s: %s\n", paramType, value)
}

// ShowDetectionMessage displays detection messages
func ShowDetectionMessage(what string) {
	fmt.Printf("\nDetecting %s...\n", what)
}

// ShowStartAutoInstallation displays message when starting automatic installation
func ShowStartAutoInstallation() {
	fmt.Println("\nAll parameters verified. Starting automatic installation...")
}

// ShowCompletion displays final success message
func ShowCompletion() {
	fmt.Println("\nArch Linux auto setup completed successfully!")
}
