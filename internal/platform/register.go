package platform

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/mailer"
	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type registerRequest struct {
	Subdomain      string            `json:"subdomain"`
	Title          string            `json:"title"`
	Repo           string            `json:"repo"`
	Branch         string            `json:"branch"`
	Path           string            `json:"path"`
	Contact        string            `json:"contact"`
	Versions       []storage.Version `json:"versions,omitempty"`
	DefaultVersion string            `json:"default_version,omitempty"`
}

func RegisterHandler(store storage.Store, builder Builder, mail mailer.Mailer, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if secret == "" || !validBearer(r.Header.Get("Authorization"), secret) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Subdomain == "" || req.Repo == "" || req.Branch == "" || req.Path == "" || req.Contact == "" {
			http.Error(w, "subdomain, repo, branch, path, and contact are all required", http.StatusBadRequest)
			return
		}
		if _, err := mailer.ParseAddress(req.Contact); err != nil {
			http.Error(w, "contact: "+err.Error(), http.StatusBadRequest)
			return
		}

		existing, _, err := store.Get(req.Subdomain)
		if err != nil {
			log.Printf("platform: reading existing entry for %q: %v", req.Subdomain, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		webhookSecret := existing.WebhookSecret
		rotate := webhookSecret == "" || ownerChanged(existing, req)
		if rotate {
			if existing.WebhookSecret != "" {
				if err := teardownPreviews(r.Context(), store, builder, req.Subdomain); err != nil {
					log.Printf("platform: tearing down previews for %q: %v", req.Subdomain, err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
			}

			webhookSecret, err = storage.GenerateSecret()
			if err != nil {
				log.Printf("platform: generating webhook secret for %q: %v", req.Subdomain, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			msg, err := deploySecretEmail(req, webhookSecret)
			if err != nil {
				log.Printf("platform: building deploy secret email for %q: %v", req.Subdomain, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if err := mail.Send(r.Context(), msg); err != nil {
				log.Printf("platform: emailing deploy secret for %q: %v", req.Subdomain, err)
				http.Error(w, "failed to email deploy secret to contact", http.StatusBadGateway)
				return
			}
		}

		if err := store.Set(storage.Entry{
			Subdomain:        req.Subdomain,
			Title:            req.Title,
			Repo:             req.Repo,
			Branch:           req.Branch,
			Path:             req.Path,
			Contact:          req.Contact,
			OutputDir:        existing.OutputDir,
			WebhookSecret:    webhookSecret,
			Versions:         req.Versions,
			DefaultVersion:   req.DefaultVersion,
			DeployedVersions: existing.DeployedVersions,
		}); err != nil {
			log.Printf("platform: writing routing entry for %q: %v", req.Subdomain, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := builder.Enqueue(r.Context(), BuildJob{
			Subdomain: req.Subdomain,
			Title:     req.Title,
			Repo:      req.Repo,
			Branch:    req.Branch,
			Path:      req.Path,
		}); err != nil {
			log.Printf("platform: enqueuing build for %q: %v", req.Subdomain, err)
			http.Error(w, "registered, but failed to enqueue build", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"accepted","subdomain":%q,"secret_emailed":%t}`, req.Subdomain, rotate)
	})
}

func ownerChanged(existing storage.Entry, req registerRequest) bool {
	if !sameRepo(existing.Repo, req.Repo) {
		return true
	}
	return existing.Contact != "" && !strings.EqualFold(existing.Contact, req.Contact)
}

func sameRepo(a, b string) bool {
	aOwner, aName, aErr := registry.ParseRepo(a)
	bOwner, bName, bErr := registry.ParseRepo(b)
	if aErr != nil || bErr != nil {
		return a == b
	}
	return strings.EqualFold(aOwner, bOwner) && strings.EqualFold(aName, bName)
}

func teardownPreviews(ctx context.Context, store storage.Store, builder Builder, subdomain string) error {
	entries, err := store.List()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if parent, ok := previewParent(e.Subdomain); ok && parent == subdomain {
			if err := builder.Teardown(ctx, e.Subdomain); err != nil {
				log.Printf("platform: tearing down preview %q for %q: %v", e.Subdomain, subdomain, err)
				continue
			}
		}
	}
	return nil
}

func validBearer(header, secret string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	token := header[len(prefix):]
	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}
