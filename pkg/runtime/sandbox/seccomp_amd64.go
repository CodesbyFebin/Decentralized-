//go:build linux && amd64

package sandbox

import "golang.org/x/sys/unix"

const (
	archName  = "amd64"
	auditArch = uint32(unix.AUDIT_ARCH_X86_64)
	// x32Bit marks x32-ABI system call numbers, which would bypass a filter
	// keyed on x86-64 numbers.
	x32Bit = uint32(0x40000000)
)

var archDenied = []uintptr{unix.SYS_IOPL, unix.SYS_IOPERM, unix.SYS_USELIB, unix.SYS_MKNOD}
