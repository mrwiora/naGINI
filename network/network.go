package network

import (
	"fmt"
	"os/exec"
)

// RunDHCPCD starts DHCP client on the specified interface in the background
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
	return nil
}
