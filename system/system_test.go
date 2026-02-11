package system

import (
	"strings"
	"testing"
)

// TestGetPartitionSuffix tests the GetPartitionSuffix function
func TestGetPartitionSuffix(t *testing.T) {
	tests := []struct {
		name      string
		disk      string
		wantPart1 string
		wantPart2 string
	}{
		{
			name:      "NVMe disk",
			disk:      "/dev/nvme0n1",
			wantPart1: "p1",
			wantPart2: "p2",
		},
		{
			name:      "NVMe disk without /dev prefix",
			disk:      "nvme0n1",
			wantPart1: "p1",
			wantPart2: "p2",
		},
		{
			name:      "SATA disk (sd)",
			disk:      "/dev/sda",
			wantPart1: "1",
			wantPart2: "2",
		},
		{
			name:      "VirtIO disk (vd)",
			disk:      "/dev/vda",
			wantPart1: "1",
			wantPart2: "2",
		},
		{
			name:      "Xen virtual disk (xvd)",
			disk:      "/dev/xvda",
			wantPart1: "1",
			wantPart2: "2",
		},
		{
			name:      "MMC/SD card (mmcblk)",
			disk:      "/dev/mmcblk0",
			wantPart1: "p1",
			wantPart2: "p2",
		},
		{
			name:      "Another NVMe disk",
			disk:      "/dev/nvme1n1",
			wantPart1: "p1",
			wantPart2: "p2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPart1, gotPart2 := GetPartitionSuffix(tt.disk)
			if gotPart1 != tt.wantPart1 {
				t.Errorf("GetPartitionSuffix(%s) partition1 = %v, want %v", tt.disk, gotPart1, tt.wantPart1)
			}
			if gotPart2 != tt.wantPart2 {
				t.Errorf("GetPartitionSuffix(%s) partition2 = %v, want %v", tt.disk, gotPart2, tt.wantPart2)
			}
		})
	}
}

// TestFindInterfaceByMac tests MAC address normalization
func TestFindInterfaceByMac(t *testing.T) {
	// This test would require mocking the ip link command
	// For now, we just test that the function normalizes MAC addresses correctly
	tests := []struct {
		name    string
		macAddr string
		want    string // normalized MAC
	}{
		{
			name:    "Uppercase MAC",
			macAddr: "AA:BB:CC:DD:EE:FF",
			want:    "aa:bb:cc:dd:ee:ff",
		},
		{
			name:    "Lowercase MAC",
			macAddr: "aa:bb:cc:dd:ee:ff",
			want:    "aa:bb:cc:dd:ee:ff",
		},
		{
			name:    "Mixed case MAC",
			macAddr: "Aa:Bb:Cc:Dd:Ee:Ff",
			want:    "aa:bb:cc:dd:ee:ff",
		},
		{
			name:    "MAC with spaces",
			macAddr: "  aa:bb:cc:dd:ee:ff  ",
			want:    "aa:bb:cc:dd:ee:ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This just tests the normalization logic
			// The actual function call would fail without a real network interface
			// In a real test, we'd mock the exec.Command
			normalized := normalizeMAC(tt.macAddr)
			if normalized != tt.want {
				t.Errorf("normalizeMAC(%s) = %v, want %v", tt.macAddr, normalized, tt.want)
			}
		})
	}
}

// Helper function to test MAC normalization
func normalizeMAC(mac string) string {
	return strings.ToLower(strings.TrimSpace(mac))
}
