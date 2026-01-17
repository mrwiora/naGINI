package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// SystemInfo holds detected system information
type SystemInfo struct {
	Disk         string
	Interface    string
	InterfaceMac string
}

// Config holds all configuration parameters
type Config struct {
	Disk        string
	Interface   string
	ScriptName  string
	Passphrase  string
	BaseURL     string
	ShowVersion bool
}

// ParseFlags parses command line arguments and returns configuration
func ParseFlags() Config {
	var config Config

	flag.StringVar(&config.Disk, "disk", "", "Target disk for installation (e.g. /dev/sda)")
	flag.StringVar(&config.Interface, "interface", "", "Network interface to use (e.g. eth0 or MAC address aa:bb:cc:dd:ee:ff)")
	flag.StringVar(&config.ScriptName, "script", "", "Script ID to download and execute")
	flag.StringVar(&config.Passphrase, "passphrase", "", "Passphrase for decrypting encrypted scripts")
	flag.StringVar(&config.BaseURL, "baseurl", "https://cdn.test.io", "Base URL for script downloads")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -disk /dev/sda -interface eth0 -script 1a2b3c4d\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -interface aa:bb:cc:dd:ee:ff -script 1234affe  # Using MAC address\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -script 1234affe -passphrase \"my-secret\"  # Decrypt encrypted script\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -script 1234affe  # Interactive mode for other parameters\n", os.Args[0])
	}

	flag.Parse()

	// Remove trailing slash from base URL if present
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")

	return config
}

// IsAutoMode returns true if all required parameters for automatic mode are provided
func (c Config) IsAutoMode() bool {
	return c.Disk != "" && c.Interface != "" && c.ScriptName != ""
}

// GetScriptURL constructs the full script URL
func (c Config) GetScriptURL() string {
	return fmt.Sprintf("%s/%s", c.BaseURL, c.ScriptName)
}

// GetScriptURLWithID constructs the full script URL with a specific script ID
func (c Config) GetScriptURLWithID(scriptID string) string {
	return fmt.Sprintf("%s/%s", c.BaseURL, scriptID)
}
