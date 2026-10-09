// Package ntfy sends push notifications through an ntfy server
// (https://ntfy.sh or self-hosted). Pure Go, so it works on every OS.
package ntfy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	Server string
	Topic  string
	Token  string
	// Priority is an ntfy priority name. "urgent" makes Android show a
	// heads-up banner over the current app and vibrate longer.
	Priority string
	HTTP     *http.Client
}

func (c *Client) Send(ctx context.Context, title, body string) error {
	url := strings.TrimRight(c.Server, "/") + "/" + c.Topic
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", "bell")
	if c.Priority != "" {
		req.Header.Set("Priority", c.Priority)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("ntfy: %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	return nil
}
