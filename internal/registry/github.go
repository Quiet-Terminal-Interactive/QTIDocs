package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type GitHub interface {
	HasWriteAccess(ctx context.Context, owner, repo, user string) (bool, error)
	OrgMember(ctx context.Context, org, user string) (bool, error)
	PathExists(ctx context.Context, owner, repo, ref, path string) (bool, error)
}

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

func (g HTTPGitHub) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL()+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	return g.client().Do(req)
}

func (g HTTPGitHub) HasWriteAccess(ctx context.Context, owner, repo, user string) (bool, error) {
	resp, err := g.get(ctx, fmt.Sprintf("/repos/%s/%s/collaborators/%s/permission", owner, repo, user))
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("registry: GitHub permission check for %s/%s: unexpected status %s", owner, repo, resp.Status)
	}

	var body struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, fmt.Errorf("registry: decoding permission response: %w", err)
	}
	return body.Permission == "admin" || body.Permission == "write", nil
}

func (g HTTPGitHub) OrgMember(ctx context.Context, org, user string) (bool, error) {
	resp, err := g.get(ctx, fmt.Sprintf("/orgs/%s/members/%s", org, user))
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("registry: GitHub org membership check for %s: unexpected status %s", org, resp.Status)
	}
}

func (g HTTPGitHub) PathExists(ctx context.Context, owner, repo, ref, path string) (bool, error) {
	resp, err := g.get(ctx, fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref))
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("registry: GitHub contents check for %s/%s@%s:%s: unexpected status %s", owner, repo, ref, path, resp.Status)
	}
}
