package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type SystemInfo struct {
	Disk      string
	Interface string
}

type Config struct {
	Disk       string
	Interface  string
	ScriptName string
	TOTPToken  string
	BaseURL    string
}

// detectDisk finds the primary disk for installation
func detectDisk() (string, error) {
	// Common disk patterns in order of preference
	diskPatterns := []string{
		"/dev/nvme0n1", // NVMe drives
		"/dev/sda",     // SATA/SCSI drives
		"/dev/vda",     // VirtIO drives (VMs)
		"/dev/xvda",    // Xen virtual drives
	}

	// Check if any of the common disks exist
	for _, disk := range diskPatterns {
		if _, err := os.Stat(disk); err == nil {
			return disk, nil
		}
	}

	// If none of the common ones exist, scan /dev for block devices
	files, err := ioutil.ReadDir("/dev")
	if err != nil {
		return "", fmt.Errorf("failed to read /dev directory: %v", err)
	}

	// Look for block devices that match disk patterns
	diskRegex := regexp.MustCompile(`^(sd[a-z]|vd[a-z]|xvd[a-z]|nvme[0-9]+n[0-9]+)$`)

	for _, file := range files {
		if diskRegex.MatchString(file.Name()) {
			diskPath := "/dev/" + file.Name()
			// Check if it's a block device
			if stat, err := os.Stat(diskPath); err == nil {
				if stat.Mode()&os.ModeDevice != 0 {
					// Additional check to ensure it's a block device
					if sysstat, ok := stat.Sys().(*syscall.Stat_t); ok {
						// Check if it's a block device (major number > 0)
						if (sysstat.Rdev>>8)&0xff > 0 {
							return diskPath, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable disk found")
}

// detectInterface finds the primary network interface
func detectInterface() (string, error) {
	// Read network interfaces from /proc/net/dev
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/net/dev: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	// Priority order for interface types
	var interfaces []string
	var ethernetInterfaces []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Extract interface name (before the colon)
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		interfaceName := strings.TrimSpace(parts[0])

		// Skip loopback interface
		if interfaceName == "lo" {
			continue
		}

		interfaces = append(interfaces, interfaceName)

		// Prefer ethernet interfaces (eth*, en*, eno*, ens*)
		if strings.HasPrefix(interfaceName, "eth") ||
			strings.HasPrefix(interfaceName, "en") ||
			strings.HasPrefix(interfaceName, "eno") ||
			strings.HasPrefix(interfaceName, "ens") {
			ethernetInterfaces = append(ethernetInterfaces, interfaceName)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /proc/net/dev: %v", err)
	}

	// Return first ethernet interface if available
	if len(ethernetInterfaces) > 0 {
		return ethernetInterfaces[0], nil
	}

	// Return first available interface
	if len(interfaces) > 0 {
		return interfaces[0], nil
	}

	return "", fmt.Errorf("no network interface found")
}

// confirmWithUser asks user to confirm the detected values
func confirmWithUser(info SystemInfo) bool {
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

// runDHCPCD starts DHCP client on the detected interface in the background
func runDHCPCD(iface string) error {
	fmt.Printf("Starting DHCP client on interface %s in background...\n", iface)

	// Stop any existing dhcpcd processes in background
	go func() {
		exec.Command("killall", "dhcpcd").Run()
	}()

	// Start dhcpcd on the interface in background
	cmd := exec.Command("dhcpcd", iface)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dhcpcd: %v", err)
	}

	fmt.Printf("DHCP client started in background (PID: %d)\n", cmd.Process.Pid)
	return nil
}

// downloadScript downloads the script and returns its content and SHA256 hash
func downloadScript(url string) ([]byte, string, error) {
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

// sha256ToBase32 converts a SHA256 hex string to base32 for TOTP apps
func sha256ToBase32(sha256Hex string) (string, error) {
	// Treat hex string as ASCII bytes (matching user's encoding approach)
	hexAsBytes := []byte(sha256Hex)

	// Encode to base32 (standard encoding with padding for most TOTP apps)
	base32Secret := base32.StdEncoding.EncodeToString(hexAsBytes)
	return base32Secret, nil
}

// generateTOTP generates a TOTP token for the given secret and timestamp
func generateTOTP(secret string, timestamp int64) (string, error) {
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

// verifyTOTP verifies a TOTP token against the secret, accepting current, previous, and next time windows
func verifyTOTP(secret, token string) bool {
	currentTime := time.Now().Unix()

	// Check current, previous (-30s), and next (+30s) time windows
	timeWindows := []int64{
		currentTime - 30, // Previous window
		currentTime,      // Current window
		currentTime + 30, // Next window
	}

	for _, timestamp := range timeWindows {
		expectedToken, err := generateTOTP(secret, timestamp)
		if err != nil {
			continue
		}
		if token == expectedToken {
			return true
		}
	}

	return false
}

// promptForTOTP prompts user for TOTP token and verifies it
func promptForTOTP(scriptHash string) error {
	// Convert SHA256 to base32 for user reference
	base32Secret, err := sha256ToBase32(scriptHash)
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
	if !verifyTOTP(scriptHash, token) {
		return fmt.Errorf("invalid TOTP token")
	}

	fmt.Println("✓ TOTP verification successful!")
	return nil
}

// executeScript executes the downloaded script content
func executeScript(scriptContent []byte) error {
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
		fmt.Sprintf("DISK=%s", os.Getenv("DISK")),
		fmt.Sprintf("INTERFACE=%s", os.Getenv("INTERFACE")),
		fmt.Sprintf("PASSWORD=%s", os.Getenv("PASSWORD")),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script execution failed: %v", err)
	}

	fmt.Println("Script executed successfully")
	return nil
}

// parseFlags parses command line arguments and returns configuration
func parseFlags() Config {
	var config Config

	flag.StringVar(&config.Disk, "disk", "", "Target disk for installation (e.g. /dev/sda)")
	flag.StringVar(&config.Interface, "interface", "", "Network interface to use (e.g. eth0)")
	flag.StringVar(&config.ScriptName, "script", "", "Script ID to download and execute")
	flag.StringVar(&config.TOTPToken, "totp", "", "TOTP token for verification")
	flag.StringVar(&config.BaseURL, "baseurl", "https://cdn.test.io", "Base URL for script downloads")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -disk /dev/sda -interface eth0 -script 1a2b3c4d -totp 123456\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -script 1a2b3c4d  # Interactive mode for other parameters\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  DEBUG=1     Show SHA256 hash and Base32 secret for TOTP setup\n")
	}

	flag.Parse()

	// Remove trailing slash from base URL if present
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")

	return config
}

// validateDisk checks if the specified disk exists and is valid
func validateDisk(disk string) error {
	if disk == "" {
		return fmt.Errorf("disk parameter is empty")
	}

	// Check if disk exists
	if _, err := os.Stat(disk); err != nil {
		return fmt.Errorf("disk %s does not exist: %v", disk, err)
	}

	// Check if it's a block device
	if stat, err := os.Stat(disk); err == nil {
		if stat.Mode()&os.ModeDevice != 0 {
			if sysstat, ok := stat.Sys().(*syscall.Stat_t); ok {
				if (sysstat.Rdev>>8)&0xff > 0 {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("disk %s is not a valid block device", disk)
}

// validateInterface checks if the specified network interface exists
func validateInterface(iface string) error {
	if iface == "" {
		return fmt.Errorf("interface parameter is empty")
	}

	// Check if interface exists in /proc/net/dev
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return fmt.Errorf("failed to open /proc/net/dev: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		interfaceName := strings.TrimSpace(parts[0])
		if interfaceName == iface {
			return nil
		}
	}

	return fmt.Errorf("interface %s not found", iface)
}

// getScriptID prompts user for the script ID
func getScriptID() (string, error) {
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

// getPassword uses script ID as password
func getPassword(scriptID string) string {
	// Use script ID as password
	password := scriptID
	fmt.Printf("Using script ID as password: %s\n", password)
	return password
}

func main() {
	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("This program must be run as root")
		os.Exit(1)
	}

	// Parse command line flags
	config := parseFlags()

	var info SystemInfo
	var scriptID string
	var autoMode bool

	// Check if all required parameters are provided for auto mode
	if config.Disk != "" && config.Interface != "" && config.ScriptName != "" && config.TOTPToken != "" {
		autoMode = true
		fmt.Println("=== Automatic Installation Mode ===")

		// Validate provided parameters
		if err := validateDisk(config.Disk); err != nil {
			fmt.Printf("Invalid disk parameter: %v\n", err)
			os.Exit(1)
		}

		if err := validateInterface(config.Interface); err != nil {
			fmt.Printf("Invalid interface parameter: %v\n", err)
			os.Exit(1)
		}

		info = SystemInfo{
			Disk:      config.Disk,
			Interface: config.Interface,
		}
		scriptID = config.ScriptName

		fmt.Printf("Using provided parameters:\n")
		fmt.Printf("  DISK: %s\n", info.Disk)
		fmt.Printf("  INTERFACE: %s\n", info.Interface)
		fmt.Printf("  SCRIPT: %s\n", scriptID)
		fmt.Printf("  Base URL: %s\n", config.BaseURL)
	} else {
		// Interactive mode
		fmt.Println("=== Interactive Installation Mode ===")

		// Use provided parameters or detect/prompt for missing ones
		var disk, iface string
		var err error

		if config.Disk != "" {
			if err := validateDisk(config.Disk); err != nil {
				fmt.Printf("Invalid disk parameter: %v\n", err)
				os.Exit(1)
			}
			disk = config.Disk
			fmt.Printf("Using provided disk: %s\n", disk)
		} else {
			fmt.Println("\nDetecting primary disk...")
			disk, err = detectDisk()
			if err != nil {
				fmt.Printf("Error detecting disk: %v\n", err)
				os.Exit(1)
			}
		}

		if config.Interface != "" {
			if err := validateInterface(config.Interface); err != nil {
				fmt.Printf("Invalid interface parameter: %v\n", err)
				os.Exit(1)
			}
			iface = config.Interface
			fmt.Printf("Using provided interface: %s\n", iface)
		} else {
			fmt.Println("Detecting network interface...")
			iface, err = detectInterface()
			if err != nil {
				fmt.Printf("Error detecting network interface: %v\n", err)
				os.Exit(1)
			}
		}

		info = SystemInfo{
			Disk:      disk,
			Interface: iface,
		}

		// Confirm with user unless all parameters are provided
		if !confirmWithUser(info) {
			fmt.Println("Setup cancelled by user")
			os.Exit(1)
		}
	}

	// Set environment variables
	os.Setenv("DISK", info.Disk)
	os.Setenv("INTERFACE", info.Interface)

	// Run dhcpcd
	if err := runDHCPCD(info.Interface); err != nil {
		fmt.Printf("Error starting DHCP: %v\n", err)
		os.Exit(1)
	}

	// Get script ID
	if config.ScriptName != "" {
		scriptID = config.ScriptName
	} else {
		var err error
		scriptID, err = getScriptID()
		if err != nil {
			fmt.Printf("Error getting script ID: %v\n", err)
			os.Exit(1)
		}
	}

	// Get password using script ID
	password := getPassword(scriptID)

	// Add password to environment variables
	os.Setenv("PASSWORD", password)

	// Construct script URL
	scriptURL := fmt.Sprintf("%s/%s", config.BaseURL, scriptID)
	fmt.Printf("\nScript URL: %s\n", scriptURL)

	// Download the script and get its hash
	scriptContent, scriptHash, err := downloadScript(scriptURL)
	if err != nil {
		fmt.Printf("Error downloading script: %v\n", err)
		os.Exit(1)
	}

	// Show all exported variables before script execution
	fmt.Printf("=== Configuration Summary ===\n")
	fmt.Printf("DISK=%s\n", info.Disk)
	fmt.Printf("INTERFACE=%s\n", info.Interface)
	fmt.Printf("PASSWORD=%s\n", password)
	fmt.Printf("Script SHA256: %s\n", scriptHash)

	// Handle TOTP verification
	if config.TOTPToken != "" {
		// Auto mode with provided TOTP token
		fmt.Printf("Verifying provided TOTP token...\n")
		if !verifyTOTP(scriptHash, config.TOTPToken) {
			fmt.Printf("Invalid TOTP token provided\n")
			os.Exit(1)
		}
		fmt.Println("✓ TOTP verification successful!")
	} else {
		// Interactive TOTP verification
		if err := promptForTOTP(scriptHash); err != nil {
			fmt.Printf("TOTP verification failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Final confirmation in interactive mode
	if !autoMode {
		fmt.Print("\nTOTP verified. Do you want to proceed with executing this script? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			os.Exit(1)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if !(input == "y" || input == "yes") {
			fmt.Println("Script execution cancelled by user")
			os.Exit(1)
		}
	} else {
		fmt.Println("\nAll parameters verified. Starting automatic installation...")
	}

	// Execute the script
	if err := executeScript(scriptContent); err != nil {
		fmt.Printf("Error executing script: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nArch Linux auto setup completed successfully!")
}
