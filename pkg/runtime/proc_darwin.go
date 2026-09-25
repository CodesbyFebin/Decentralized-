//go:build darwin

package runtime

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// ProcStartToken returns the kernel start time of pid, used to detect pid
// reuse when re-adopting a process after an agent restart.
func ProcStartToken(pid int) string {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || kp.Proc.P_pid != int32(pid) {
		return ""
	}
	st := kp.Proc.P_starttime
	return fmt.Sprintf("%d.%06d", st.Sec, st.Usec)
}
