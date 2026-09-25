package evidence

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var fsMagic = map[int64]string{
	0xEF53: "ext4/ext3/ext2", 0x58465342: "xfs", 0x9123683E: "btrfs", 0x794c7630: "overlayfs",
	0x01021994: "tmpfs", 0x2FC12FC1: "zfs", 0x6a656a63: "fakeowner (Docker Desktop bind mount)", 0x65735546: "fuse",
	0x6969: "nfs", 0x517B: "smb", 0x786f4256: "virtiofs",
}

func platformEnv(e *Env) {
	e.CPUModel = readFirst("/proc/cpuinfo", "model name\t:")
	if kb := readFirst("/proc/meminfo", "MemTotal:"); kb != "" {
		n, _ := strconv.ParseInt(strings.Fields(kb)[0], 10, 64)
		e.MemBytes = n * 1024
	}
	e.Distro = strings.Trim(readFirst("/etc/os-release", "PRETTY_NAME="), `"`)
	var st syscall.Statfs_t
	if syscall.Statfs(e.DataDir, &st) == nil {
		e.DiskBytes = int64(st.Blocks) * int64(st.Bsize)
		e.DiskFreeBytes = int64(st.Bavail) * int64(st.Bsize)
		if name, ok := fsMagic[int64(st.Type)]; ok {
			e.FSType = name
		} else {
			e.FSType = fmt.Sprintf("magic 0x%x", st.Type)
		}
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		e.Container = "docker"
	} else if _, err := os.Stat("/run/.containerenv"); err == nil {
		e.Container = "podman"
	}
	if v := run("systemd-detect-virt"); v != "" && v != "none" {
		e.Virtualization = v
	} else if b, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		e.Virtualization = "dmi: " + strings.TrimSpace(string(b))
	} else if strings.Contains(e.Kernel, "linuxkit") {
		e.Virtualization = "linuxkit VM (Docker Desktop)"
	}
}
