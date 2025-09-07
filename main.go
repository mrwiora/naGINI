package main

import (
	"fmt"
	"os"

	"nagini/config"
	"nagini/network"
	"nagini/script"
	"nagini/security"
	"nagini/system"
	"nagini/ui"
)

// VERSION is set via ldflags during build
var VERSION = "dev"

func main() {
	// Always show version at startup
	ui.ShowVersion(VERSION)

	// Parse command line flags first
	cfg := config.ParseFlags()

	// Handle version flag (just exit after showing version)
	if cfg.ShowVersion {
		os.Exit(0)
	}

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("This program must be run as root")
		os.Exit(1)
	}

	var info config.SystemInfo
	var scriptID string
	var autoMode bool

	// Check if all required parameters are provided for auto mode
	if cfg.IsAutoMode() {
		autoMode = true
		ui.ShowAutoModeInfo(config.SystemInfo{}, cfg.ScriptName, cfg.BaseURL)

		// Validate provided parameters
		if err := system.ValidateDisk(cfg.Disk); err != nil {
			fmt.Printf("Invalid disk parameter: %v\n", err)
			os.Exit(1)
		}

		if err := system.ValidateInterface(cfg.Interface); err != nil {
			fmt.Printf("Invalid interface parameter: %v\n", err)
			os.Exit(1)
		}

		info = config.SystemInfo{
			Disk:      cfg.Disk,
			Interface: cfg.Interface,
		}
		scriptID = cfg.ScriptName

		ui.ShowAutoModeInfo(info, scriptID, cfg.BaseURL)
	} else {
		// Interactive mode
		ui.ShowInteractiveModeInfo()

		// Use provided parameters or detect/prompt for missing ones
		var disk, iface string
		var err error

		if cfg.Disk != "" {
			if err := system.ValidateDisk(cfg.Disk); err != nil {
				fmt.Printf("Invalid disk parameter: %v\n", err)
				os.Exit(1)
			}
			disk = cfg.Disk
			ui.ShowProvidedParameter("disk", disk)
		} else {
			ui.ShowDetectionMessage("primary disk")
			disk, err = system.DetectDisk()
			if err != nil {
				fmt.Printf("Error detecting disk: %v\n", err)
				os.Exit(1)
			}
		}

		if cfg.Interface != "" {
			if err := system.ValidateInterface(cfg.Interface); err != nil {
				fmt.Printf("Invalid interface parameter: %v\n", err)
				os.Exit(1)
			}
			iface = cfg.Interface
			ui.ShowProvidedParameter("interface", iface)
		} else {
			ui.ShowDetectionMessage("network interface")
			iface, err = system.DetectInterface()
			if err != nil {
				fmt.Printf("Error detecting network interface: %v\n", err)
				os.Exit(1)
			}
		}

		info = config.SystemInfo{
			Disk:      disk,
			Interface: iface,
		}

		// Confirm with user unless all parameters are provided
		if !ui.ConfirmWithUser(info) {
			fmt.Println("Setup cancelled by user")
			os.Exit(1)
		}
	}

	// Set environment variables
	os.Setenv("DISK", info.Disk)
	os.Setenv("INTERFACE", info.Interface)

	// Run dhcpcd
	if err := network.RunDHCPCD(info.Interface); err != nil {
		fmt.Printf("Error starting DHCP: %v\n", err)
		os.Exit(1)
	}

	// Get script ID
	if cfg.ScriptName != "" {
		scriptID = cfg.ScriptName
	} else {
		var err error
		scriptID, err = ui.GetScriptID()
		if err != nil {
			fmt.Printf("Error getting script ID: %v\n", err)
			os.Exit(1)
		}
		// Update config with user-provided script ID
		cfg.ScriptName = scriptID
	}

	// Get password using script ID
	password := ui.GetPassword(scriptID)

	// Add password to environment variables
	os.Setenv("PASSWORD", password)

	// Construct script URL
	scriptURL := cfg.GetScriptURLWithID(scriptID)
	fmt.Printf("\nScript URL: %s\n", scriptURL)

	// Download the script and get its hash
	scriptContent, scriptHash, err := script.DownloadScript(scriptURL)
	if err != nil {
		fmt.Printf("Error downloading script: %v\n", err)
		os.Exit(1)
	}

	// Show all exported variables before script execution
	ui.ShowConfigurationSummary(info, password, scriptHash)

	// Handle TOTP verification
	if cfg.TOTPToken != "" {
		// Auto mode with provided TOTP token
		fmt.Printf("Verifying provided TOTP token...\n")
		if !security.VerifyTOTP(scriptHash, cfg.TOTPToken) {
			fmt.Printf("Invalid TOTP token provided\n")
			os.Exit(1)
		}
		fmt.Println("✓ TOTP verification successful!")
	} else {
		// Interactive TOTP verification
		if err := security.PromptForTOTP(scriptHash); err != nil {
			fmt.Printf("TOTP verification failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Final confirmation in interactive mode
	if !autoMode {
		if !ui.ConfirmExecution() {
			fmt.Println("Script execution cancelled by user")
			os.Exit(1)
		}
	} else {
		ui.ShowStartAutoInstallation()
	}

	// Execute the script
	if err := script.ExecuteScript(scriptContent, info.Disk, info.Interface, password); err != nil {
		fmt.Printf("Error executing script: %v\n", err)
		os.Exit(1)
	}

	ui.ShowCompletion()
}
