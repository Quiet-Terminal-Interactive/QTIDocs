package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/analytics"
	"github.com/quiet-terminal-interactive/qtidocs/internal/server"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

func main() {
	root := flag.String("root", "", "sites root directory (one rendered site output per subdomain subdirectory); mutually exclusive with -routing")
	routing := flag.String("routing", "", "path to the routing table file (see internal/storage); mutually exclusive with -root")
	addr := flag.String("addr", ":8080", "listen address")
	analyticsDir := flag.String("analytics", "", "directory to store first-party page-view analytics under (see internal/analytics); omit to disable page-view collection and the public /_stats page")
	analyticsRetention := flag.Duration("analytics-retention", 30*24*time.Hour, "how long raw page-view events are kept before being purged; rolled-up daily aggregates are kept indefinitely regardless")
	flag.Parse()

	if (*root == "") == (*routing == "") {
		fmt.Fprintln(os.Stderr, "server: exactly one of -root or -routing is required")
		flag.Usage()
		os.Exit(2)
	}

	var resolver server.Resolver
	if *root != "" {
		resolver = server.FixtureResolver{Root: *root}
		fmt.Fprintf(os.Stderr, "server: listening on %s, serving fixture sites from %s\n", *addr, *root)
	} else {
		resolver = server.StorageResolver{Store: storage.Open(*routing)}
		fmt.Fprintf(os.Stderr, "server: listening on %s, serving sites from routing table %s\n", *addr, *routing)
	}

	srv := server.New(resolver)
	if *analyticsDir != "" {
		store := analytics.Open(*analyticsDir)
		srv.Analytics = store
		go runAnalyticsSweepLoop(store, *analyticsRetention)
		fmt.Fprintf(os.Stderr, "server: analytics enabled, storing under %s (retention %s)\n", *analyticsDir, *analyticsRetention)
	}

	if err := http.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

func runAnalyticsSweepLoop(store analytics.Store, retention time.Duration) {
	sweep := func() {
		if err := analytics.Sweep(store, retention, time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "server: analytics sweep failed: %v\n", err)
		}
	}

	sweep()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		sweep()
	}
}
