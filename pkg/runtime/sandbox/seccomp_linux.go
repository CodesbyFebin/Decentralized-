//go:build linux

package sandbox

import (
	"errors"
	"fmt"
	goruntime "runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// denied are system calls a sandboxed workload never needs and that widen
// its reach into the kernel or the host: namespaces and mounts, tracing
// other processes, loading kernel code, kernel keyrings, clock and power
// control, device creation. They fail with EPERM.
var denied = append([]uintptr{
	unix.SYS_MOUNT, unix.SYS_UMOUNT2, unix.SYS_PIVOT_ROOT, unix.SYS_CHROOT,
	unix.SYS_UNSHARE, unix.SYS_SETNS,
	unix.SYS_FSOPEN, unix.SYS_FSCONFIG, unix.SYS_FSMOUNT, unix.SYS_MOVE_MOUNT, unix.SYS_OPEN_TREE, unix.SYS_MOUNT_SETATTR,
	unix.SYS_PTRACE, unix.SYS_PROCESS_VM_READV, unix.SYS_PROCESS_VM_WRITEV, unix.SYS_KCMP, unix.SYS_PIDFD_GETFD,
	unix.SYS_KEXEC_LOAD, unix.SYS_KEXEC_FILE_LOAD, unix.SYS_INIT_MODULE, unix.SYS_FINIT_MODULE, unix.SYS_DELETE_MODULE,
	unix.SYS_BPF, unix.SYS_PERF_EVENT_OPEN, unix.SYS_USERFAULTFD, unix.SYS_FANOTIFY_INIT,
	unix.SYS_KEYCTL, unix.SYS_ADD_KEY, unix.SYS_REQUEST_KEY,
	unix.SYS_REBOOT, unix.SYS_SWAPON, unix.SYS_SWAPOFF, unix.SYS_ACCT, unix.SYS_VHANGUP, unix.SYS_SYSLOG,
	unix.SYS_SETTIMEOFDAY, unix.SYS_CLOCK_SETTIME, unix.SYS_CLOCK_ADJTIME, unix.SYS_ADJTIMEX,
	unix.SYS_QUOTACTL, unix.SYS_OPEN_BY_HANDLE_AT, unix.SYS_NAME_TO_HANDLE_AT, unix.SYS_LOOKUP_DCOOKIE,
	unix.SYS_MKNODAT, unix.SYS_PERSONALITY,
}, archDenied...)

// nsFlags are clone(2) flags that create namespaces.
const nsFlags = unix.CLONE_NEWUSER | unix.CLONE_NEWNS | unix.CLONE_NEWPID | unix.CLONE_NEWNET |
	unix.CLONE_NEWIPC | unix.CLONE_NEWUTS | unix.CLONE_NEWCGROUP | unix.CLONE_NEWTIME

func stmt(code uint16, k uint32) unix.SockFilter { return unix.SockFilter{Code: code, K: k} }
func jump(code uint16, k uint32, jt, jf uint8) unix.SockFilter {
	return unix.SockFilter{Code: code, K: k, Jt: jt, Jf: jf}
}

// seccompProgram builds the filter:
//
//	wrong architecture          → kill the process
//	x32 ABI (amd64)             → EPERM
//	a denied syscall            → EPERM
//	clone with namespace flags  → EPERM
//	clone3                      → ENOSYS (arguments are in memory and cannot be
//	                              inspected; libc falls back to clone)
//	anything else               → allow
func seccompProgram() ([]unix.SockFilter, error) {
	const (
		offNr   = 0
		offArch = 4
		offArg0 = 16 // low 32 bits of args[0] (little endian)
	)
	eperm := uint32(unix.SECCOMP_RET_ERRNO) | uint32(unix.EPERM)
	enosys := uint32(unix.SECCOMP_RET_ERRNO) | uint32(unix.ENOSYS)
	allow := uint32(unix.SECCOMP_RET_ALLOW)
	kill := uint32(unix.SECCOMP_RET_KILL_PROCESS)

	var p []unix.SockFilter
	p = append(p,
		stmt(unix.BPF_LD|unix.BPF_W|unix.BPF_ABS, offArch),
		jump(unix.BPF_JMP|unix.BPF_JEQ|unix.BPF_K, auditArch, 1, 0),
		stmt(unix.BPF_RET|unix.BPF_K, kill),
		stmt(unix.BPF_LD|unix.BPF_W|unix.BPF_ABS, offNr),
	)
	if x32Bit != 0 {
		p = append(p,
			jump(unix.BPF_JMP|unix.BPF_JGE|unix.BPF_K, x32Bit, 0, 1),
			stmt(unix.BPF_RET|unix.BPF_K, eperm),
		)
	}
	for _, nr := range denied {
		p = append(p,
			jump(unix.BPF_JMP|unix.BPF_JEQ|unix.BPF_K, uint32(nr), 0, 1),
			stmt(unix.BPF_RET|unix.BPF_K, eperm),
		)
	}
	p = append(p,
		jump(unix.BPF_JMP|unix.BPF_JEQ|unix.BPF_K, uint32(unix.SYS_CLONE3), 0, 1),
		stmt(unix.BPF_RET|unix.BPF_K, enosys),
		// clone: refuse namespace creation.
		jump(unix.BPF_JMP|unix.BPF_JEQ|unix.BPF_K, uint32(unix.SYS_CLONE), 0, 3),
		stmt(unix.BPF_LD|unix.BPF_W|unix.BPF_ABS, offArg0),
		jump(unix.BPF_JMP|unix.BPF_JSET|unix.BPF_K, uint32(nsFlags), 0, 1),
		stmt(unix.BPF_RET|unix.BPF_K, eperm),
		stmt(unix.BPF_RET|unix.BPF_K, allow),
	)
	if len(p) > 4096 {
		return nil, errors.New("seccomp: program too long")
	}
	return p, nil
}

// applySeccomp installs the filter on the calling OS thread, which must be
// locked and must be the thread that calls execve: the filter is inherited
// across exec. no_new_privs is set first, as the kernel requires.
func applySeccomp() error {
	if goruntime.GOARCH != archName {
		return fmt.Errorf("seccomp: built for %s, running on %s", archName, goruntime.GOARCH)
	}
	prog, err := seccompProgram()
	if err != nil {
		return err
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("no_new_privs: %w", err)
	}
	fprog := unix.SockFprog{Len: uint16(len(prog)), Filter: &prog[0]}
	if _, _, e := unix.RawSyscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, 0, uintptr(unsafe.Pointer(&fprog))); e != 0 {
		return fmt.Errorf("seccomp: %w", e)
	}
	return nil
}
