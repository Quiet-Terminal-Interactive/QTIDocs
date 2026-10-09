package webhook

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type internalBuildRequest struct {
	Subdomain string `json:"subdomain"`
	Repo      string `json:"repo"`
	Branch    string `json:"branch"`
	Path      string `json:"path"`
}

func InternalBuildHandler(store storage.Store, enqueue func(build.Job), secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if secret == "" || !validBearer(r.Header.Get("Authorization"), secret) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req internalBuildRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Subdomain == "" || req.Repo == "" || req.Branch == "" || req.Path == "" {
			http.Error(w, "subdomain, repo, branch, and path are all required", http.StatusBadRequest)
			return
		}

		owner, name, err := registry.ParseRepo(req.Repo)
		if err != nil {
			http.Error(w, "invalid repo: "+err.Error(), http.StatusBadRequest)
			return
		}

		entry, ok, err := store.Get(req.Subdomain)
		if err != nil {
			log.Printf("webhook: reading entry for %q: %v", req.Subdomain, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "subdomain is not registered", http.StatusNotFound)
			return
		}

		enqueue(jobFor(entry, owner, name, req.Branch))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"accepted","subdomain":%q}`, req.Subdomain)
	})
}

type internalTeardownRequest struct {
	Subdomain string `json:"subdomain"`
}

func InternalTeardownHandler(store storage.Store, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if secret == "" || !validBearer(r.Header.Get("Authorization"), secret) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req internalTeardownRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Subdomain == "" {
			http.Error(w, "subdomain is required", http.StatusBadRequest)
			return
		}

		if err := build.Teardown(store, req.Subdomain); err != nil {
			log.Printf("webhook: tearing down %q: %v", req.Subdomain, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"accepted","subdomain":%q}`, req.Subdomain)
	})
}

func validBearer(header, secret string) bool {
	const prefix = "Bearer "
	token, ok := strings.CutPrefix(header, prefix)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}
