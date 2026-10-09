package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPBuilder struct {
	URL         string
	TeardownURL string
	Secret      string
	Client      *http.Client
}

func (b HTTPBuilder) client() *http.Client {
	if b.Client != nil {
		return b.Client
	}
	return http.DefaultClient
}

func (b HTTPBuilder) Enqueue(ctx context.Context, job BuildJob) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("platform: encoding build job: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("platform: building worker request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.Secret)

	resp, err := b.client().Do(req)
	if err != nil {
		return fmt.Errorf("platform: calling worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("platform: worker returned %s", resp.Status)
	}
	return nil
}

func (b HTTPBuilder) Teardown(ctx context.Context, subdomain string) error {
	body, err := json.Marshal(struct {
		Subdomain string `json:"subdomain"`
	}{subdomain})
	if err != nil {
		return fmt.Errorf("platform: encoding teardown request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.TeardownURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("platform: building worker teardown request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.Secret)

	resp, err := b.client().Do(req)
	if err != nil {
		return fmt.Errorf("platform: calling worker teardown: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("platform: worker teardown returned %s", resp.Status)
	}
	return nil
}
