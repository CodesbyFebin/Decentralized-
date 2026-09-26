//go:build linux

package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// MaybeInit must be the first thing main (and TestMain) does. In the
// re-executed sandbox init it prepares the sandbox and execs the workload;
// it never returns there. Anywhere else it returns immediately.
func MaybeInit() {
	if len(os.Args) == 0 || os.Args[0] != initArg0 {
		return
	}
	// Capability, seccomp and no_new_privs state is per thread; everything
	// happens on this one, which also calls execve.
	goruntime.LockOSThread()
	errPipe := os.NewFile(5, "err")
	if err := runInit(); err != nil {
		fmt.Fprintf(errPipe, "%v", err)
		os.Exit(1)
	}
}

func runInit() error {
	var ic initConfig
	cfg := os.NewFile(3, "cfg")
	if err := json.NewDecoder(cfg).Decode(&ic); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	cfg.Close()
	// Wait until the parent has put us into the cgroups.
	syncf := os.NewFile(4, "sync")
	if _, err := syncf.Read(make([]byte, 1)); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	syncf.Close()

	if err := buildRoot(&ic); err != nil {
		return err
	}
	if err := unix.Sethostname([]byte("sandbox")); err != nil {
		return fmt.Errorf("hostname: %w", err)
	}
	if ic.NewNet {
		if err := loopbackUp(); err != nil {
			return fmt.Errorf("loopback: %w", err)
		}
	}
	for _, r := range []struct {
		res int
		val uint64
	}{{unix.RLIMIT_CORE, 0}, {unix.RLIMIT_NOFILE, 4096}} {
		if err := unix.Setrlimit(r.res, &unix.Rlimit{Cur: r.val, Max: r.val}); err != nil {
			return fmt.Errorf("rlimit: %w", err)
		}
	}
	if err := dropPrivileges(ic.RunUID); err != nil {
		return err
	}
	if err := unix.Chdir(WorkDir); err != nil {
		return err
	}
	// Nothing but stdio crosses into the workload; the error pipe closes on
	// exec, which tells the parent the workload started.
	closeFrom(3)
	if err := applySeccomp(); err != nil {
		return err
	}
	argv := append([]string{filepath.Join(AppDir, filepath.Base(ic.Executable))}, ic.Args...)
	return fmt.Errorf("exec: %w", unix.Exec(argv[0], argv, ic.Env))
}

func closeFrom(fd int) {
	ents, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return
	}
	for _, e := range ents {
		n, err := strconv.Atoi(e.Name())
		if err == nil && n >= fd {
			unix.CloseOnExec(n)
		}
	}
}

// mountFlags are the per-mount flags an unprivileged user namespace may not
// clear on a bind mount; a read-only remount must repeat them.
func lockedFlags(path string) uintptr {
	var st unix.Statfs_t
	if unix.Statfs(path, &st) != nil {
		return 0
	}
	var f uintptr
	for _, m := range []struct{ st, ms int64 }{
		{unix.ST_NOSUID, unix.MS_NOSUID}, {unix.ST_NODEV, unix.MS_NODEV}, {unix.ST_NOEXEC, unix.MS_NOEXEC},
		{unix.ST_NOATIME, unix.MS_NOATIME}, {unix.ST_NODIRATIME, unix.MS_NODIRATIME}, {unix.ST_RELATIME, unix.MS_RELATIME},
	} {
		if st.Flags&m.st != 0 {
			f |= uintptr(m.ms)
		}
	}
	return f
}

func bind(src, dst string, readOnly bool, extra uintptr) error {
	if err := unix.Mount(src, dst, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("bind %s: %w", dst, err)
	}
	flags := uintptr(unix.MS_REMOUNT|unix.MS_BIND|unix.MS_NOSUID|unix.MS_NODEV) | lockedFlags(dst) | extra
	if readOnly {
		flags |= unix.MS_RDONLY
	}
	if err := unix.Mount("", dst, "", flags, ""); err != nil {
		return fmt.Errorf("remount %s: %w", dst, err)
	}
	return nil
}

// systemDirs are shared read-only so dynamically linked programs run. /etc
// is not among them: the sandbox gets its own minimal /etc.
var systemDirs = []string{"/usr", "/lib", "/lib64", "/lib32", "/bin", "/sbin"}

var devices = []string{"null", "zero", "full", "random", "urandom"}

func buildRoot(ic *initConfig) error {
	r := ic.Root
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}
	if err := unix.Mount("tmpfs", r, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "mode=0755,size=8m"); err != nil {
		return fmt.Errorf("root tmpfs: %w", err)
	}
	mk := func(p string, mode os.FileMode) error { return os.MkdirAll(filepath.Join(r, p), mode) }
	for _, d := range systemDirs {
		fi, err := os.Lstat(d)
		switch {
		case err != nil:
			continue
		case fi.Mode()&os.ModeSymlink != 0: // merged /usr: /bin -> usr/bin
			target, err := os.Readlink(d)
			if err != nil {
				return err
			}
			if err := os.Symlink(target, filepath.Join(r, d)); err != nil {
				return err
			}
		case fi.IsDir():
			if err := mk(d, 0o755); err != nil {
				return err
			}
			if err := bind(d, filepath.Join(r, d), true, 0); err != nil {
				return err
			}
		}
	}
	// The artifact, read-only, executable.
	if err := mk(AppDir, 0o755); err != nil {
		return err
	}
	app := filepath.Join(r, AppDir, filepath.Base(ic.Executable))
	if err := os.WriteFile(app, nil, 0o755); err != nil {
		return err
	}
	// Bound by real host path: binds happen before pivot_root, while host
	// paths are still visible, and these paths (the verified artifact and the
	// agent's managed volume directories) are ones the agent owns.
	if err := bind(ic.Executable, app, true, 0); err != nil {
		return err
	}
	for _, m := range ic.Mounts {
		if err := mk(m.Path, 0o755); err != nil {
			return err
		}
		if err := bind(m.HostPath, filepath.Join(r, m.Path), m.ReadOnly, 0); err != nil {
			return err
		}
	}
	// /dev: a tmpfs with a handful of harmless device nodes.
	if err := mk("dev", 0o755); err != nil {
		return err
	}
	if err := unix.Mount("tmpfs", filepath.Join(r, "dev"), "tmpfs", unix.MS_NOSUID|unix.MS_NOEXEC, "mode=0755,size=64k"); err != nil {
		return fmt.Errorf("/dev: %w", err)
	}
	for _, d := range devices {
		p := filepath.Join(r, "dev", d)
		if err := os.WriteFile(p, nil, 0o666); err != nil {
			return err
		}
		if err := bind("/dev/"+d, p, false, 0); err != nil {
			return err
		}
	}
	if err := mk("dev/shm", 0o1777); err != nil {
		return err
	}
	if err := unix.Mount("tmpfs", filepath.Join(r, "dev/shm"), "tmpfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, "mode=1777,size=16m"); err != nil {
		return fmt.Errorf("/dev/shm: %w", err)
	}
	for _, l := range [][2]string{{"/proc/self/fd", "dev/fd"}, {"/proc/self/fd/0", "dev/stdin"}, {"/proc/self/fd/1", "dev/stdout"}, {"/proc/self/fd/2", "dev/stderr"}} {
		if err := os.Symlink(l[0], filepath.Join(r, l[1])); err != nil {
			return err
		}
	}
	// /proc of the sandbox's own PID namespace.
	if err := mk("proc", 0o555); err != nil {
		return err
	}
	if err := unix.Mount("proc", filepath.Join(r, "proc"), "proc", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("/proc: %w", err)
	}
	// /tmp, the only writable place besides volumes.
	if err := mk("tmp", 0o1777); err != nil {
		return err
	}
	if err := unix.Mount("tmpfs", filepath.Join(r, "tmp"), "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, fmt.Sprintf("mode=1777,size=%d", ic.Effective.TmpBytes)); err != nil {
		return fmt.Errorf("/tmp: %w", err)
	}
	// A minimal /etc (no shadow, no host configuration).
	if err := mk("etc", 0o755); err != nil {
		return err
	}
	etc := map[string]string{
		"passwd":        "root:x:0:0:root:/tmp:/sbin/nologin\napp:x:1000:1000:app:/tmp:/sbin/nologin\n",
		"group":         "root:x:0:\napp:x:1000:\n",
		"hosts":         "127.0.0.1 localhost sandbox\n::1 localhost\n",
		"nsswitch.conf": "hosts: files dns\n",
	}
	if !ic.NewNet {
		if b, err := os.ReadFile("/etc/resolv.conf"); err == nil {
			etc["resolv.conf"] = string(b)
		}
	}
	for name, body := range etc {
		if err := os.WriteFile(filepath.Join(r, "etc", name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	if fi, err := os.Stat("/etc/ssl"); err == nil && fi.IsDir() {
		if err := mk("etc/ssl", 0o755); err != nil {
			return err
		}
		if err := bind("/etc/ssl", filepath.Join(r, "etc/ssl"), true, 0); err != nil {
			return err
		}
	}

	// Switch to the new root and drop every host mount.
	if err := mk(".old", 0o700); err != nil {
		return err
	}
	if err := unix.Chdir(r); err != nil {
		return err
	}
	if err := unix.PivotRoot(".", ".old"); err != nil {
		return fmt.Errorf("pivot_root: %w", err)
	}
	if err := unix.Chdir("/"); err != nil {
		return err
	}
	if err := unix.Unmount("/.old", unix.MNT_DETACH); err != nil {
		return fmt.Errorf("detach host root: %w", err)
	}
	if err := os.Remove("/.old"); err != nil {
		return err
	}
	if err := unix.Mount("", "/", "", unix.MS_REMOUNT|unix.MS_RDONLY|unix.MS_NOSUID|unix.MS_NODEV, ""); err != nil {
		return fmt.Errorf("read-only root: %w", err)
	}
	return nil
}

func loopbackUp() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	ifr, err := unix.NewIfreq("lo")
	if err != nil {
		return err
	}
	if err := unix.IoctlIfreq(fd, unix.SIOCGIFFLAGS, ifr); err != nil {
		return err
	}
	ifr.SetUint16(ifr.Uint16() | unix.IFF_UP)
	return unix.IoctlIfreq(fd, unix.SIOCSIFFLAGS, ifr)
}

// dropPrivileges empties the capability bounding and ambient sets, becomes
// runUID:runUID, and then zeroes every capability set. When runUID is 0
// (namespace-root, because no subordinate uid range is available) the uid
// change keeps capabilities, so the explicit capset is what leaves the
// workload with none; with an empty bounding set and no_new_privs no later
// execve can regain one.
func dropPrivileges(runUID int) error {
	last := 40
	if b, err := os.ReadFile("/proc/sys/kernel/cap_last_cap"); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
			last = n
		}
	}
	for c := 0; c <= last; c++ {
		if err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(c), 0, 0, 0); err != nil {
			return fmt.Errorf("drop bounding capability %d: %w", c, err)
		}
		_ = unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0)
	}
	if err := unix.Setresgid(runUID, runUID, runUID); err != nil {
		return fmt.Errorf("setresgid: %w", err)
	}
	if err := unix.Setresuid(runUID, runUID, runUID); err != nil {
		return fmt.Errorf("setresuid: %w", err)
	}
	// Zero all capability sets on this thread (which becomes the workload).
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3, Pid: 0}
	var data [2]unix.CapUserData
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		return fmt.Errorf("capset: %w", err)
	}
	return nil
}
