package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type hostClient struct{ addr string }

func (h *hostClient) post(path string, body, out any) error {
	b, _ := json.Marshal(body)
	resp, err := http.Post("http://"+h.addr+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
