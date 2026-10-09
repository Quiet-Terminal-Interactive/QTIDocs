package platform

import "context"

type BuildJob struct {
	Subdomain string `json:"subdomain"`
	Title     string `json:"title,omitempty"`
	Repo      string `json:"repo"`
	Branch    string `json:"branch"`
	Path      string `json:"path"`
}

type Builder interface {
	Enqueue(ctx context.Context, job BuildJob) error
	Teardown(ctx context.Context, subdomain string) error
}
