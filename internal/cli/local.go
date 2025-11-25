package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func resolveLocalURL(explicit string) string {
	if explicit != "" {
		return strings.TrimRight(addHTTP(explicit), "/")
	}
	if env := os.Getenv("TRANSIRE_HTTP_ADDR"); env != "" {
		return strings.TrimRight(addHTTP(env), "/")
	}
	if env := os.Getenv("PORT"); env != "" {
		return fmt.Sprintf("http://localhost:%s", env)
	}
	if env := os.Getenv("TRANSIRE_PORT"); env != "" {
		return fmt.Sprintf("http://localhost:%s", env)
	}
	return "http://localhost:8080"
}

func addHTTP(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}

func ensureLocalRunning(ctx context.Context) (string, error) {
	base := resolveLocalURL("")
	url := fmt.Sprintf("%s/_transire/health", base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("local transire run not reachable at %s: %w", base, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("local transire run returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return base, nil
}

func sendLocalQueue(ctx context.Context, baseURL, queue string, payload []byte) error {
	url := fmt.Sprintf("%s/_transire/queues/%s", resolveLocalURL(baseURL), queue)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("local send failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func triggerLocalSchedule(ctx context.Context, baseURL, schedule string) error {
	url := fmt.Sprintf("%s/_transire/schedules/%s", resolveLocalURL(baseURL), schedule)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("local trigger failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
