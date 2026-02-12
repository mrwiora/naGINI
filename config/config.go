package config

import (
	"flag"
	"fmt"
	"net/url"
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
	ScriptURL   string
	Passphrase  string
	ShowVersion bool
}

// ParseFlags parses command line arguments and returns configuration
func ParseFlags() Config {
	var config Config

	flag.StringVar(&config.Disk, "disk", "", "Target disk for installation (e.g. /dev/sda)")
	flag.StringVar(&config.Interface, "interface", "", "Network interface to use (e.g. eth0 or MAC address aa:bb:cc:dd:ee:ff)")
	flag.StringVar(&config.ScriptURL, "url", "", "Full script URL (e.g. https://script.0x7e.eu/b343cbd0)")
	flag.StringVar(&config.Passphrase, "passphrase", "", "Passphrase for decrypting encrypted scripts")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -disk /dev/sda -interface eth0 -url https://script.0x7e.eu/1a2b3c4d\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -interface aa:bb:cc:dd:ee:ff -url https://script.0x7e.eu/1234affe  # Using MAC address\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -url https://script.0x7e.eu/1234affe -passphrase \"my-secret\"  # Decrypt encrypted script\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -url https://script.0x7e.eu/1234affe  # Interactive mode for other parameters\n", os.Args[0])
	}

	flag.Parse()

	// Remove trailing slash from script URL if present
	config.ScriptURL = strings.TrimSuffix(config.ScriptURL, "/")

	return config
}

// IsAutoMode returns true if all required parameters for automatic mode are provided
func (c Config) IsAutoMode() bool {
	return c.Disk != "" && c.Interface != "" && c.ScriptURL != ""
}

// GetScriptURL returns the full script URL
func (c Config) GetScriptURL() string {
	return c.ScriptURL
}

// GetScriptID extracts the script ID (last path segment) from the script URL
func (c Config) GetScriptID() string {
	if c.ScriptURL == "" {
		return ""
	}

	parsed, err := url.Parse(c.ScriptURL)
	if err != nil {
		// Fallback: treat the last segment after '/' as the script ID
		parts := strings.Split(c.ScriptURL, "/")
		return parts[len(parts)-1]
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// GetBaseURL extracts the base URL (everything before the script ID) from the script URL
func (c Config) GetBaseURL() string {
	if c.ScriptURL == "" {
		return ""
	}

	scriptID := c.GetScriptID()
	if scriptID == "" {
		return c.ScriptURL
	}

	return strings.TrimSuffix(c.ScriptURL, "/"+scriptID)
}
