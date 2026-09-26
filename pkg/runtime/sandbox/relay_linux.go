//go:build linux

package sandbox

import (
	"fmt"
	"io"
	"net"
	"os"
	goruntime "runtime"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// relay makes a RESTRICTED workload's single declared port reachable from
// the host: the agent listens on a host loopback port and, for each
// connection, dials 127.0.0.1:<inner> inside the workload's network
// namespace. The workload itself has no route out.
type relay struct {
	ln    net.Listener
	port  int64
	pid   int
	inner int64
	once  sync.Once
}

func (r *relay) close() { r.once.Do(func() { r.ln.Close() }) }

// ensureRelay starts (or keeps) the relay for h, on hostPort when given.
func (e *Engine) ensureRelay(h Handle, inner, hostPort int64) (*relay, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r := e.relays[h.ID]; r != nil && r.pid == h.PID {
		return r, nil
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", hostPort))
	if err != nil {
		return nil, fmt.Errorf("relay listen: %w", err)
	}
	r := &relay{ln: ln, port: int64(ln.Addr().(*net.TCPAddr).Port), pid: h.PID, inner: inner}
	e.relays[h.ID] = r
	go r.serve()
	return r, nil
}

func (r *relay) serve() {
	for {
		c, err := r.ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			up, err := dialInNetns(r.pid, fmt.Sprintf("127.0.0.1:%d", r.inner))
			if err != nil {
				return
			}
			defer up.Close()
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(up, c); closeWrite(up); done <- struct{}{} }()
			go func() { _, _ = io.Copy(c, up); closeWrite(c); done <- struct{}{} }()
			<-done
			<-done
		}()
	}
}

func closeWrite(c net.Conn) {
	if t, ok := c.(*net.TCPConn); ok {
		_ = t.CloseWrite()
	}
}

// dialInNetns opens a TCP connection from inside pid's network namespace.
// The socket is created on a locked thread switched into that namespace; a
// socket keeps its namespace for life, so the thread switches back at once.
// If switching back fails the thread is left locked and dies with the
// goroutine rather than serve other work in the wrong namespace.
func dialInNetns(pid int, addr string) (net.Conn, error) {
	target, err := os.Open(fmt.Sprintf("/proc/%d/ns/net", pid))
	if err != nil {
		return nil, err
	}
	defer target.Close()
	type result struct {
		c   net.Conn
		err error
	}
	ch := make(chan result, 1)
	go func() {
		goruntime.LockOSThread()
		own, err := os.Open(fmt.Sprintf("/proc/self/task/%d/ns/net", unix.Gettid()))
		if err != nil {
			goruntime.UnlockOSThread()
			ch <- result{err: err}
			return
		}
		defer own.Close()
		if err := unix.Setns(int(target.Fd()), unix.CLONE_NEWNET); err != nil {
			goruntime.UnlockOSThread()
			ch <- result{err: fmt.Errorf("setns: %w", err)}
			return
		}
		fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
		back := unix.Setns(int(own.Fd()), unix.CLONE_NEWNET)
		if back == nil {
			goruntime.UnlockOSThread()
		}
		if err != nil {
			ch <- result{err: err}
			return
		}
		// Connect from any thread: the socket already belongs to the sandbox.
		f := os.NewFile(uintptr(fd), "relay")
		tcp, _ := net.ResolveTCPAddr("tcp", addr)
		sa := &unix.SockaddrInet4{Port: tcp.Port}
		copy(sa.Addr[:], tcp.IP.To4())
		if err := unix.Connect(fd, sa); err != nil {
			f.Close()
			ch <- result{err: err}
			return
		}
		c, err := net.FileConn(f)
		f.Close()
		ch <- result{c: c, err: err}
	}()
	select {
	case r := <-ch:
		return r.c, r.err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("relay: dial %s timed out", addr)
	}
}
