package system

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

// DetectDisk finds the primary disk for installation
func DetectDisk() (string, error) {
	// First, scan /dev for all available block devices
	availableDisks, err := findAllBlockDevices()
	if err != nil {
		return "", fmt.Errorf("failed to scan for block devices: %v", err)
	}

	if len(availableDisks) == 0 {
		return "", fmt.Errorf("no suitable disk found")
	}

	// Priority order for disk types (higher priority = checked first)
	diskPriority := []string{
		"mmcblk", // MMC/SD card devices (e.g. Raspberry Pi, embedded systems)
		"nvme",   // NVMe drives
		"sd",     // SATA/SCSI drives
		"vd",     // VirtIO drives (VMs)
		"xvd",    // Xen virtual drives
	}

	// Select the best disk based on priority
	for _, priority := range diskPriority {
		for _, disk := range availableDisks {
			if strings.HasPrefix(disk, "/dev/"+priority) {
				return disk, nil
			}
		}
	}

	// If no priority match, return the first available disk
	return availableDisks[0], nil
}

// findAllBlockDevices scans /dev and returns a list of all valid block devices
func findAllBlockDevices() ([]string, error) {
	files, err := ioutil.ReadDir("/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to read /dev directory: %v", err)
	}

	var blockDevices []string

	// Look for block devices that match disk patterns (not partitions)
	diskRegex := regexp.MustCompile(`^(sd[a-z]|vd[a-z]|xvd[a-z]|nvme[0-9]+n[0-9]+|mmcblk[0-9]+)$`)

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
							blockDevices = append(blockDevices, diskPath)
						}
					}
				}
			}
		}
	}

	return blockDevices, nil
}

// DetectInterface finds the primary network interface using ip link
// Filters for UP interfaces and prefers ethernet over wifi
func DetectInterface() (string, error) {
	// Execute ip link show command
	cmd := exec.Command("ip", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute ip link: %v", err)
	}

	var ethernetInterfaces []string
	var otherInterfaces []string

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		// Look for interface lines (format: "2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> ...")
		if !strings.Contains(line, ": <") {
			continue
		}

		// Extract interface name
		parts := strings.Split(line, ": ")
		if len(parts) < 2 {
			continue
		}

		interfaceName := strings.TrimSpace(parts[1])

		// Skip loopback interface
		if interfaceName == "lo" {
			continue
		}

		// Check if interface is UP
		if !strings.Contains(line, "UP") {
			continue
		}

		// Categorize interface
		isWifi := strings.HasPrefix(interfaceName, "wlan") ||
			strings.HasPrefix(interfaceName, "wl")

		isEthernet := strings.HasPrefix(interfaceName, "eth") ||
			strings.HasPrefix(interfaceName, "en") ||
			strings.HasPrefix(interfaceName, "eno") ||
			strings.HasPrefix(interfaceName, "ens")

		if isEthernet {
			ethernetInterfaces = append(ethernetInterfaces, interfaceName)
		} else if !isWifi {
			// Add non-wifi, non-ethernet interfaces to other list
			otherInterfaces = append(otherInterfaces, interfaceName)
		} else {
			// Wifi interfaces go last
			otherInterfaces = append(otherInterfaces, interfaceName)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error parsing ip link output: %v", err)
	}

	// Prefer ethernet interfaces
	if len(ethernetInterfaces) > 0 {
		return ethernetInterfaces[0], nil
	}

	// Fall back to other UP interfaces
	if len(otherInterfaces) > 0 {
		return otherInterfaces[0], nil
	}

	return "", fmt.Errorf("no UP network interface found")
}

// GetInterfaceMac returns the MAC address for a given interface
func GetInterfaceMac(iface string) (string, error) {
	// Execute ip link show for specific interface
	cmd := exec.Command("ip", "link", "show", iface)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get MAC address for %s: %v", iface, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for line with "link/ether" or "link/loopback"
		if strings.HasPrefix(line, "link/ether ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("MAC address not found for interface %s", iface)
}

// FindInterfaceByMac finds an interface name by its MAC address
func FindInterfaceByMac(macAddr string) (string, error) {
	// Normalize MAC address to lowercase
	macAddr = strings.ToLower(strings.TrimSpace(macAddr))

	// Execute ip link show command
	cmd := exec.Command("ip", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute ip link: %v", err)
	}

	var currentInterface string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()

		// Interface line (format: "2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> ...")
		if strings.Contains(line, ": <") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				currentInterface = strings.TrimSpace(parts[1])
			}
			continue
		}

		// MAC address line (format: "    link/ether aa:bb:cc:dd:ee:ff ...")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "link/ether ") && currentInterface != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				foundMac := strings.ToLower(parts[1])
				if foundMac == macAddr {
					return currentInterface, nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error parsing ip link output: %v", err)
	}

	return "", fmt.Errorf("no interface found with MAC address %s", macAddr)
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

	// Execute ip link show for specific interface
	cmd := exec.Command("ip", "link", "show", iface)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("interface %s not found", iface)
	}

	return nil
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
// Supports interface name or MAC address as input
func ResolveInterface(providedInterface string) (string, error) {
	if providedInterface != "" {
		// Check if providedInterface is a MAC address (contains colons)
		if strings.Contains(providedInterface, ":") {
			// Try to find interface by MAC address
			iface, err := FindInterfaceByMac(providedInterface)
			if err != nil {
				return "", fmt.Errorf("invalid MAC address parameter: %v", err)
			}
			return iface, nil
		}

		// Validate provided interface name
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
// Returns disk, interface, and interface MAC address
func ResolveSystemInfo(providedDisk, providedInterface string) (string, string, string, error) {
	disk, err := ResolveDisk(providedDisk)
	if err != nil {
		return "", "", "", err
	}

	iface, err := ResolveInterface(providedInterface)
	if err != nil {
		return "", "", "", err
	}

	// Get MAC address for the interface
	mac, err := GetInterfaceMac(iface)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get MAC address: %v", err)
	}

	return disk, iface, mac, nil
}

// GetPartitionSuffix returns the partition suffix based on disk type
// For mmcblk, sd, vd, xvd disks: returns "1" and "2"
// For nvme disks: returns "p1" and "p2"
func GetPartitionSuffix(disk string) (string, string) {
	// Extract the base disk name from the path
	diskName := strings.TrimPrefix(disk, "/dev/")

	// Check if it's an nvme disk
	if strings.HasPrefix(diskName, "nvme") {
		return "p1", "p2"
	}

	// For mmcblk, sd, vd, xvd disks
	return "1", "2"
}
