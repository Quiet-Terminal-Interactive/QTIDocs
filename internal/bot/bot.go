package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
)

type Reporter interface {
	ReportBuildFailure(ctx context.Context, job build.Job, buildErr error) error
}

type GitHubReporter struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

func (r GitHubReporter) baseURL() string {
	if r.BaseURL != "" {
		return r.BaseURL
	}
	return "https://api.github.com"
}

func (r GitHubReporter) client() *http.Client {
	if r.Client != nil {
		return r.Client
	}
	return http.DefaultClient
}

func (r GitHubReporter) ReportBuildFailure(ctx context.Context, job build.Job, buildErr error) error {
	if r.Token == "" {
		return nil
	}

	body := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{
		Title: fmt.Sprintf("qtidocs build failed for %s (%s)", job.Subdomain, job.Ref),
		Body: fmt.Sprintf(
			"qtidocs tried to build and deploy this repo's `%s` folder at `%s` for **%s.qtidocs.dev**, and the build failed:\n\n```\n%s\n```\n\nThe site is still serving its last successful build — no action is needed to keep it online. Push a fix (or re-run the failing commit) to deploy again.",
			job.Path, job.Ref, job.Subdomain, buildErr.Error(),
		),
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("bot: encoding issue body: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", r.baseURL(), job.Owner, job.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("bot: building issue request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.Token)

	resp, err := r.client().Do(req)
	if err != nil {
		return fmt.Errorf("bot: opening issue on %s/%s: %w", job.Owner, job.Repo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("bot: opening issue on %s/%s: unexpected status %s", job.Owner, job.Repo, resp.Status)
	}
	return nil
}
