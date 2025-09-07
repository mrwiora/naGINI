package system

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"syscall"
)

// DetectDisk finds the primary disk for installation
func DetectDisk() (string, error) {
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

// DetectInterface finds the primary network interface
func DetectInterface() (string, error) {
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

// ValidateDisk checks if the specified disk exists and is valid
func ValidateDisk(disk string) error {
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

// ValidateInterface checks if the specified network interface exists
func ValidateInterface(iface string) error {
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

// ResolveDisk handles disk parameter resolution (provided or detected) with validation
func ResolveDisk(providedDisk string) (string, error) {
	if providedDisk != "" {
		// Validate provided disk
		if err := ValidateDisk(providedDisk); err != nil {
			return "", fmt.Errorf("invalid disk parameter: %v", err)
		}
		return providedDisk, nil
	}

	// Auto-detect disk
	disk, err := DetectDisk()
	if err != nil {
		return "", fmt.Errorf("error detecting disk: %v", err)
	}

	return disk, nil
}

// ResolveInterface handles interface parameter resolution (provided or detected) with validation
func ResolveInterface(providedInterface string) (string, error) {
	if providedInterface != "" {
		// Validate provided interface
		if err := ValidateInterface(providedInterface); err != nil {
			return "", fmt.Errorf("invalid interface parameter: %v", err)
		}
		return providedInterface, nil
	}

	// Auto-detect interface
	iface, err := DetectInterface()
	if err != nil {
		return "", fmt.Errorf("error detecting network interface: %v", err)
	}

	return iface, nil
}

// ResolveSystemInfo resolves both disk and interface parameters with validation
func ResolveSystemInfo(providedDisk, providedInterface string) (string, string, error) {
	disk, err := ResolveDisk(providedDisk)
	if err != nil {
		return "", "", err
	}

	iface, err := ResolveInterface(providedInterface)
	if err != nil {
		return "", "", err
	}

	return disk, iface, nil
}
