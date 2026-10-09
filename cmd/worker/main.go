package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/bot"
	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/queue"
	"github.com/quiet-terminal-interactive/qtidocs/internal/reconcile"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
	"github.com/quiet-terminal-interactive/qtidocs/internal/webhook"
)

func main() {
	routing := flag.String("routing", "", "path to the routing table file (see internal/storage)")
	sitesRoot := flag.String("sites-root", "", "base directory rendered site output is written under")
	addr := flag.String("addr", ":8082", "listen address")
	secret := flag.String("secret", os.Getenv("QTIDOCS_WORKER_SECRET"), "shared secret authenticating cmd/platform's internal build-trigger calls (defaults to $QTIDOCS_WORKER_SECRET)")
	githubToken := flag.String("github-token", os.Getenv("QTIDOCS_GITHUB_TOKEN"), "GitHub token for tarball fetches and reconciliation ref lookups (optional; raises the unauthenticated rate limit; defaults to $QTIDOCS_GITHUB_TOKEN)")
	botToken := flag.String("bot-token", os.Getenv("QTIDOCS_BOT_TOKEN"), "Quiet-Terminal-Bot's GitHub token, used to open an issue on a repo when its build fails (optional; defaults to $QTIDOCS_BOT_TOKEN; failure issues are skipped entirely if unset)")
	reconcileInterval := flag.Duration("reconcile-interval", 20*time.Minute, "how often the reconciliation cron compares each registered repo's branch-tip SHA against its last-deployed SHA and re-enqueues a build on mismatch")

	maxExtractedBytes := flag.Int64("max-extracted-bytes", build.DefaultLimits().MaxExtractedBytes, "max decompressed bytes extracted from a source tarball's qtidocs/ path")
	maxFiles := flag.Int("max-files", build.DefaultLimits().MaxFiles, "max tar entries scanned per build")
	maxBuildTime := flag.Duration("max-build-time", build.DefaultLimits().MaxBuildTime, "max combined fetch+render time per build")
	maxOutputBytes := flag.Int64("max-output-bytes", build.DefaultLimits().MaxOutputBytes, "max rendered output size per build")
	flag.Parse()

	if *routing == "" {
		fmt.Fprintln(os.Stderr, "worker: -routing is required")
		flag.Usage()
		os.Exit(2)
	}
	if *sitesRoot == "" {
		fmt.Fprintln(os.Stderr, "worker: -sites-root is required")
		flag.Usage()
		os.Exit(2)
	}
	if *secret == "" {
		fmt.Fprintln(os.Stderr, "worker: -secret (or $QTIDOCS_WORKER_SECRET) is required")
		flag.Usage()
		os.Exit(2)
	}

	store := storage.Open(*routing)
	pipeline := build.Pipeline{
		Fetcher:   build.GitHubFetcher{Token: *githubToken},
		Store:     store,
		SitesRoot: *sitesRoot,
		Limits: build.Limits{
			MaxExtractedBytes: *maxExtractedBytes,
			MaxFiles:          *maxFiles,
			MaxBuildTime:      *maxBuildTime,
			MaxOutputBytes:    *maxOutputBytes,
		},
	}

	reporter := bot.GitHubReporter{Token: *botToken}
	if *botToken == "" {
		fmt.Fprintln(os.Stderr, "worker: -bot-token not set; build failures will not open a GitHub issue on the source repo, see internal/bot.GitHubReporter")
	}

	q := queue.New(func(ctx context.Context, job build.Job) {
		if err := pipeline.Run(ctx, job); err != nil {
			fmt.Fprintf(os.Stderr, "worker: build failed for %q (%s/%s@%s:%s): %v\n", job.Subdomain, job.Owner, job.Repo, job.Ref, job.Path, err)
			if rerr := reporter.ReportBuildFailure(context.Background(), job, err); rerr != nil {
				fmt.Fprintf(os.Stderr, "worker: failed to open failure issue on %s/%s: %v\n", job.Owner, job.Repo, rerr)
			}
		}
	})
	enqueue := func(job build.Job) {
		q.Enqueue(context.Background(), job.Subdomain, job)
	}

	reconciler := reconcile.Reconciler{
		GitHub:  reconcile.HTTPGitHub{Token: *githubToken},
		Store:   store,
		Enqueue: enqueue,
	}
	go runReconciliationLoop(reconciler, *reconcileInterval)

	mux := http.NewServeMux()
	mux.Handle("/deploy", webhook.Handler(store, enqueue))
	mux.Handle("/internal/build", webhook.InternalBuildHandler(store, enqueue, *secret))
	mux.Handle("/internal/teardown", webhook.InternalTeardownHandler(store, *secret))
	mux.Handle("/preview/deploy", webhook.PreviewDeployHandler(store, enqueue))
	mux.Handle("/preview/teardown", webhook.PreviewTeardownHandler(store))

	fmt.Fprintf(os.Stderr, "worker: listening on %s, routing table at %s, output under %s, reconciling every %s\n", *addr, *routing, *sitesRoot, *reconcileInterval)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
}

func runReconciliationLoop(reconciler reconcile.Reconciler, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		n, err := reconciler.Run(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "worker: reconciliation pass failed: %v\n", err)
			continue
		}
		if n > 0 {
			fmt.Fprintf(os.Stderr, "worker: reconciliation re-enqueued %d build(s) for mismatched/never-deployed subdomains\n", n)
		}
	}
}
