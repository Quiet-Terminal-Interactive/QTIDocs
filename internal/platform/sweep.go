package platform

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

var previewSubdomainPattern = regexp.MustCompile(`^pr-[0-9]+\.(.+)$`)

func previewParent(subdomain string) (string, bool) {
	m := previewSubdomainPattern.FindStringSubmatch(subdomain)
	if m == nil {
		return "", false
	}
	return m[1], true
}

type sweepRequest struct {
	Registered []string `json:"registered"`
}

func SweepHandler(store storage.Store, builder Builder, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if secret == "" || !validBearer(r.Header.Get("Authorization"), secret) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req sweepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		registered := make(map[string]bool, len(req.Registered))
		for _, s := range req.Registered {
			registered[s] = true
		}

		entries, err := store.List()
		if err != nil {
			log.Printf("platform: listing routing entries for sweep: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		torn := 0
		for _, e := range entries {
			name := e.Subdomain
			if parent, ok := previewParent(name); ok {
				name = parent
			}
			if registered[name] {
				continue
			}
			if err := builder.Teardown(r.Context(), e.Subdomain); err != nil {
				log.Printf("platform: tearing down unregistered subdomain %q: %v", e.Subdomain, err)
				continue
			}
			torn++
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"accepted","torn_down":%d}`, torn)
	})
}
