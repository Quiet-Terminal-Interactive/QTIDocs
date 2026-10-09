package webhook

import (
	"fmt"
	"log"
	"net/http"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

func PreviewSubdomain(parent string, pr int) string {
	return fmt.Sprintf("pr-%d.%s", pr, parent)
}

type previewDeployRequest struct {
	Repo string `json:"repo"`
	PR   int    `json:"pr"`
	Ref  string `json:"ref"`
}

type previewTeardownRequest struct {
	Repo string `json:"repo"`
	PR   int    `json:"pr"`
}

func PreviewDeployHandler(store storage.Store, enqueue func(build.Job)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, req, ok := readJSONBody[previewDeployRequest](w, r)
		if !ok {
			return
		}
		if req.Repo == "" || req.Ref == "" || req.PR <= 0 {
			http.Error(w, "repo, ref, and a positive pr are all required", http.StatusBadRequest)
			return
		}

		owner, name, matches, ok := authorizeMatches(w, store, body, r.Header.Get(SignatureHeader), req.Repo)
		if !ok {
			return
		}

		jobs := 0
		for _, e := range matches {
			preview := PreviewSubdomain(e.Subdomain, req.PR)

			existing, _, err := store.Get(preview)
			if err != nil {
				log.Printf("webhook: reading preview entry %q: %v", preview, err)
				continue
			}
			if err := store.Set(storage.Entry{
				Subdomain: preview,
				Title:     e.Title,
				Repo:      e.Repo,
				Path:      e.Path,
				OutputDir: existing.OutputDir,
			}); err != nil {
				log.Printf("webhook: registering preview entry %q: %v", preview, err)
				continue
			}

			enqueue(build.Job{Subdomain: preview, Title: e.Title, Owner: owner, Repo: name, Ref: req.Ref, Path: e.Path})
			jobs++
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"status":"accepted","jobs":%d}`, jobs)
	})
}

func PreviewTeardownHandler(store storage.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, req, ok := readJSONBody[previewTeardownRequest](w, r)
		if !ok {
			return
		}
		if req.Repo == "" || req.PR <= 0 {
			http.Error(w, "repo and a positive pr are both required", http.StatusBadRequest)
			return
		}

		_, _, matches, ok := authorizeMatches(w, store, body, r.Header.Get(SignatureHeader), req.Repo)
		if !ok {
			return
		}

		for _, e := range matches {
			preview := PreviewSubdomain(e.Subdomain, req.PR)
			if err := build.Teardown(store, preview); err != nil {
				log.Printf("webhook: tearing down preview %q: %v", preview, err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, `{"status":"accepted"}`)
	})
}
