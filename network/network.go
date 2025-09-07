package network

import (
	"fmt"
	"os/exec"
)

// RunDHCPCD starts DHCP client on the specified interface in the background
// and configures time synchronization
func RunDHCPCD(iface string) error {
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

	// Configure time synchronization
	fmt.Println("Configuring time synchronization...")

	// Enable NTP
	if err := exec.Command("timedatectl", "set-ntp", "true").Run(); err != nil {
		fmt.Printf("Warning: failed to enable NTP: %v\n", err)
	} else {
		fmt.Println("NTP enabled successfully")
	}

	// Restart systemd-timesyncd
	if err := exec.Command("systemctl", "restart", "systemd-timesyncd").Run(); err != nil {
		fmt.Printf("Warning: failed to restart systemd-timesyncd: %v\n", err)
	} else {
		fmt.Println("systemd-timesyncd restarted successfully")
	}

	return nil
}
