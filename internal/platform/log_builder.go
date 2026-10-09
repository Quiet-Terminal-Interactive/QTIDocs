package platform

import (
	"context"
	"log"
)

type LogBuilder struct{}

func (LogBuilder) Enqueue(_ context.Context, job BuildJob) error {
	log.Printf("platform: build triggered for %q (%s@%s:%s)", job.Subdomain, job.Repo, job.Branch, job.Path)
	return nil
}

func (LogBuilder) Teardown(_ context.Context, subdomain string) error {
	log.Printf("platform: sweep would tear down %q — no worker wired up", subdomain)
	return nil
}
