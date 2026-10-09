package dispute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPGitHub struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

func (g HTTPGitHub) baseURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	return "https://api.github.com"
}

func (g HTTPGitHub) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}

func (g HTTPGitHub) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("dispute: encoding request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, g.baseURL()+path, reader)
	if err != nil {
		return nil, fmt.Errorf("dispute: building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	return g.client().Do(req)
}

func (g HTTPGitHub) ListOpenIssues(ctx context.Context, owner, repo, label string) ([]Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues?state=open&labels=%s&per_page=100", owner, repo, label)
	resp, err := g.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("dispute: listing issues on %s/%s: %w", owner, repo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dispute: listing issues on %s/%s: unexpected status %s", owner, repo, resp.Status)
	}

	var raw []struct {
		Number int    `json:"number"`
		Body   string `json:"body"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("dispute: decoding issues response: %w", err)
	}

	issues := make([]Issue, 0, len(raw))
	for _, r := range raw {
		labels := make([]string, 0, len(r.Labels))
		for _, l := range r.Labels {
			labels = append(labels, l.Name)
		}
		issues = append(issues, Issue{Number: r.Number, Body: r.Body, Labels: labels})
	}
	return issues, nil
}

func (g HTTPGitHub) CommentOnIssue(ctx context.Context, owner, repo string, number int, body string) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
	resp, err := g.do(ctx, http.MethodPost, path, struct {
		Body string `json:"body"`
	}{body})
	if err != nil {
		return fmt.Errorf("dispute: commenting on %s/%s#%d: %w", owner, repo, number, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("dispute: commenting on %s/%s#%d: unexpected status %s", owner, repo, number, resp.Status)
	}
	return nil
}

func (g HTTPGitHub) AddLabel(ctx context.Context, owner, repo string, number int, label string) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", owner, repo, number)
	resp, err := g.do(ctx, http.MethodPost, path, struct {
		Labels []string `json:"labels"`
	}{[]string{label}})
	if err != nil {
		return fmt.Errorf("dispute: labeling %s/%s#%d: %w", owner, repo, number, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dispute: labeling %s/%s#%d: unexpected status %s", owner, repo, number, resp.Status)
	}
	return nil
}

func (g HTTPGitHub) CloseIssue(ctx context.Context, owner, repo string, number int, reason string) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
	resp, err := g.do(ctx, http.MethodPatch, path, struct {
		State       string `json:"state"`
		StateReason string `json:"state_reason"`
	}{"closed", reason})
	if err != nil {
		return fmt.Errorf("dispute: closing %s/%s#%d: %w", owner, repo, number, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dispute: closing %s/%s#%d: unexpected status %s", owner, repo, number, resp.Status)
	}
	return nil
}
