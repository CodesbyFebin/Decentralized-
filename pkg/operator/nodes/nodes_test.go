package nodes

import (
	"testing"
	"time"
)

func TestRegisterNode(t *testing.T) {
	nr := NewNodeRegistry()

	hardware := HardwareProfile{
		CPUCores:    4,
		CPUSpeedGHz: 2.4,
		MemoryGB:    16,
		DiskGB:      500,
		DiskType:    "SSD",
		ValidatedAt: time.Now(),
	}

	network := NetworkProfile{
		IPv4Address: "192.168.1.100",
		Port:        8080,
		Bandwidth:   1000,
		Latency:     50 * time.Millisecond,
		PacketLoss:  0.1,
		ConnectionType: "public",
		TestedAt:    time.Now(),
	}

	location := GeographicLocation{
		Region:  "us-east",
		Country: "US",
		Latitude: 40.7128,
		Longitude: -74.0060,
		Timezone: "America/New_York",
	}

	node, err := nr.RegisterNode("dh1abcdefghijklmnopqrstuv", "op_001", hardware, network, location)
	if err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	if node.Status != HEALTHY {
		t.Errorf("expected HEALTHY status, got %s", node.Status)
	}
}

func TestRegisterNodeInvalidHardware(t *testing.T) {
	nr := NewNodeRegistry()

	// Invalid hardware (low CPU)
	hardware := HardwareProfile{
		CPUCores:    1,
		MemoryGB:    4,
		DiskGB:      100,
	}

	network := NetworkProfile{
		IPv4Address: "192.168.1.100",
		Port:        8080,
		Bandwidth:   1000,
		Latency:     50 * time.Millisecond,
		PacketLoss:  0.1,
	}

	location := GeographicLocation{
		Region:  "us-east",
		Country: "US",
	}

	_, err := nr.RegisterNode("dh1abcdefghijklmnopqrstuv", "op_001", hardware, network, location)
	if err == nil {
		t.Error("expected error for invalid hardware")
	}
}

func TestHealthCheck(t *testing.T) {
	nr := NewNodeRegistry()

	hardware := HardwareProfile{CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD", ValidatedAt: time.Now()}
	network := NetworkProfile{IPv4Address: "192.168.1.100", Port: 8080, Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1, TestedAt: time.Now()}
	location := GeographicLocation{Region: "us-east", Country: "US"}

	nr.RegisterNode("dh1abcdefghijklmnopqrstuv", "op_001", hardware, network, location)

	// Record health checks
	nr.RecordHealthCheck("dh1abcdefghijklmnopqrstuv", true, 45*time.Millisecond)
	nr.RecordHealthCheck("dh1abcdefghijklmnopqrstuv", true, 50*time.Millisecond)
	nr.RecordHealthCheck("dh1abcdefghijklmnopqrstuv", false, 1*time.Second)

	node, _ := nr.GetNode("dh1abcdefghijklmnopqrstuv")
	if node.SuccessfulChecks != 2 {
		t.Errorf("expected 2 successful checks, got %d", node.SuccessfulChecks)
	}

	if node.FailedChecks != 1 {
		t.Errorf("expected 1 failed check, got %d", node.FailedChecks)
	}

	if node.UptimePercentage != 66.66666666666666 {
		t.Errorf("expected uptime ~66.7%%, got %.1f%%", node.UptimePercentage)
	}
}

func TestOperatorNodeStatus(t *testing.T) {
	nr := NewNodeRegistry()

	hardware := HardwareProfile{CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD", ValidatedAt: time.Now()}
	network := NetworkProfile{IPv4Address: "192.168.1.100", Port: 8080, Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1, TestedAt: time.Now()}

	// Register 25 nodes (below minimum)
	for i := 0; i < 25; i++ {
		location := GeographicLocation{Region: "us-east", Country: "US"}
		nodeID := "dh1" + string(rune('a'+i)) + "bcdefghijklmnopqrstuv"
		nr.RegisterNode(nodeID, "op_001", hardware, network, location)
	}

	status := nr.GetOperatorNodeStatus("op_001")
	if status.IsReadyNodeCount {
		t.Error("expected not ready (25 < 50 nodes)")
	}

	// Register more nodes to reach minimum
	for i := 25; i < 50; i++ {
		location := GeographicLocation{Region: "us-west", Country: "US"}
		nodeID := "dh1" + string(rune('a'+(i%26))) + string(rune('a'+(i/26))) + "bcdefghijklmnopqrst"
		nr.RegisterNode(nodeID, "op_001", hardware, network, location)
	}

	status = nr.GetOperatorNodeStatus("op_001")
	if !status.IsReadyNodeCount {
		t.Error("expected ready with 50+ nodes")
	}
}

func TestGeographicDiversity(t *testing.T) {
	nr := NewNodeRegistry()

	hardware := HardwareProfile{CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD", ValidatedAt: time.Now()}
	network := NetworkProfile{IPv4Address: "192.168.1.100", Port: 8080, Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1, TestedAt: time.Now()}

	// Register nodes in different regions
	regions := []string{"us-east", "us-west", "eu-west"}

	for i, region := range regions {
		location := GeographicLocation{Region: region, Country: "XX"}
		nodeID := "dh1" + string(rune('a'+i)) + "bcdefghijklmnopqrstuv"
		nr.RegisterNode(nodeID, "op_001", hardware, network, location)
	}

	status := nr.GetOperatorNodeStatus("op_001")
	if status.RegionCount != 3 {
		t.Errorf("expected 3 regions, got %d", status.RegionCount)
	}
}

func TestListOperatorNodes(t *testing.T) {
	nr := NewNodeRegistry()

	hardware := HardwareProfile{CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD", ValidatedAt: time.Now()}
	network := NetworkProfile{IPv4Address: "192.168.1.100", Port: 8080, Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1, TestedAt: time.Now()}
	location := GeographicLocation{Region: "us-east", Country: "US"}

	count := 10
	for i := 0; i < count; i++ {
		nodeID := "dh1" + string(rune('a'+i)) + "bcdefghijklmnopqrstuv"
		nr.RegisterNode(nodeID, "op_001", hardware, network, location)
	}

	nodes := nr.OperatorNodes("op_001")
	if len(nodes) != count {
		t.Errorf("expected %d nodes, got %d", count, len(nodes))
	}
}

func TestConnectionQuality(t *testing.T) {
	tests := []struct {
		name     string
		latency  time.Duration
		loss     float64
		expected string
	}{
		{"excellent", 25 * time.Millisecond, 0.05, "EXCELLENT"},
		{"good", 75 * time.Millisecond, 0.2, "GOOD"},
		{"fair", 150 * time.Millisecond, 1.0, "FAIR"},
		{"poor", 300 * time.Millisecond, 3.0, "POOR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quality := assessConnectionQuality(tt.latency, tt.loss)
			if quality != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, quality)
			}
		})
	}
}

func TestSetTag(t *testing.T) {
	nr := NewNodeRegistry()

	hardware := HardwareProfile{CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD", ValidatedAt: time.Now()}
	network := NetworkProfile{IPv4Address: "192.168.1.100", Port: 8080, Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1, TestedAt: time.Now()}
	location := GeographicLocation{Region: "us-east", Country: "US"}

	nr.RegisterNode("dh1abcdefghijklmnopqrstuv", "op_001", hardware, network, location)

	err := nr.SetTag("dh1abcdefghijklmnopqrstuv", "env", "production")
	if err != nil {
		t.Fatalf("SetTag() error = %v", err)
	}

	node, _ := nr.GetNode("dh1abcdefghijklmnopqrstuv")
	if node.Tags["env"] != "production" {
		t.Error("expected tag to be set")
	}
}
