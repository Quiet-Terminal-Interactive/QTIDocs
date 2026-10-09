package build

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Fetcher interface {
	Fetch(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
}

type GitHubFetcher struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

func (f GitHubFetcher) baseURL() string {
	if f.BaseURL != "" {
		return f.BaseURL
	}
	return "https://api.github.com"
}

func (f GitHubFetcher) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return http.DefaultClient
}

func (f GitHubFetcher) Fetch(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/tarball/%s", f.baseURL(), owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build: building tarball request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if f.Token != "" {
		req.Header.Set("Authorization", "Bearer "+f.Token)
	}

	resp, err := f.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("build: fetching tarball for %s/%s@%s: %w", owner, repo, ref, err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		return nil, fmt.Errorf("build: fetching tarball for %s/%s@%s: unexpected status %s", owner, repo, ref, resp.Status)
	}
	return resp.Body, nil
}

type ExtractResult struct {
	Dir   string
	Files int
	Bytes int64
}

func Extract(r io.Reader, configuredPath, parentDir string, limits Limits) (ExtractResult, error) {
	destDir, err := os.MkdirTemp(parentDir, "extract-*")
	if err != nil {
		return ExtractResult{}, fmt.Errorf("build: creating extraction dir: %w", err)
	}

	res, err := extract(r, configuredPath, destDir, limits)
	if err != nil {
		_ = os.RemoveAll(destDir)
		return ExtractResult{}, err
	}
	res.Dir = destDir
	return res, nil
}

func extract(r io.Reader, configuredPath, destDir string, limits Limits) (ExtractResult, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("build: opening tarball: %w", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)

	var res ExtractResult
	entries := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ExtractResult{}, fmt.Errorf("build: reading tarball: %w", err)
		}

		entries++
		if entries > limits.MaxFiles {
			return ExtractResult{}, fmt.Errorf("build: tarball has more than %d entries, exceeding the file-count limit", limits.MaxFiles)
		}

		rel, ok := relativeToPath(hdr.Name, configuredPath)
		if !ok {
			continue // outside the configured qtidocs/ path: discarded
		}
		if !filepath.IsLocal(filepath.FromSlash(rel)) {
			return ExtractResult{}, fmt.Errorf("build: tarball entry %q resolves outside the extraction root", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if rel == "" {
				continue
			}
			if err := os.MkdirAll(filepath.Join(destDir, rel), 0o755); err != nil {
				return ExtractResult{}, fmt.Errorf("build: %w", err)
			}

		case tar.TypeReg:
			if rel == "" {
				continue
			}
			target := filepath.Join(destDir, rel)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return ExtractResult{}, fmt.Errorf("build: %w", err)
			}
			n, err := extractFile(tr, target, limits.MaxExtractedBytes-res.Bytes)
			if err != nil {
				return ExtractResult{}, err
			}
			res.Bytes += n
			res.Files++

		default:
			// Symlinks, hardlinks, devices, FIFOs, etc
		}
	}

	return res, nil
}

func extractFile(r io.Reader, target string, remaining int64) (int64, error) {
	if remaining <= 0 {
		return 0, fmt.Errorf("build: extracted content exceeds the configured size limit")
	}

	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("build: %w", err)
	}
	defer func() { _ = f.Close() }()

	n, err := io.Copy(f, io.LimitReader(r, remaining+1))
	if err != nil {
		return n, fmt.Errorf("build: writing %s: %w", target, err)
	}
	if n > remaining {
		return n, fmt.Errorf("build: extracted content exceeds the configured size limit")
	}
	return n, nil
}

func relativeToPath(name, configuredPath string) (rel string, ok bool) {
	cleaned := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(name)), "/")
	parts := strings.SplitN(cleaned, "/", 2)
	if len(parts) < 2 {
		return "", false
	}
	rest := strings.TrimPrefix(path.Clean("/"+parts[1]), "/")

	cleanConfigured := strings.Trim(path.Clean("/"+configuredPath), "/")
	if cleanConfigured == "" || cleanConfigured == "." {
		return rest, rest != ""
	}
	if rest == cleanConfigured {
		return "", false
	}

	prefix := cleanConfigured + "/"
	if !strings.HasPrefix(rest, prefix) {
		return "", false
	}
	return strings.TrimPrefix(rest, prefix), true
}
