package cli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSendCommandLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_transire/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/_transire/queues/work-events":
			body, _ := io.ReadAll(r.Body)
			if string(body) != "demo" {
				t.Fatalf("unexpected payload: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	appDir := filepath.Join(wd, "..", "..", "examples", "all-handlers-cli")
	exitOnError(os.Chdir(appDir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("TRANSIRE_HTTP_ADDR", server.URL)

	cmd := newSendCmd()
	cmd.SetArgs([]string{"work-events", "demo", "--env", "local"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatalf("send cmd failed: %v", err)
	}
}

func TestTriggerCommandLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_transire/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/_transire/schedules/heartbeat":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	appDir := filepath.Join(wd, "..", "..", "examples", "all-handlers-cli")
	exitOnError(os.Chdir(appDir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("TRANSIRE_HTTP_ADDR", server.URL)

	cmd := newTriggerCmd()
	cmd.SetArgs([]string{"heartbeat", "--env", "local"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatalf("trigger cmd failed: %v", err)
	}
}
