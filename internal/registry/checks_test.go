package registry

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeGitHub struct {
	writeAccess map[string]bool
	orgMembers  map[string]bool
	paths       map[string]bool
	err         error
}

func newFakeGitHub() *fakeGitHub {
	return &fakeGitHub{
		writeAccess: map[string]bool{},
		orgMembers:  map[string]bool{},
		paths:       map[string]bool{},
	}
}

func (f *fakeGitHub) HasWriteAccess(ctx context.Context, owner, repo, user string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.writeAccess[owner+"/"+repo+"/"+user], nil
}
func (f *fakeGitHub) OrgMember(ctx context.Context, org, user string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.orgMembers[org+"/"+user], nil
}
func (f *fakeGitHub) PathExists(ctx context.Context, owner, repo, ref, path string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.paths[owner+"/"+repo+"/"+ref+"/"+path], nil
}

func baseRegInput() RegistrationInput {
	return RegistrationInput{
		Entry:    validEntry(),
		Author:   "alice",
		Existing: map[string]Entry{},
		Reserved: ReservedConfig{},
		QTIOrg:   "quiet-terminal-interactive",
	}
}

func TestCheckRegistration_Success(t *testing.T) {
	gh := newFakeGitHub()
	gh.writeAccess["acme/widgets/alice"] = true
	gh.paths["acme/widgets/main/qtidocs"] = true

	err := CheckRegistration(context.Background(), gh, baseRegInput())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckRegistration_SuccessViaOrgMembership(t *testing.T) {
	gh := newFakeGitHub()
	gh.orgMembers["quiet-terminal-interactive/alice"] = true
	gh.paths["acme/widgets/main/qtidocs"] = true

	err := CheckRegistration(context.Background(), gh, baseRegInput())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckRegistration_InvalidEntry(t *testing.T) {
	in := baseRegInput()
	in.Entry = Entry{}
	gh := newFakeGitHub()

	err := CheckRegistration(context.Background(), gh, in)
	if err == nil {
		t.Fatal("expected error for invalid entry, got nil")
	}
}

func TestCheckRegistration_DuplicateSubdomain(t *testing.T) {
	in := baseRegInput()
	in.Existing["acme"] = validEntry()
	gh := newFakeGitHub()
	gh.writeAccess["acme/widgets/alice"] = true
	gh.paths["acme/widgets/main/qtidocs"] = true

	err := CheckRegistration(context.Background(), gh, in)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Errorf("err = %v, want already-registered complaint", err)
	}
}

func TestCheckRegistration_ReservedSubdomain(t *testing.T) {
	in := baseRegInput()
	in.Reserved = ReservedConfig{Words: []string{"acme"}}
	gh := newFakeGitHub()
	gh.writeAccess["acme/widgets/alice"] = true
	gh.paths["acme/widgets/main/qtidocs"] = true

	err := CheckRegistration(context.Background(), gh, in)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("err = %v, want reserved complaint", err)
	}
}

func TestCheckRegistration_MissingQTIOrg(t *testing.T) {
	in := baseRegInput()
	in.QTIOrg = ""
	gh := newFakeGitHub()

	err := CheckRegistration(context.Background(), gh, in)
	if err == nil || !strings.Contains(err.Error(), "QTIOrg") {
		t.Errorf("err = %v, want QTIOrg complaint", err)
	}
}

func TestCheckRegistration_NoWriteAccessNoOrgMembership(t *testing.T) {
	gh := newFakeGitHub()
	gh.paths["acme/widgets/main/qtidocs"] = true

	err := CheckRegistration(context.Background(), gh, baseRegInput())
	if err == nil || !strings.Contains(err.Error(), "write access") {
		t.Errorf("err = %v, want write-access complaint", err)
	}
}

func TestCheckRegistration_PathDoesNotExist(t *testing.T) {
	gh := newFakeGitHub()
	gh.writeAccess["acme/widgets/alice"] = true

	err := CheckRegistration(context.Background(), gh, baseRegInput())
	if err == nil || !strings.Contains(err.Error(), "has no") {
		t.Errorf("err = %v, want path-missing complaint", err)
	}
}

func TestCheckRegistration_GitHubAPIError(t *testing.T) {
	gh := newFakeGitHub()
	gh.err = errors.New("rate limited")

	err := CheckRegistration(context.Background(), gh, baseRegInput())
	if err == nil {
		t.Fatal("expected error when GitHub API calls fail")
	}
}

func TestCheckRegistration_VersionPathMissing(t *testing.T) {
	in := baseRegInput()
	in.Entry.Versions = []Version{{Name: "v1", Ref: "v1-branch"}}
	gh := newFakeGitHub()
	gh.writeAccess["acme/widgets/alice"] = true
	gh.paths["acme/widgets/main/qtidocs"] = true

	err := CheckRegistration(context.Background(), gh, in)
	if err == nil || !strings.Contains(err.Error(), `version "v1"`) {
		t.Errorf("err = %v, want version-missing complaint", err)
	}
}

func TestCheckRegistration_AllErrorsJoined(t *testing.T) {
	in := baseRegInput()
	in.Entry = Entry{}
	in.QTIOrg = ""
	gh := newFakeGitHub()

	err := CheckRegistration(context.Background(), gh, in)
	msg := err.Error()
	if !strings.Contains(msg, "invalid entry") || !strings.Contains(msg, "QTIOrg") {
		t.Errorf("expected joined errors for both entry validation and QTIOrg, got: %s", msg)
	}
}

func TestCheckEdit_Success(t *testing.T) {
	existing := validEntry()
	in := EditInput{Existing: existing, New: existing, Author: "alice"}
	if err := CheckEdit(in); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckEdit_UnauthorizedAuthor(t *testing.T) {
	existing := validEntry()
	in := EditInput{Existing: existing, New: existing, Author: "mallory"}
	err := CheckEdit(in)
	if err == nil || !strings.Contains(err.Error(), "maintainers") {
		t.Errorf("err = %v, want maintainers complaint", err)
	}
}

func TestCheckEdit_AuthorCaseInsensitive(t *testing.T) {
	existing := validEntry()
	in := EditInput{Existing: existing, New: existing, Author: "ALICE"}
	if err := CheckEdit(in); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckEdit_SubdomainChanged(t *testing.T) {
	existing := validEntry()
	newEntry := existing
	newEntry.Subdomain = "different"
	in := EditInput{Existing: existing, New: newEntry, Author: "alice"}
	err := CheckEdit(in)
	if err == nil || !strings.Contains(err.Error(), "cannot be changed") {
		t.Errorf("err = %v, want subdomain-change complaint", err)
	}
}

func TestCheckEdit_InvalidNewEntry(t *testing.T) {
	existing := validEntry()
	newEntry := existing
	newEntry.Branch = ""
	in := EditInput{Existing: existing, New: newEntry, Author: "alice"}
	err := CheckEdit(in)
	if err == nil {
		t.Fatal("expected error for invalid new entry")
	}
}

func TestCheckUnregister_Success(t *testing.T) {
	in := UnregisterInput{Existing: validEntry(), Author: "alice"}
	if err := CheckUnregister(in); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckUnregister_AuthorCaseInsensitive(t *testing.T) {
	in := UnregisterInput{Existing: validEntry(), Author: "ALICE"}
	if err := CheckUnregister(in); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckUnregister_Unauthorized(t *testing.T) {
	in := UnregisterInput{Existing: validEntry(), Author: "mallory"}
	err := CheckUnregister(in)
	if err == nil || !strings.Contains(err.Error(), "maintainers") {
		t.Errorf("err = %v, want maintainers complaint", err)
	}
}
