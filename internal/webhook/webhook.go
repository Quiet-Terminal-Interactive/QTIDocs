package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

const SignatureHeader = "X-QTIDocs-Signature-256"

const maxBodyBytes = 1 << 16

type deployRequest struct {
	Repo string `json:"repo"`
	Ref  string `json:"ref"`
}

func Verify(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	hexSig, ok := strings.CutPrefix(header, prefix)
	if !ok {
		return false
	}
	sig, err := hex.DecodeString(hexSig)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(sig, mac.Sum(nil))
}

func Handler(store storage.Store, enqueue func(build.Job)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, req, ok := readJSONBody[deployRequest](w, r)
		if !ok {
			return
		}
		if req.Repo == "" || req.Ref == "" {
			http.Error(w, "repo and ref are both required", http.StatusBadRequest)
			return
		}

		owner, name, matches, ok := authorizeMatches(w, store, body, r.Header.Get(SignatureHeader), req.Repo)
		if !ok {
			return
		}
		for _, e := range matches {
			enqueue(jobFor(e, owner, name, req.Ref))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"accepted","jobs":%d}`, len(matches))
	})
}

func readJSONBody[T any](w http.ResponseWriter, r *http.Request) (body []byte, req T, ok bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		http.Error(w, "reading request body: "+err.Error(), http.StatusBadRequest)
		return nil, req, false
	}
	if len(body) > maxBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return nil, req, false
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return nil, req, false
	}
	return body, req, true
}

func authorizeMatches(w http.ResponseWriter, store storage.Store, body []byte, sig, repo string) (owner, name string, matches []storage.Entry, ok bool) {
	owner, name, err := registry.ParseRepo(repo)
	if err != nil {
		http.Error(w, "invalid repo: "+err.Error(), http.StatusBadRequest)
		return "", "", nil, false
	}

	matches, err = matchingEntries(store, body, sig, owner, name)
	if err != nil {
		log.Printf("webhook: listing registered entries: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return "", "", nil, false
	}
	if len(matches) == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return "", "", nil, false
	}
	return owner, name, matches, true
}

func matchingEntries(store storage.Store, body []byte, sig, owner, name string) ([]storage.Entry, error) {
	entries, err := store.List()
	if err != nil {
		return nil, err
	}
	var matched []storage.Entry
	for _, e := range entries {
		eOwner, eName, err := registry.ParseRepo(e.Repo)
		if err != nil || !strings.EqualFold(eOwner, owner) || !strings.EqualFold(eName, name) {
			continue
		}
		if e.WebhookSecret == "" || !Verify(e.WebhookSecret, body, sig) {
			continue
		}
		matched = append(matched, e)
	}
	return matched, nil
}

func jobFor(e storage.Entry, owner, name, ref string) build.Job {
	if len(e.Versions) == 0 {
		return build.Job{Subdomain: e.Subdomain, Title: e.Title, Owner: owner, Repo: name, Ref: ref, Path: e.Path}
	}

	versions := make([]build.Version, len(e.Versions))
	for i, v := range e.Versions {
		versions[i] = build.Version{Name: v.Name, Ref: v.Ref}
	}
	return build.Job{
		Subdomain: e.Subdomain, Title: e.Title, Owner: owner, Repo: name, Path: e.Path,
		Versions: versions, DefaultVersion: e.DefaultVersion,
	}
}
