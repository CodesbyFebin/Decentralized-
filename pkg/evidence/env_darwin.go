package evidence

import (
	"strconv"
	"syscall"
)

func platformEnv(e *Env) {
	e.CPUModel = run("sysctl", "-n", "machdep.cpu.brand_string")
	if n, err := strconv.ParseInt(run("sysctl", "-n", "hw.memsize"), 10, 64); err == nil {
		e.MemBytes = n
	}
	e.Distro = "macOS " + run("sw_vers", "-productVersion")
	var st syscall.Statfs_t
	if syscall.Statfs(e.DataDir, &st) == nil {
		e.DiskBytes = int64(st.Blocks) * int64(st.Bsize)
		e.DiskFreeBytes = int64(st.Bavail) * int64(st.Bsize)
		b := make([]byte, 0, len(st.Fstypename))
		for _, c := range st.Fstypename {
			if c == 0 {
				break
			}
			b = append(b, byte(c))
		}
		e.FSType = string(b)
	}
	if run("sysctl", "-n", "kern.hv_vmm_present") == "1" {
		e.Virtualization = "hypervisor present"
	}
}
