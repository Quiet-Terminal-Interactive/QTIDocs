package registry

import (
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry/registrytest"
)

func gitRepo(t *testing.T) string {
	return registrytest.InitRepo(t)
}

func gitCommitFile(t *testing.T, dir, path, content string) {
	registrytest.CommitFile(t, dir, path, content)
}

func gitRemoveFile(t *testing.T, dir, path string) {
	registrytest.RemoveFile(t, dir, path)
}

func chdir(t *testing.T, dir string) {
	registrytest.Chdir(t, dir)
}

func TestChangedFiles_AddedFile(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/base.yaml", "subdomain: base\n")
	chdir(t, dir)

	base := headSHA(t, dir)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\n")

	changes, err := ChangedFiles(base, "sites")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 || changes[0].Status != 'A' || changes[0].Path != "sites/acme.yaml" {
		t.Errorf("changes = %+v", changes)
	}
}

func TestChangedFiles_ModifiedFile(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\nbranch: main\n")
	chdir(t, dir)
	base := headSHA(t, dir)

	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\nbranch: develop\n")

	changes, err := ChangedFiles(base, "sites")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 || changes[0].Status != 'M' {
		t.Errorf("changes = %+v", changes)
	}
}

func TestChangedFiles_DeletedFile(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\n")
	chdir(t, dir)
	base := headSHA(t, dir)

	gitRemoveFile(t, dir, "sites/acme.yaml")

	changes, err := ChangedFiles(base, "sites")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 || changes[0].Status != 'D' {
		t.Errorf("changes = %+v", changes)
	}
}

func TestChangedFiles_NoChanges(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\n")
	chdir(t, dir)
	base := headSHA(t, dir)

	changes, err := ChangedFiles(base, "sites")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("changes = %+v, want none", changes)
	}
}

func TestChangedFiles_ScopedToDir(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\n")
	chdir(t, dir)
	base := headSHA(t, dir)

	gitCommitFile(t, dir, "README.md", "unrelated change")

	changes, err := ChangedFiles(base, "sites")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("changes = %+v, want none (change is outside sites/)", changes)
	}
}

func TestChangedFiles_InvalidBase(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\n")
	chdir(t, dir)

	_, err := ChangedFiles("not-a-real-ref", "sites")
	if err == nil {
		t.Fatal("expected error for invalid base ref, got nil")
	}
}

func TestEntryAtRef(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\nbranch: main\n")
	chdir(t, dir)
	base := headSHA(t, dir)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\nbranch: develop\n")

	e, err := EntryAtRef(base, "sites/acme.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Branch != "main" {
		t.Errorf("Branch = %q, want main (the version at base, not HEAD)", e.Branch)
	}
}

func TestEntryAtRef_NonexistentPath(t *testing.T) {
	dir := gitRepo(t)
	gitCommitFile(t, dir, "sites/acme.yaml", "subdomain: acme\n")
	chdir(t, dir)

	_, err := EntryAtRef("HEAD", "sites/ghost.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func headSHA(t *testing.T, dir string) string {
	return registrytest.HeadSHA(t, dir)
}
