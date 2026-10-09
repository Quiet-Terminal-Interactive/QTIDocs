package dispute

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDeadlineFromBody_Valid(t *testing.T) {
	body := "Some text\nDeadline: 2026-01-15\nMore text\n"
	got, ok := DeadlineFromBody(body)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDeadlineFromBody_Missing(t *testing.T) {
	_, ok := DeadlineFromBody("No deadline line here.")
	if ok {
		t.Error("expected ok=false")
	}
}

func TestDeadlineFromBody_InvalidDate(t *testing.T) {
	_, ok := DeadlineFromBody("Deadline: not-a-date")
	if ok {
		t.Error("expected ok=false for invalid date")
	}
}

func TestDeadlineFromBody_RequiresLineAnchors(t *testing.T) {
	_, ok := DeadlineFromBody("The Deadline: 2026-01-15 is soon")
	if ok {
		t.Error("expected ok=false when Deadline: isn't at line start")
	}
}

func TestDeadlineFromBody_TrailingWhitespace(t *testing.T) {
	got, ok := DeadlineFromBody("Deadline: 2026-01-15   \n")
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestValidateInitialFiling_Matches(t *testing.T) {
	created := time.Date(2026, 1, 1, 15, 30, 0, 0, time.UTC)
	body := "Deadline: 2026-01-15\n"
	want, ok := ValidateInitialFiling(created, body)
	if !ok {
		t.Error("expected ok=true for correctly computed deadline")
	}
	wantDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !want.Equal(wantDate) {
		t.Errorf("want = %v, want %v", want, wantDate)
	}
}

func TestValidateInitialFiling_Mismatch(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	body := "Deadline: 2026-02-01\n"
	_, ok := ValidateInitialFiling(created, body)
	if ok {
		t.Error("expected ok=false for mismatched deadline")
	}
}

func TestValidateInitialFiling_MissingDeadline(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, ok := ValidateInitialFiling(created, "no deadline line")
	if ok {
		t.Error("expected ok=false when body has no deadline")
	}
}

func TestValidateAppealFiling(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantHasReass bool
		wantOK       bool
		wantWant     string
	}{
		{
			name:         "exactly 31 days after reassignment",
			body:         "Reassigned: 2026-09-01\n\nDeadline: 2026-10-02",
			wantHasReass: true,
			wantOK:       true,
			wantWant:     "2026-10-02",
		},
		{
			name:         "one day short",
			body:         "Reassigned: 2026-09-01\n\nDeadline: 2026-10-01",
			wantHasReass: true,
			wantOK:       false,
			wantWant:     "2026-10-02",
		},
		{
			name:         "missing reassigned line",
			body:         "Deadline: 2026-10-02",
			wantHasReass: false,
			wantOK:       false,
			wantWant:     "",
		},
		{
			name:         "missing deadline line",
			body:         "Reassigned: 2026-09-01",
			wantHasReass: true,
			wantOK:       false,
			wantWant:     "2026-10-02",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want, hasReass, ok := ValidateAppealFiling(tc.body)
			if hasReass != tc.wantHasReass {
				t.Fatalf("hasReassigned = %v, want %v", hasReass, tc.wantHasReass)
			}
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if tc.wantWant != "" && want.Format("2006-01-02") != tc.wantWant {
				t.Fatalf("want = %v, want %v", want.Format("2006-01-02"), tc.wantWant)
			}
		})
	}
}

type fakeGitHub struct {
	issues      []Issue
	comments    map[int][]string
	labelsAdded map[int][]string
	closed      map[int]string
	listErr     error
	commentErr  map[int]error
	addLabelErr map[int]error
}

func newFakeGitHub(issues []Issue) *fakeGitHub {
	return &fakeGitHub{
		issues:      issues,
		comments:    map[int][]string{},
		labelsAdded: map[int][]string{},
		closed:      map[int]string{},
	}
}

func (f *fakeGitHub) ListOpenIssues(ctx context.Context, owner, repo, label string) ([]Issue, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.issues, nil
}

func (f *fakeGitHub) CommentOnIssue(ctx context.Context, owner, repo string, number int, body string) error {
	if err := f.commentErr[number]; err != nil {
		return err
	}
	f.comments[number] = append(f.comments[number], body)
	return nil
}

func (f *fakeGitHub) AddLabel(ctx context.Context, owner, repo string, number int, label string) error {
	if err := f.addLabelErr[number]; err != nil {
		return err
	}
	f.labelsAdded[number] = append(f.labelsAdded[number], label)
	return nil
}

func (f *fakeGitHub) CloseIssue(ctx context.Context, owner, repo string, number int, reason string) error {
	f.closed[number] = reason
	return nil
}

func TestChecker_Run_RemindsApproachingDeadline(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 1, Body: "Deadline: 2026-01-02\n"}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checked != 1 || res.Reminded != 1 || res.Escalated != 0 {
		t.Errorf("res = %+v, want Checked=1 Reminded=1 Escalated=0", res)
	}
	if len(gh.comments[1]) != 1 {
		t.Errorf("comments = %v, want 1 comment", gh.comments[1])
	}
	if len(gh.labelsAdded[1]) != 1 || gh.labelsAdded[1][0] != ReminderSentLabel {
		t.Errorf("labelsAdded = %v, want [%s]", gh.labelsAdded[1], ReminderSentLabel)
	}
}

func TestChecker_Run_EscalatesPassedDeadline(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 2, Body: "Deadline: 2026-01-05\n"}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Escalated != 1 || res.Reminded != 0 {
		t.Errorf("res = %+v, want Escalated=1 Reminded=0", res)
	}
	if len(gh.labelsAdded[2]) != 1 || gh.labelsAdded[2][0] != DeadlinePassedLabel {
		t.Errorf("labelsAdded = %v, want [%s]", gh.labelsAdded[2], DeadlinePassedLabel)
	}
}

func TestChecker_Run_SkipsAlreadyReminded(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 3, Body: "Deadline: 2026-01-02\n", Labels: []string{ReminderSentLabel}}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reminded != 0 {
		t.Errorf("res.Reminded = %d, want 0 (already reminded)", res.Reminded)
	}
	if len(gh.comments[3]) != 0 {
		t.Errorf("comments = %v, want none", gh.comments[3])
	}
}

func TestChecker_Run_SkipsAlreadyEscalated(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 4, Body: "Deadline: 2026-01-05\n", Labels: []string{DeadlinePassedLabel}}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Escalated != 0 {
		t.Errorf("res.Escalated = %d, want 0 (already escalated)", res.Escalated)
	}
}

func TestChecker_Run_PassedDeadlineIgnoresReminderSentLabel(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 5, Body: "Deadline: 2026-01-05\n", Labels: []string{ReminderSentLabel}}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Escalated != 1 {
		t.Errorf("res.Escalated = %d, want 1", res.Escalated)
	}
}

func TestChecker_Run_SkipsMalformedBody(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 6, Body: "no deadline here"}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checked != 1 || res.Reminded != 0 || res.Escalated != 0 {
		t.Errorf("res = %+v, want only Checked=1", res)
	}
}

func TestChecker_Run_NotYetInReminderWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	issue := Issue{Number: 7, Body: "Deadline: 2026-02-01\n"}
	gh := newFakeGitHub([]Issue{issue})
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reminded != 0 || res.Escalated != 0 {
		t.Errorf("res = %+v, want no action", res)
	}
}

func TestChecker_Run_ListIssuesError(t *testing.T) {
	gh := newFakeGitHub(nil)
	gh.listErr = errors.New("boom")
	c := Checker{GitHub: gh}

	_, err := c.Run(context.Background(), "owner", "repo", time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChecker_Run_ContinuesAfterCommentError(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	issues := []Issue{
		{Number: 1, Body: "Deadline: 2026-01-05\n"},
		{Number: 2, Body: "Deadline: 2026-01-05\n"},
	}
	gh := newFakeGitHub(issues)
	gh.commentErr = map[int]error{1: errors.New("boom")}
	c := Checker{GitHub: gh}

	res, err := c.Run(context.Background(), "owner", "repo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Escalated != 1 {
		t.Errorf("res.Escalated = %d, want 1 (issue 1 should be skipped, issue 2 should succeed)", res.Escalated)
	}
}

func TestChecker_Label_DefaultsToDispute(t *testing.T) {
	c := Checker{}
	if got := c.label(); got != "dispute" {
		t.Errorf("label() = %q, want %q", got, "dispute")
	}
}

func TestChecker_Label_UsesOverride(t *testing.T) {
	c := Checker{Label: "custom"}
	if got := c.label(); got != "custom" {
		t.Errorf("label() = %q, want %q", got, "custom")
	}
}

func TestIssue_HasLabel(t *testing.T) {
	i := Issue{Labels: []string{"a", "b"}}
	if !i.hasLabel("a") {
		t.Error("expected hasLabel(a) = true")
	}
	if i.hasLabel("c") {
		t.Error("expected hasLabel(c) = false")
	}
}
