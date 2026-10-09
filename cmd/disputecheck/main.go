package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/dispute"
)

func main() {
	owner := flag.String("owner", "quiet-terminal-interactive", "GitHub org/user that owns the registry repo")
	repo := flag.String("repo", "qtidocs", "registry repo name")
	label := flag.String("label", "dispute", "issue label marking a name-dispute issue")
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token with issues:write on the registry repo (defaults to $GITHUB_TOKEN)")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "disputecheck: -token (or $GITHUB_TOKEN) is required")
		flag.Usage()
		os.Exit(2)
	}

	c := dispute.Checker{
		GitHub: dispute.HTTPGitHub{
			Token:  *token,
			Client: &http.Client{Timeout: 30 * time.Second},
		},
		Label: *label,
	}

	res, err := c.Run(context.Background(), *owner, *repo, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "disputecheck: FAIL\n%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("disputecheck: PASS (checked %d, reminded %d, escalated %d)\n", res.Checked, res.Reminded, res.Escalated)
}
