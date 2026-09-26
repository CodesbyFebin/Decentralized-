//go:build linux

package sandbox

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// cgroups enforces limits under a per-instance group. It supports the
// unified hierarchy (v2) and the legacy per-controller hierarchy (v1).
type cgroups struct {
	v2    bool
	paths map[string]string // controller → directory (v2: "" → directory)
}

const cgroupRoot = "/sys/fs/cgroup"

// ownCgroup returns this process's cgroup path for a v1 controller, or the
// v2 path when controller is "".
func ownCgroup(controller string) (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		if controller == "" && parts[0] == "0" && parts[1] == "" {
			return parts[2], nil
		}
		for _, c := range strings.Split(parts[1], ",") {
			if c == controller {
				return parts[2], nil
			}
		}
	}
	return "", fmt.Errorf("cgroup: %q not found in /proc/self/cgroup", controller)
}

func cgroupV2() bool {
	_, err := os.Stat(filepath.Join(cgroupRoot, "cgroup.controllers"))
	return err == nil
}

// v1Controllers are the controllers limits need on the legacy hierarchy.
var v1Controllers = []string{"memory", "pids", "cpu"}

// newCgroups creates <own cgroup>/dh-sandbox/<id> and applies limits.
func newCgroups(id string, l Limits) (*cgroups, error) {
	cg := &cgroups{v2: cgroupV2(), paths: map[string]string{}}
	if cg.v2 {
		own, err := ownCgroup("")
		if err != nil {
			return nil, err
		}
		parent := filepath.Join(cgroupRoot, own, "dh-sandbox")
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return nil, fmt.Errorf("cgroup v2: %w", err)
		}
		// Controllers must be enabled for the children of both our own group
		// and dh-sandbox. With processes in our own group this fails (the
		// "no internal processes" rule): then the host must delegate a group.
		for _, dir := range []string{filepath.Join(cgroupRoot, own), parent} {
			if err := os.WriteFile(filepath.Join(dir, "cgroup.subtree_control"), []byte("+memory +pids +cpu"), 0o644); err != nil {
				return nil, fmt.Errorf("cgroup v2: cannot enable memory/pids/cpu under %s (%v); run dh-noded in a delegated cgroup (systemd Delegate=yes)", dir, err)
			}
		}
		dir := filepath.Join(parent, id)
		if err := os.Mkdir(dir, 0o755); err != nil {
			return nil, fmt.Errorf("cgroup v2: %w", err)
		}
		cg.paths[""] = dir
	} else {
		for _, c := range v1Controllers {
			own, err := ownCgroup(c)
			if err != nil {
				cg.remove()
				return nil, err
			}
			dir := filepath.Join(cgroupRoot, c, own, "dh-sandbox", id)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				cg.remove()
				return nil, fmt.Errorf("cgroup v1 %s: %w", c, err)
			}
			cg.paths[c] = dir
		}
	}
	if err := cg.limit(l); err != nil {
		cg.remove()
		return nil, err
	}
	return cg, nil
}

func write(dir, file, val string) error {
	if err := os.WriteFile(filepath.Join(dir, file), []byte(val), 0o644); err != nil {
		return fmt.Errorf("cgroup %s=%s: %w", file, val, err)
	}
	return nil
}

func (cg *cgroups) limit(l Limits) error {
	if cg.v2 {
		d := cg.paths[""]
		if l.MemBytes > 0 {
			if err := write(d, "memory.max", strconv.FormatInt(l.MemBytes, 10)); err != nil {
				return err
			}
			_ = write(d, "memory.swap.max", "0")
		}
		if l.PIDs > 0 {
			if err := write(d, "pids.max", strconv.FormatInt(l.PIDs, 10)); err != nil {
				return err
			}
		}
		if l.CPUMilli > 0 {
			if err := write(d, "cpu.max", fmt.Sprintf("%d 100000", l.CPUMilli*100)); err != nil {
				return err
			}
		}
		return nil
	}
	if l.MemBytes > 0 {
		if err := write(cg.paths["memory"], "memory.limit_in_bytes", strconv.FormatInt(l.MemBytes, 10)); err != nil {
			return err
		}
		// Without swap accounting the file is absent; memory.limit_in_bytes still holds.
		_ = write(cg.paths["memory"], "memory.memsw.limit_in_bytes", strconv.FormatInt(l.MemBytes, 10))
		_ = write(cg.paths["memory"], "memory.swappiness", "0")
	}
	if l.PIDs > 0 {
		if err := write(cg.paths["pids"], "pids.max", strconv.FormatInt(l.PIDs, 10)); err != nil {
			return err
		}
	}
	if l.CPUMilli > 0 {
		if err := write(cg.paths["cpu"], "cpu.cfs_period_us", "100000"); err != nil {
			return err
		}
		if err := write(cg.paths["cpu"], "cpu.cfs_quota_us", strconv.FormatInt(l.CPUMilli*100, 10)); err != nil {
			return err
		}
	}
	return nil
}

// add moves a process into every group.
func (cg *cgroups) add(pid int) error {
	for c, d := range cg.paths {
		if err := write(d, "cgroup.procs", strconv.Itoa(pid)); err != nil {
			return fmt.Errorf("cgroup %s: %w", c, err)
		}
	}
	return nil
}

// oomKills reports how many times the kernel OOM-killed inside the group.
func (cg *cgroups) oomKills() int64 {
	var file, dir string
	if cg.v2 {
		dir, file = cg.paths[""], "memory.events"
	} else {
		dir, file = cg.paths["memory"], "memory.oom_control"
	}
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == "oom_kill" {
			n, _ := strconv.ParseInt(f[1], 10, 64)
			return n
		}
	}
	return 0
}

// file returns the path of a control file (tests read limits back).
func (cg *cgroups) file(controller, name string) string {
	if cg.v2 {
		return filepath.Join(cg.paths[""], name)
	}
	return filepath.Join(cg.paths[controller], name)
}

// remove deletes the groups; a group still holding processes stays.
func (cg *cgroups) remove() error {
	var errs []error
	for _, d := range cg.paths {
		if err := os.Remove(d); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// cgroupsFor re-opens the groups of an existing instance (after an agent restart).
func cgroupsFor(id string) *cgroups {
	cg := &cgroups{v2: cgroupV2(), paths: map[string]string{}}
	if cg.v2 {
		if own, err := ownCgroup(""); err == nil {
			cg.paths[""] = filepath.Join(cgroupRoot, own, "dh-sandbox", id)
		}
		return cg
	}
	for _, c := range v1Controllers {
		if own, err := ownCgroup(c); err == nil {
			cg.paths[c] = filepath.Join(cgroupRoot, c, own, "dh-sandbox", id)
		}
	}
	return cg
}
