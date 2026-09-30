package node

import (
	"context"
	"time"

	"decentralized.host/pkg/system"
)

// HardwareMonitor periodically probes local hardware and emits observations to heartbeat
type HardwareMonitor struct {
	prober    *system.Prober
	interval  time.Duration
	done      chan struct{}
	LastProbe *system.HardwareProfile
}

// NewHardwareMonitor creates a new hardware monitor
func NewHardwareMonitor(interval time.Duration) *HardwareMonitor {
	return &HardwareMonitor{
		prober:   system.NewProber(),
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start begins periodic hardware probing
func (hm *HardwareMonitor) Start(ctx context.Context) error {
	// Probe immediately on startup
	if err := hm.probe(ctx); err != nil {
		return err
	}

	// Then probe periodically
	ticker := time.NewTicker(hm.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(hm.done)
				return
			case <-hm.done:
				return
			case <-ticker.C:
				_ = hm.probe(ctx)
			}
		}
	}()

	return nil
}

// Stop halts the hardware monitor
func (hm *HardwareMonitor) Stop() {
	close(hm.done)
}

// probe detects current hardware state
func (hm *HardwareMonitor) probe(ctx context.Context) error {
	profile, err := hm.prober.Probe(ctx)
	if err != nil {
		return err
	}
	hm.LastProbe = profile
	return nil
}

// GetProfile returns the most recent hardware profile (cached)
func (hm *HardwareMonitor) GetProfile() *system.HardwareProfile {
	return hm.LastProbe
}
