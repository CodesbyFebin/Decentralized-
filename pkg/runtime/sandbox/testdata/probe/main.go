// Command probe is a test workload for the sandbox. It reports what the
// kernel let it observe from inside a sandbox, so tests can assert the
// isolation boundary holds. It changes nothing on the host.
//
//	probe report            print one JSON document describing what is visible
//	probe mem MB            allocate and touch MB of memory (memory-limit test)
//	probe fork N            hold up to N sleeping children (pid-limit test)
//	probe serve             answer "ok" over HTTP on $PORT (relay test)
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

type report struct {
	UID           int      `json:"uid"`
	GID           int      `json:"gid"`
	PID1          bool     `json:"pid1"`          // are we PID 1 of our namespace
	ProcPIDs      int      `json:"procPids"`      // number of processes visible in /proc
	RootWritable  bool     `json:"rootWritable"`  // could we create /probe-root-write
	ShadowRead    bool     `json:"shadowRead"`    // could we read a host-style /etc/shadow
	ShadowExists  bool     `json:"shadowExists"`  // is /etc/shadow even present
	DevSda        bool     `json:"devSda"`        // is a block device /dev/sda present
	DevKmsg       bool     `json:"devKmsg"`       // is /dev/kmsg present
	CapEff        string   `json:"capEff"`        // effective capabilities (hex) from /proc/self/status
	NoNewPrivs    int      `json:"noNewPrivs"`    // prctl PR_GET_NO_NEW_PRIVS
	SeccompMode   int      `json:"seccompMode"`   // prctl PR_GET_SECCOMP
	MountAttempt  string   `json:"mountAttempt"`  // error from a mount attempt ("" = it succeeded)
	MknodAttempt  string   `json:"mknodAttempt"`  // error from creating a device node
	VisibleMounts int      `json:"visibleMounts"` // lines in /proc/self/mountinfo
	Env           []string `json:"env"`
	Hostname      string   `json:"hostname"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: probe <report|mem|fork|serve>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "report":
		doReport()
	case "mem":
		mb, _ := strconv.Atoi(arg(2, "64"))
		buf := make([]byte, mb<<20)
		for i := range buf {
			buf[i] = byte(i)
		}
		fmt.Println(len(buf))
		time.Sleep(time.Second)
	case "fork":
		n, _ := strconv.Atoi(arg(2, "100"))
		got := 0
		for i := 0; i < n; i++ {
			if _, _, e := unix.RawSyscall(unix.SYS_CLONE, uintptr(unix.SIGCHLD), 0, 0); e == 0 {
				got++
			} else {
				break
			}
		}
		fmt.Println("children", got)
		time.Sleep(500 * time.Millisecond)
	case "serve":
		http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "ok") })
		ln, err := net.Listen("tcp", "127.0.0.1:"+os.Getenv("PORT"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = http.Serve(ln, nil)
	default:
		os.Exit(2)
	}
}

func arg(i int, def string) string {
	if len(os.Args) > i {
		return os.Args[i]
	}
	return def
}

func doReport() {
	r := report{UID: os.Getuid(), GID: os.Getgid(), PID1: os.Getpid() == 1, Env: os.Environ()}
	r.Hostname, _ = os.Hostname()

	if ents, err := os.ReadDir("/proc"); err == nil {
		for _, e := range ents {
			if _, err := strconv.Atoi(e.Name()); err == nil {
				r.ProcPIDs++
			}
		}
	}
	if f, err := os.Create("/probe-root-write"); err == nil {
		r.RootWritable = true
		f.Close()
		os.Remove("/probe-root-write")
	}
	if _, err := os.Stat("/etc/shadow"); err == nil {
		r.ShadowExists = true
		if b, err := os.ReadFile("/etc/shadow"); err == nil && len(b) > 0 {
			r.ShadowRead = true
		}
	}
	_, err := os.Stat("/dev/sda")
	r.DevSda = err == nil
	_, err = os.Stat("/dev/kmsg")
	r.DevKmsg = err == nil

	r.CapEff = field("/proc/self/status", "CapEff:")
	r.NoNewPrivs, _ = unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
	r.SeccompMode, _ = unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0)

	if err := unix.Mount("tmpfs", "/tmp/probe-mnt", "tmpfs", 0, ""); err != nil {
		r.MountAttempt = err.Error()
	} else {
		_ = unix.Unmount("/tmp/probe-mnt", 0)
	}
	if err := unix.Mknod("/tmp/probe-dev", unix.S_IFBLK|0o600, int(unix.Mkdev(8, 0))); err != nil {
		r.MknodAttempt = err.Error()
	} else {
		os.Remove("/tmp/probe-dev")
	}
	if b, err := os.ReadFile("/proc/self/mountinfo"); err == nil {
		for _, line := range splitLines(b) {
			if line != "" {
				r.VisibleMounts++
			}
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(r)
}

func splitLines(b []byte) []string {
	var out []string
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, string(b[start:i]))
			start = i + 1
		}
	}
	return out
}

func field(path, prefix string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range splitLines(b) {
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			return trim(line[len(prefix):])
		}
	}
	return ""
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}
