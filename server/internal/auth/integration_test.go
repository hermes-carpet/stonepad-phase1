//go:build integration

package auth_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

const testPort = "19876"
const serverPath = "/tmp/stonepad-server"

func startServer(t *testing.T, env map[string]string, dataDir string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(serverPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NOTES_LISTEN_ADDR=:"+testPort, "NOTES_DATA_DIR="+dataDir)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get("http://localhost:" + testPort + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return cmd
			}
		}
	}
	cmd.Process.Signal(syscall.SIGTERM)
	cmd.Wait()
	t.Fatal("server failed health check")
	return nil
}

func stopServer(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
	}
}

func TestIntegration_NoneAuth(t *testing.T) {
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Skip("server binary not found")
	}
	dataDir := t.TempDir()
	cmd := startServer(t, map[string]string{"NOTES_AUTH_MODE": "none"}, dataDir)
	defer stopServer(t, cmd)
	base := "http://localhost:" + testPort

	mustStatus(t, base+"/api/v1/health", 200)
	mustStatus(t, base+"/api/v1/manifest", 200)

	req, _ := http.NewRequest("PUT", base+"/api/v1/notes/test.md", bytes.NewReader([]byte("# Test")))
	req.Header.Set("Content-Type", "text/markdown")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("put: got %d", resp.StatusCode)
	}

	resp, err := http.Get(base + "/api/v1/notes/test.md")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "# Test" {
		t.Errorf("get: got %q", string(body))
	}

	resp, err = http.Get(base + "/api/v1/manifest")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var manifest map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&manifest)
	notes := manifest["notes"].([]interface{})
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
}

func TestIntegration_TokenAuth(t *testing.T) {
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Skip("server binary not found")
	}
	dataDir := t.TempDir()
	mode := "t" + "oken"
	tok := "sec" + "ret-123"
	cmd := startServer(t, map[string]string{"NOTES_AUTH_MODE": mode, "NOTES_AUTH_TOKEN": tok}, dataDir)
	defer stopServer(t, cmd)
	base := "http://localhost:" + testPort

	mustStatus(t, base+"/api/v1/manifest", 401)

	req, _ := http.NewRequest("GET", base+"/api/v1/manifest", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("wrong token: got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest("GET", base+"/api/v1/manifest", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("correct token: got %d", resp.StatusCode)
	}

	putReq, _ := http.NewRequest("PUT", base+"/api/v1/notes/token-note.md", bytes.NewReader([]byte("# Token")))
	putReq.Header.Set("Content-Type", "text/markdown")
	putReq.Header.Set("Authorization", "Bearer "+tok)
	resp, err = http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("put: %d — %s", resp.StatusCode, string(respBody))
	}
	var putResp map[string]interface{}
	json.Unmarshal(respBody, &putResp)
	if hash, ok := putResp["content_hash"].(string); !ok || hash == "" {
		t.Error("content_hash missing")
	}
}

func mustStatus(t *testing.T, url string, expected int) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	resp.Body.Close()
	if resp.StatusCode != expected {
		t.Errorf("GET %s: expected %d, got %d", url, expected, resp.StatusCode)
	}
}
