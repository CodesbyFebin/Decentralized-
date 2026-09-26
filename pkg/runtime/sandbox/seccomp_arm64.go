//go:build linux && arm64

package sandbox

import "golang.org/x/sys/unix"

const (
	archName  = "arm64"
	auditArch = uint32(unix.AUDIT_ARCH_AARCH64)
	x32Bit    = uint32(0)
)

var archDenied = []uintptr{}
