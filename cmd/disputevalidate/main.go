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
	number := flag.Int("number", 0, "issue number to validate (required)")
	created := flag.String("created", "", "issue's created_at timestamp, RFC3339 (required for -kind initial)")
	kind := flag.String("kind", "initial", `which template this issue came from: "initial" or "appeal"`)
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token with issues:write on the registry repo (defaults to $GITHUB_TOKEN)")
	flag.Parse()

	if *number == 0 {
		fmt.Fprintln(os.Stderr, "disputevalidate: -number is required")
		flag.Usage()
		os.Exit(2)
	}
	if *kind != "initial" && *kind != "appeal" {
		fmt.Fprintf(os.Stderr, "disputevalidate: -kind must be \"initial\" or \"appeal\", got %q\n", *kind)
		flag.Usage()
		os.Exit(2)
	}
	if *kind == "initial" && *created == "" {
		fmt.Fprintln(os.Stderr, "disputevalidate: -created is required for -kind initial")
		flag.Usage()
		os.Exit(2)
	}
	if *token == "" {
		fmt.Fprintln(os.Stderr, "disputevalidate: -token (or $GITHUB_TOKEN) is required")
		flag.Usage()
		os.Exit(2)
	}

	body := os.Getenv("ISSUE_BODY")

	var (
		ok      bool
		comment string
	)
	switch *kind {
	case "initial":
		createdAt, err := time.Parse(time.RFC3339, *created)
		if err != nil {
			fmt.Fprintf(os.Stderr, "disputevalidate: parsing -created: %v\n", err)
			os.Exit(2)
		}
		var want time.Time
		want, ok = dispute.ValidateInitialFiling(createdAt, body)
		if !ok {
			comment = fmt.Sprintf(
				"This dispute's `Deadline:` line isn't exactly 14 calendar days from today — it must be **%s**. Please open it again with the deadline being exactly 14 calendar days from today.",
				want.Format("2006-01-02"),
			)
		}

	case "appeal":
		var (
			want          time.Time
			hasReassigned bool
		)
		want, hasReassigned, ok = dispute.ValidateAppealFiling(body)
		switch {
		case ok:
			// nothing to do
		case !hasReassigned:
			comment = "This appeal is missing a valid `Reassigned: YYYY-MM-DD` line (the date the subdomain was actually reassigned away from you). Please open it again with that line filled in, and the deadline being exactly 31 calendar days after it."
		default:
			comment = fmt.Sprintf(
				"This appeal's `Deadline:` line isn't exactly 31 calendar days after the reassignment date — it must be **%s**. Please open it again with the deadline being exactly 31 calendar days after the reassignment date.",
				want.Format("2006-01-02"),
			)
		}
	}

	if ok {
		fmt.Printf("disputevalidate: PASS (%s deadline is correct)\n", *kind)
		return
	}

	gh := dispute.HTTPGitHub{
		Token:  *token,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
	ctx := context.Background()

	if err := gh.CommentOnIssue(ctx, *owner, *repo, *number, comment); err != nil {
		fmt.Fprintf(os.Stderr, "disputevalidate: FAIL\n%v\n", err)
		os.Exit(1)
	}
	if err := gh.CloseIssue(ctx, *owner, *repo, *number, "not_planned"); err != nil {
		fmt.Fprintf(os.Stderr, "disputevalidate: FAIL\n%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("disputevalidate: closed #%d (%s deadline was wrong or missing)\n", *number, *kind)
}
