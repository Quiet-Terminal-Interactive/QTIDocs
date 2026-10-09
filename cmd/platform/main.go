package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/quiet-terminal-interactive/qtidocs/internal/mailer"
	"github.com/quiet-terminal-interactive/qtidocs/internal/platform"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

func main() {
	routing := flag.String("routing", "", "path to the routing table file (see internal/storage)")
	addr := flag.String("addr", ":8081", "listen address")
	secret := flag.String("secret", os.Getenv("QTIDOCS_PLATFORM_SECRET"), "shared secret authenticating registration requests (defaults to $QTIDOCS_PLATFORM_SECRET)")
	workerURL := flag.String("worker-url", "", "cmd/worker's internal build-trigger endpoint, e.g. https://worker.internal.qtidocs.dev/internal/build (optional; falls back to only logging triggered builds if unset, see internal/platform.LogBuilder)")
	workerTeardownURL := flag.String("worker-teardown-url", "", "cmd/worker's internal teardown-trigger endpoint, e.g. https://worker.internal.qtidocs.dev/internal/teardown (required if -worker-url is set; used by /internal/sweep)")
	workerSecret := flag.String("worker-secret", os.Getenv("QTIDOCS_WORKER_SECRET"), "shared secret authenticating against cmd/worker's internal build-trigger and teardown-trigger endpoints (defaults to $QTIDOCS_WORKER_SECRET; required if -worker-url is set)")
	smtpHost := flag.String("smtp-host", os.Getenv("QTIDOCS_SMTP_HOST"), "SMTP relay used to email deploy secrets to new sites' contact addresses (defaults to $QTIDOCS_SMTP_HOST; required)")
	smtpPort := flag.Int("smtp-port", envInt("QTIDOCS_SMTP_PORT", 587), "SMTP relay port; 465 uses implicit TLS, anything else requires STARTTLS (defaults to $QTIDOCS_SMTP_PORT, else 587)")
	smtpUsername := flag.String("smtp-username", os.Getenv("QTIDOCS_SMTP_USERNAME"), "SMTP username (defaults to $QTIDOCS_SMTP_USERNAME)")
	smtpPassword := flag.String("smtp-password", os.Getenv("QTIDOCS_SMTP_PASSWORD"), "SMTP password (defaults to $QTIDOCS_SMTP_PASSWORD)")
	smtpFrom := flag.String("smtp-from", os.Getenv("QTIDOCS_SMTP_FROM"), "From address for deploy-secret emails, e.g. noreply@qtidocs.dev (defaults to $QTIDOCS_SMTP_FROM; required)")
	flag.Parse()

	if *routing == "" {
		fmt.Fprintln(os.Stderr, "platform: -routing is required")
		flag.Usage()
		os.Exit(2)
	}
	if *secret == "" {
		fmt.Fprintln(os.Stderr, "platform: -secret (or $QTIDOCS_PLATFORM_SECRET) is required")
		flag.Usage()
		os.Exit(2)
	}
	if *workerURL != "" && *workerSecret == "" {
		fmt.Fprintln(os.Stderr, "platform: -worker-secret (or $QTIDOCS_WORKER_SECRET) is required when -worker-url is set")
		flag.Usage()
		os.Exit(2)
	}
	if *workerURL != "" && *workerTeardownURL == "" {
		fmt.Fprintln(os.Stderr, "platform: -worker-teardown-url is required when -worker-url is set")
		flag.Usage()
		os.Exit(2)
	}

	mail := mailer.SMTP{
		Host:     *smtpHost,
		Port:     *smtpPort,
		Username: *smtpUsername,
		Password: *smtpPassword,
		From:     *smtpFrom,
	}
	if err := mail.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "platform: %v (set the -smtp-* flags or $QTIDOCS_SMTP_* variables)\n", err)
		flag.Usage()
		os.Exit(2)
	}

	var builder platform.Builder = platform.LogBuilder{}
	if *workerURL != "" {
		builder = platform.HTTPBuilder{URL: *workerURL, TeardownURL: *workerTeardownURL, Secret: *workerSecret}
	} else {
		fmt.Fprintln(os.Stderr, "platform: -worker-url not set; triggered builds and teardowns will only be logged, see internal/platform.LogBuilder")
	}

	store := storage.Open(*routing)
	mux := http.NewServeMux()
	mux.Handle("/internal/register", platform.RegisterHandler(store, builder, mail, *secret))
	mux.Handle("/internal/sweep", platform.SweepHandler(store, builder, *secret))

	fmt.Fprintf(os.Stderr, "platform: listening on %s, routing table at %s\n", *addr, *routing)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "platform: %v\n", err)
		os.Exit(1)
	}
}

func envInt(name string, fallback int) int {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "platform: $%s=%q is not a number\n", name, v)
		os.Exit(2)
	}
	return n
}
