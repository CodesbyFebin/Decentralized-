package node

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"decentralized.host/pkg/api"
)

// hwProbe measures hardware from the OS. Root is prepended to every /proc
// and /sys path so tests can run the parsers against a recorded tree; run
// executes a command and returns its stdout ("" when it is missing or fails).
type hwProbe struct {
	Root string
	GOOS string
	Run  func(name string, args ...string) string
}

func systemProbe() hwProbe {
	return hwProbe{Root: "/", GOOS: goruntime.GOOS, Run: runOut}
}

func runOut(name string, args ...string) string {
	if _, err := exec.LookPath(name); err != nil {
		return ""
	}
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (p hwProbe) path(rel string) string { return filepath.Join(p.Root, rel) }

// measure fills the measured hardware fields of f. Anything it cannot
// establish is left empty and listed in f.Unknown; it never substitutes a
// configured or assumed value.
func (p hwProbe) measure(f *api.Facts, dataDir string) {
	var unknown []string
	miss := func(name string) { unknown = append(unknown, name) }

	switch p.GOOS {
	case "linux":
		info := p.keyValues("proc/meminfo", ":")
		if kb, ok := kiB(info["MemTotal"]); ok {
			f.MemBytes = kb
		} else {
			miss("memBytes")
		}
		if kb, ok := kiB(info["SwapTotal"]); ok {
			f.SwapBytes = kb
		} else {
			miss("swapBytes")
		}
		f.CPUModel, f.PhysicalCores = p.cpuinfo()
		if b, err := os.ReadFile(p.path("proc/uptime")); err == nil {
			if fs := strings.Fields(string(b)); len(fs) > 0 {
				if v, err := strconv.ParseFloat(fs[0], 64); err == nil {
					f.UptimeSec = int64(v)
				}
			}
		}
		disks, ok := p.disks()
		if ok {
			f.Disks = disks
		} else {
			miss("disks")
		}
		gpus, ok := p.gpus()
		if ok {
			f.GPUs = gpus
		} else {
			miss("gpus")
		}
	case "darwin":
		if n, err := strconv.ParseInt(p.Run("sysctl", "-n", "hw.memsize"), 10, 64); err == nil && n > 0 {
			f.MemBytes = n
		} else {
			miss("memBytes")
		}
		f.CPUModel = p.Run("sysctl", "-n", "machdep.cpu.brand_string")
		if n, err := strconv.ParseInt(p.Run("sysctl", "-n", "hw.physicalcpu"), 10, 64); err == nil {
			f.PhysicalCores = n
		}
		miss("swapBytes")
		miss("uptimeSec")
		miss("disks")
		miss("gpus")
	default:
		for _, n := range []string{"memBytes", "swapBytes", "uptimeSec", "disks", "gpus"} {
			miss(n)
		}
	}
	if f.CPUModel == "" {
		miss("cpuModel")
	}
	if f.PhysicalCores == 0 {
		miss("physicalCores")
	}
	if p.GOOS == "linux" && f.UptimeSec == 0 {
		miss("uptimeSec")
	}
	var st syscall.Statfs_t
	if dataDir != "" && syscall.Statfs(dataDir, &st) == nil {
		f.DataFS = &api.FSInfo{Path: dataDir, TotalBytes: int64(st.Blocks) * int64(st.Bsize), FreeBytes: int64(st.Bavail) * int64(st.Bsize)}
	} else {
		miss("dataFs")
	}
	// Reachability from the internet (NAT type, open ports) cannot be learned
	// from inside the host; it needs an outside observer, which dh/v1 does not
	// have yet.
	miss("natType")
	sort.Strings(unknown)
	f.Unknown = unknown
}

func (p hwProbe) keyValues(rel, sep string) map[string]string {
	out := map[string]string{}
	fh, err := os.Open(p.path(rel))
	if err != nil {
		return out
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), sep)
		if ok {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

// kiB parses a /proc/meminfo value such as "16318412 kB".
func kiB(v string) (int64, bool) {
	fs := strings.Fields(v)
	if len(fs) == 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(fs[0], 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	if len(fs) > 1 && strings.EqualFold(fs[1], "kB") {
		n *= 1024
	}
	return n, true
}

// cpuinfo returns the first model name and the number of distinct
// (physical id, core id) pairs. Architectures that report neither (many ARM
// kernels) return "" and 0, which the caller lists as unknown.
func (p hwProbe) cpuinfo() (string, int64) {
	fh, err := os.Open(p.path("proc/cpuinfo"))
	if err != nil {
		return "", 0
	}
	defer fh.Close()
	model := ""
	cores := map[string]bool{}
	phys, core := "", ""
	flush := func() {
		if core != "" {
			cores[phys+"/"+core] = true
		}
		phys, core = "", ""
	}
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "model name", "Model":
			if model == "" {
				model = v
			}
		case "physical id":
			phys = v
		case "core id":
			core = v
		}
	}
	flush()
	return model, int64(len(cores))
}

func (p hwProbe) readTrim(rel string) string {
	b, err := os.ReadFile(p.path(rel))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// disks lists block devices from /sys/block. ok is false when the directory
// cannot be read; an empty list with ok means none are visible.
func (p hwProbe) disks() ([]api.Disk, bool) {
	ents, err := os.ReadDir(p.path("sys/block"))
	if err != nil {
		return nil, false
	}
	out := []api.Disk{}
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		base := filepath.Join("sys/block", name)
		sectors, err := strconv.ParseInt(p.readTrim(filepath.Join(base, "size")), 10, 64)
		if err != nil {
			continue
		}
		out = append(out, api.Disk{
			Name:       name,
			SizeBytes:  sectors * 512, // /sys/block/*/size is always in 512-byte units
			Rotational: p.readTrim(filepath.Join(base, "queue/rotational")) == "1",
			Removable:  p.readTrim(filepath.Join(base, "removable")) == "1",
			Model:      p.readTrim(filepath.Join(base, "device/model")),
		})
	}
	return out, true
}

var pciVendors = map[string]string{"0x10de": "nvidia", "0x1002": "amd", "0x8086": "intel", "0x1af4": "virtio", "0x1234": "qemu", "0x15ad": "vmware"}

// gpus lists DRM devices from sysfs and, when nvidia-smi is installed,
// enriches NVIDIA entries with what it reports. ok is false only when the
// kernel has no DRM class at all, in which case GPUs are unknown rather than
// absent.
func (p hwProbe) gpus() ([]api.GPU, bool) {
	out := []api.GPU{}
	if smi := p.Run("nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader,nounits"); smi != "" {
		for _, line := range strings.Split(smi, "\n") {
			fs := strings.Split(line, ",")
			if len(fs) < 3 {
				continue
			}
			g := api.GPU{Vendor: "nvidia", Model: strings.TrimSpace(fs[0]), Driver: strings.TrimSpace(fs[2]), Source: "nvidia-smi"}
			if mib, err := strconv.ParseInt(strings.TrimSpace(fs[1]), 10, 64); err == nil {
				g.VRAMBytes = mib << 20
			}
			out = append(out, g)
		}
	}
	ents, err := os.ReadDir(p.path("sys/class/drm"))
	if err != nil {
		return out, len(out) > 0
	}
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, "card") || strings.Contains(name, "-") {
			continue // connectors such as card0-HDMI-A-1
		}
		dev := filepath.Join("sys/class/drm", name, "device")
		vendorID := p.readTrim(filepath.Join(dev, "vendor"))
		vendor := pciVendors[vendorID]
		if vendor == "" {
			vendor = vendorID
		}
		if vendor == "nvidia" && len(out) > 0 && out[0].Source == "nvidia-smi" {
			continue // already reported with more detail
		}
		g := api.GPU{Vendor: vendor, Source: "sysfs"}
		if l, err := os.Readlink(p.path(filepath.Join(dev, "driver"))); err == nil {
			g.Driver = filepath.Base(l)
		}
		if n, err := strconv.ParseInt(p.readTrim(filepath.Join(dev, "mem_info_vram_total")), 10, 64); err == nil {
			g.VRAMBytes = n
		}
		out = append(out, g)
	}
	return out, true
}
