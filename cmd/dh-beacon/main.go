// dh-beacon is the reference workload: a small HTTP service that reports
// which assignment, generation and host it runs as, exposes a health check,
// and stores key/value data in its dh volume so storage replication and
// restore can be demonstrated end to end.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var keyRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func main() {
	addr := os.Getenv("DH_LISTEN")
	if addr == "" {
		addr = "0.0.0.0:" + envOr("PORT", "8080")
	}
	data := os.Getenv("DH_VOLUME_DATA")
	started := time.Now()
	host, _ := os.Hostname()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "app": os.Getenv("DH_APP"), "assignment": os.Getenv("DH_ASSIGNMENT"), "generation": os.Getenv("DH_GENERATION"),
			"replica": os.Getenv("DH_REPLICA"), "node": os.Getenv("DH_NODE"), "pid": os.Getpid(), "hostname": host,
			"uptimeMs": time.Since(started).Milliseconds(), "message": os.Getenv("MESSAGE"), "volume": data != "",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if data != "" {
			if _, err := os.Stat(filepath.Join(data, ".unhealthy")); err == nil {
				http.Error(w, "unhealthy flag present", http.StatusServiceUnavailable)
				return
			}
		}
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/kv/", func(w http.ResponseWriter, r *http.Request) {
		if data == "" {
			http.Error(w, "no volume attached", http.StatusNotImplemented)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, "/kv/")
		if key == "" && r.Method == http.MethodGet {
			entries, _ := os.ReadDir(data)
			var keys []string
			for _, e := range entries {
				if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					keys = append(keys, e.Name())
				}
			}
			_ = json.NewEncoder(w).Encode(keys)
			return
		}
		if !keyRE.MatchString(key) {
			http.Error(w, "bad key", 400)
			return
		}
		p := filepath.Join(data, key)
		switch r.Method {
		case http.MethodGet:
			b, err := os.ReadFile(p)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(b)
		case http.MethodPut, http.MethodPost:
			b, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			tmp := p + ".tmp"
			if err := os.WriteFile(tmp, b, 0o644); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if f, err := os.Open(tmp); err == nil {
				_ = f.Sync()
				f.Close()
			}
			if err := os.Rename(tmp, p); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			_ = os.Remove(p)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", 405)
		}
	})
	log.Printf("dh-beacon %s gen %s listening on %s (volume %q)", os.Getenv("DH_ASSIGNMENT"), os.Getenv("DH_GENERATION"), addr, data)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
