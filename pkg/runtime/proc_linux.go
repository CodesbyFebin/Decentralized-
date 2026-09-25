//go:build linux

package runtime

import (
	"fmt"
	"os"
	"strings"
)

// ProcStartToken returns field 22 (starttime) of /proc/<pid>/stat.
func ProcStartToken(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return ""
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return ""
	}
	fields := strings.Fields(s[i+1:])
	if len(fields) < 20 {
		return ""
	}
	return fields[19]
}
