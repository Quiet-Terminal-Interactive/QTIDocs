package dispute

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"
)

const ReminderWindow = 3 * 24 * time.Hour

const (
	ReminderSentLabel   = "deadline-reminder-sent"
	DeadlinePassedLabel = "deadline-passed"
)

var deadlineLine = regexp.MustCompile(`(?m)^Deadline:\s*(\d{4}-\d{2}-\d{2})\s*$`)

func DeadlineFromBody(body string) (deadline time.Time, ok bool) {
	m := deadlineLine.FindStringSubmatch(body)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

var reassignedLine = regexp.MustCompile(`(?m)^Reassigned:\s*(\d{4}-\d{2}-\d{2})\s*$`)

func ReassignedFromBody(body string) (reassigned time.Time, ok bool) {
	m := reassignedLine.FindStringSubmatch(body)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func ValidateInitialFiling(createdAt time.Time, body string) (want time.Time, ok bool) {
	created := time.Date(createdAt.Year(), createdAt.Month(), createdAt.Day(), 0, 0, 0, 0, time.UTC)
	want = created.AddDate(0, 0, 14)

	got, gotOK := DeadlineFromBody(body)
	return want, gotOK && got.Equal(want)
}

func ValidateAppealFiling(body string) (want time.Time, hasReassigned, ok bool) {
	reassigned, rOK := ReassignedFromBody(body)
	if !rOK {
		return time.Time{}, false, false
	}
	want = reassigned.AddDate(0, 0, 31)

	got, gotOK := DeadlineFromBody(body)
	return want, true, gotOK && got.Equal(want)
}

type Issue struct {
	Number int
	Body   string
	Labels []string
}

func (i Issue) hasLabel(label string) bool {
	for _, l := range i.Labels {
		if l == label {
			return true
		}
	}
	return false
}

type GitHub interface {
	ListOpenIssues(ctx context.Context, owner, repo, label string) ([]Issue, error)
	CommentOnIssue(ctx context.Context, owner, repo string, number int, body string) error
	AddLabel(ctx context.Context, owner, repo string, number int, label string) error
	CloseIssue(ctx context.Context, owner, repo string, number int, reason string) error
}

type Checker struct {
	GitHub GitHub
	Label  string
}

func (c Checker) label() string {
	if c.Label != "" {
		return c.Label
	}
	return "dispute"
}

type Result struct {
	Checked   int
	Reminded  int
	Escalated int
}

func (c Checker) Run(ctx context.Context, owner, repo string, now time.Time) (Result, error) {
	issues, err := c.GitHub.ListOpenIssues(ctx, owner, repo, c.label())
	if err != nil {
		return Result{}, fmt.Errorf("dispute: listing open %q issues: %w", c.label(), err)
	}

	var res Result
	for _, issue := range issues {
		res.Checked++

		deadline, ok := DeadlineFromBody(issue.Body)
		if !ok {
			continue
		}

		switch {
		case !now.Before(deadline):
			if issue.hasLabel(DeadlinePassedLabel) {
				continue
			}
			if err := c.GitHub.AddLabel(ctx, owner, repo, issue.Number, DeadlinePassedLabel); err != nil {
				log.Printf("dispute: labeling issue #%d %s: %v", issue.Number, DeadlinePassedLabel, err)
				continue
			}
			comment := fmt.Sprintf(
				"⏰ This dispute's deadline (%s) has passed. A Quiet-Terminal-Interactive org member needs to resolve it per DISPUTES.md, this is a flag for human review, not an automatic outcome.",
				deadline.Format("2006-01-02"),
			)
			if err := c.GitHub.CommentOnIssue(ctx, owner, repo, issue.Number, comment); err != nil {
				log.Printf("dispute: commenting on issue #%d: %v", issue.Number, err)
				continue
			}
			res.Escalated++

		case deadline.Sub(now) <= ReminderWindow:
			if issue.hasLabel(ReminderSentLabel) || issue.hasLabel(DeadlinePassedLabel) {
				continue
			}
			if err := c.GitHub.AddLabel(ctx, owner, repo, issue.Number, ReminderSentLabel); err != nil {
				log.Printf("dispute: labeling issue #%d %s: %v", issue.Number, ReminderSentLabel, err)
				continue
			}
			comment := fmt.Sprintf(
				"⏰ Reminder: this dispute's deadline is %s (%s from now).",
				deadline.Format("2006-01-02"), deadline.Sub(now).Round(time.Hour),
			)
			if err := c.GitHub.CommentOnIssue(ctx, owner, repo, issue.Number, comment); err != nil {
				log.Printf("dispute: commenting on issue #%d: %v", issue.Number, err)
				continue
			}
			res.Reminded++
		}
	}

	return res, nil
}
