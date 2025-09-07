package main

import (
	"fmt"
	"os"

	"nagini/config"
	"nagini/network"
	"nagini/script"
	"nagini/security"
	"nagini/ui"
)

// VERSION is set via ldflags during build
var VERSION = "dev"

func main() {
	// Always show version at startup
	ui.ShowVersion(VERSION)

	// Parse command line flags first
	cfg := config.ParseFlags()

	// Show base URL being used
	ui.ShowBaseURL(cfg.BaseURL)

	// Handle version flag (just exit after showing version)
	if cfg.ShowVersion {
		os.Exit(0)
	}

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Printf("%sThis program must be run as root%s\n", ui.Red, ui.Reset)
		os.Exit(1)
	}

	var info config.SystemInfo
	var scriptID string
	var autoMode bool

	// Check if all required parameters are provided for auto mode
	if cfg.IsAutoMode() {
		autoMode = true
		ui.ShowAutoModeInfo(config.SystemInfo{}, cfg.ScriptName, cfg.BaseURL)

		// Resolve system info for auto mode
		var err error
		info, err = ui.ResolveSystemInfoAuto(cfg.Disk, cfg.Interface)
		if err != nil {
			fmt.Printf("%s%v%s\n", ui.Red, err, ui.Reset)
			os.Exit(1)
		}
		scriptID = cfg.ScriptName

		ui.ShowAutoModeInfo(info, scriptID, cfg.BaseURL)
	} else {
		// Interactive mode
		ui.ShowInteractiveModeInfo()

		// Resolve system info interactively
		var err error
		info, err = ui.ResolveSystemInfoInteractive(cfg.Disk, cfg.Interface)
		if err != nil {
			fmt.Printf("%s%v%s\n", ui.Red, err, ui.Reset)
			os.Exit(1)
		}

		// Confirm with user
		if !ui.ConfirmWithUser(info) {
			fmt.Printf("%sSetup cancelled by user%s\n", ui.Yellow, ui.Reset)
			os.Exit(1)
		}
	}

	// Set environment variables
	os.Setenv("DISK", info.Disk)
	os.Setenv("INTERFACE", info.Interface)

	// Network configuration section
	ui.ShowNetworkSection()

	// Run dhcpcd
	if err := network.RunDHCPCD(info.Interface); err != nil {
		fmt.Printf("%sError starting DHCP: %v%s\n", ui.Red, err, ui.Reset)
		os.Exit(1)
	}

	// Script configuration section
	ui.ShowScriptSection()

	// Get script ID
	if cfg.ScriptName != "" {
		scriptID = cfg.ScriptName
	} else {
		var err error
		scriptID, err = ui.GetScriptID()
		if err != nil {
			fmt.Printf("%sError getting script ID: %v%s\n", ui.Red, err, ui.Reset)
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
	fmt.Printf("\nScript URL: %s%s%s\n", ui.Cyan, scriptURL, ui.Reset)

	// Download the script and get its hash and metadata
	scriptContent, scriptHash, scriptMetadata, err := script.DownloadScript(scriptURL)
	if err != nil {
		fmt.Printf("%sError downloading script: %v%s\n", ui.Red, err, ui.Reset)
		os.Exit(1)
	}

	// Show script metadata to user
	ui.ShowScriptMetadata(ui.ScriptMetadata{
		NaGINIVersion:   scriptMetadata.NaGINIVersion,
		Author:          scriptMetadata.Author,
		TemplateVersion: scriptMetadata.TemplateVersion,
		Info:            scriptMetadata.Info,
	})

	// Show all exported variables before script execution
	ui.ShowConfigurationSummary(info, password, scriptHash)

	// Script verification section
	ui.ShowVerificationSection()

	// Handle STARTCODE verification
	if cfg.StartCode != "" {
		// Auto mode with provided STARTCODE token
		fmt.Printf("%sVerifying provided STARTCODE...%s\n", ui.Yellow, ui.Reset)
		if !security.VerifySTARTCODE(scriptHash, cfg.StartCode) {
			fmt.Printf("%sInvalid STARTCODE provided%s\n", ui.Red, ui.Reset)
			os.Exit(1)
		}
		fmt.Printf("%s✓ STARTCODE verification successful!%s\n", ui.Green, ui.Reset)
	} else {
		// Interactive STARTCODE verification
		if err := security.PromptForSTARTCODE(scriptHash, scriptContent); err != nil {
			fmt.Printf("%sSTARTCODE verification failed: %v%s\n", ui.Red, err, ui.Reset)
			os.Exit(1)
		}
	}

	// Final confirmation in interactive mode
	if !autoMode {
		if !ui.ConfirmExecution() {
			fmt.Printf("%sScript execution cancelled by user%s\n", ui.Yellow, ui.Reset)
			os.Exit(1)
		}
	} else {
		ui.ShowStartAutoInstallation()
	}

	// Script execution section
	ui.ShowExecutionSection()

	// Execute the script
	if err := script.ExecuteScript(scriptContent, info.Disk, info.Interface, password); err != nil {
		fmt.Printf("%sError executing script: %v%s\n", ui.Red, err, ui.Reset)
		os.Exit(1)
	}

	ui.ShowCompletion()
}
