package mesh

import (
	"io"
	"net/http"
	"testing"
	"time"
)

// TestThroughputMeasured transfers data through the tunnel and reports the
// measured rate (no assertion on speed; the number is evidence, not a claim).
func TestThroughputMeasured(t *testing.T) {
	a, b := pair(t)
	ln, err := b.ListenTCP(7801)
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 64<<10)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload) })}
	go srv.Serve(ln)
	defer srv.Close()
	c := a.HTTPClient(10 * time.Second)
	start := time.Now()
	n := 0
	for i := 0; i < 100; i++ {
		resp, err := c.Get("http://10.77.0.2:7801/")
		if err != nil {
			t.Fatal(err)
		}
		m, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		n += int(m)
	}
	d := time.Since(start)
	t.Logf("100 sequential 64KiB requests over userspace WireGuard: %v total, %.1f ms/request, %.1f MiB/s", d, float64(d.Milliseconds())/100, float64(n)/d.Seconds()/(1<<20))
}
