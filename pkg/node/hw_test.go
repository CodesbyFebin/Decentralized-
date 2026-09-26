package node

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"decentralized.host/pkg/api"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const cpuinfo2x2 = `processor	: 0
model name	: Test CPU @ 3.00GHz
physical id	: 0
core id		: 0

processor	: 1
model name	: Test CPU @ 3.00GHz
physical id	: 0
core id		: 0

processor	: 2
model name	: Test CPU @ 3.00GHz
physical id	: 0
core id		: 1

processor	: 3
model name	: Test CPU @ 3.00GHz
physical id	: 1
core id		: 0
`

func TestHardwareProbeMeasuresRecordedLinuxTree(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"proc/meminfo":                                   "MemTotal:       16318412 kB\nMemFree:  100 kB\nSwapTotal:       2097148 kB\n",
		"proc/cpuinfo":                                   cpuinfo2x2,
		"proc/uptime":                                    "3251.55 11414.62\n",
		"sys/block/loop0/size":                           "8\n",
		"sys/block/nvme0n1/size":                         "1953525168\n",
		"sys/block/nvme0n1/queue/rotational":             "0\n",
		"sys/block/nvme0n1/removable":                    "0\n",
		"sys/block/nvme0n1/device/model":                 "Example NVMe 1TB\n",
		"sys/block/sda/size":                             "7814037168\n",
		"sys/block/sda/queue/rotational":                 "1\n",
		"sys/block/sda/removable":                        "0\n",
		"sys/class/drm/card0/device/vendor":              "0x1002\n",
		"sys/class/drm/card0/device/mem_info_vram_total": "17163091968\n",
		"sys/class/drm/card0-DP-1/status":                "connected\n",
	})
	var f api.Facts
	hwProbe{Root: root, GOOS: "linux", Run: func(string, ...string) string { return "" }}.measure(&f, root)

	if f.MemBytes != 16318412*1024 || f.SwapBytes != 2097148*1024 {
		t.Fatalf("memory: %d swap: %d", f.MemBytes, f.SwapBytes)
	}
	if f.CPUModel != "Test CPU @ 3.00GHz" || f.PhysicalCores != 3 {
		t.Fatalf("cpu: %q cores %d", f.CPUModel, f.PhysicalCores)
	}
	if f.UptimeSec != 3251 {
		t.Fatalf("uptime %d", f.UptimeSec)
	}
	if len(f.Disks) != 2 || f.Disks[0].Name != "nvme0n1" || f.Disks[0].Rotational || f.Disks[0].SizeBytes != 1953525168*512 || f.Disks[0].Model != "Example NVMe 1TB" || !f.Disks[1].Rotational {
		t.Fatalf("disks: %+v", f.Disks)
	}
	if len(f.GPUs) != 1 || f.GPUs[0].Vendor != "amd" || f.GPUs[0].VRAMBytes != 17163091968 || f.GPUs[0].Model != "" || f.GPUs[0].Source != "sysfs" {
		t.Fatalf("gpus: %+v", f.GPUs)
	}
	if f.DataFS == nil || f.DataFS.TotalBytes <= 0 {
		t.Fatalf("data fs: %+v", f.DataFS)
	}
	if !slices.Equal(f.Unknown, []string{"natType"}) {
		t.Fatalf("unknown: %v", f.Unknown)
	}
}

func TestHardwareProbeReportsWhatItCannotMeasure(t *testing.T) {
	root := t.TempDir() // an empty tree: nothing can be measured
	var f api.Facts
	hwProbe{Root: root, GOOS: "linux", Run: func(string, ...string) string { return "" }}.measure(&f, filepath.Join(root, "missing"))
	if f.MemBytes != 0 || f.Disks != nil || f.GPUs != nil || f.DataFS != nil {
		t.Fatalf("invented values: %+v", f)
	}
	want := []string{"cpuModel", "dataFs", "disks", "gpus", "memBytes", "natType", "physicalCores", "swapBytes", "uptimeSec"}
	if !slices.Equal(f.Unknown, want) {
		t.Fatalf("unknown = %v, want %v", f.Unknown, want)
	}
}

func TestHardwareProbeNoGPUIsNotUnknown(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"sys/class/drm/version": "drm 1.1.0\n"})
	var f api.Facts
	hwProbe{Root: root, GOOS: "linux", Run: func(string, ...string) string { return "" }}.measure(&f, root)
	if f.GPUs == nil || len(f.GPUs) != 0 || slices.Contains(f.Unknown, "gpus") {
		t.Fatalf("a kernel with DRM and no cards has zero GPUs, measured: %+v unknown %v", f.GPUs, f.Unknown)
	}
}

func TestHardwareProbeUsesNvidiaSMI(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"sys/class/drm/card0/device/vendor": "0x10de\n"})
	run := func(name string, args ...string) string {
		if name == "nvidia-smi" {
			return "NVIDIA GeForce RTX 4090, 24564, 550.54.14"
		}
		return ""
	}
	var f api.Facts
	hwProbe{Root: root, GOOS: "linux", Run: run}.measure(&f, root)
	if len(f.GPUs) != 1 || f.GPUs[0].Model != "NVIDIA GeForce RTX 4090" || f.GPUs[0].VRAMBytes != 24564<<20 || f.GPUs[0].Driver != "550.54.14" || f.GPUs[0].Source != "nvidia-smi" {
		t.Fatalf("gpus: %+v", f.GPUs)
	}
}
