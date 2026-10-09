package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry/registrytest"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := registrytest.InitRepo(t)
	if err := os.MkdirAll(filepath.Join(dir, "sites"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeAndCommit(t, dir, "reserved.yaml", "reserved:\n  - www\n")
	return dir
}

func writeAndCommit(t *testing.T, dir, path, content string) {
	registrytest.CommitFile(t, dir, path, content)
}

func removeAndCommit(t *testing.T, dir, path string) {
	registrytest.RemoveFile(t, dir, path)
}

func chdir(t *testing.T, dir string) {
	registrytest.Chdir(t, dir)
}

func headSHA(t *testing.T, dir string) string {
	return registrytest.HeadSHA(t, dir)
}

func validEntryYAML(subdomain string) string {
	return registrytest.ValidEntryYAML(subdomain)
}

func TestRun_DryRun_NewRegistrationPasses(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)

	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_DryRun_ReservedSubdomainFails(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)

	writeAndCommit(t, dir, "sites/www.yaml", validEntryYAML("www"))

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("err = %v, want reserved complaint", err)
	}
}

func TestRun_DryRun_EditByMaintainerPasses(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	base := headSHA(t, dir)

	writeAndCommit(t, dir, "sites/acme.yaml", "subdomain: acme\nrepo: github.com/acme/widgets\nbranch: develop\npath: qtidocs\ncontact: dev@example.com\nmaintainers:\n  - alice\n")

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_DryRun_EditByNonMaintainerFails(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	base := headSHA(t, dir)

	writeAndCommit(t, dir, "sites/acme.yaml", "subdomain: acme\nrepo: github.com/acme/widgets\nbranch: develop\npath: qtidocs\ncontact: dev@example.com\nmaintainers:\n  - alice\n")

	err := run("sites", "reserved.yaml", base, "mallory", "quiet-terminal-interactive", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "maintainers") {
		t.Errorf("err = %v, want maintainers complaint", err)
	}
}

func TestRun_DryRun_DeletionByMaintainerPasses(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	writeAndCommit(t, dir, "sites/other.yaml", validEntryYAML("other"))
	base := headSHA(t, dir)

	removeAndCommit(t, dir, "sites/acme.yaml")

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_DryRun_DeletionByNonMaintainerFails(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	writeAndCommit(t, dir, "sites/other.yaml", validEntryYAML("other"))
	base := headSHA(t, dir)

	removeAndCommit(t, dir, "sites/acme.yaml")

	err := run("sites", "reserved.yaml", base, "mallory", "quiet-terminal-interactive", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "maintainers") {
		t.Errorf("err = %v, want maintainers complaint", err)
	}
}

func TestRun_NoChangesFails(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	base := headSHA(t, dir)

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "no changed files") {
		t.Errorf("err = %v, want no-changed-files complaint", err)
	}
}

func TestRun_IgnoresUnrelatedFileChanges(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)
	writeAndCommit(t, dir, "README.md", "unrelated")

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "no changed files") {
		t.Errorf("err = %v, want no-changed-files complaint (README.md isn't under sites/)", err)
	}
}

func TestRun_MissingReservedFile(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	if err := os.Remove(filepath.Join(dir, "reserved.yaml")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	base := headSHA(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))

	err := run("sites", "reserved.yaml", base, "alice", "quiet-terminal-interactive", "", "", true)
	if err == nil {
		t.Fatal("expected error for missing reserved.yaml, got nil")
	}
}

func TestAlwaysAllow(t *testing.T) {
	var a alwaysAllow
	ok, err := a.HasWriteAccess(context.TODO(), "o", "r", "u")
	if err != nil || !ok {
		t.Errorf("HasWriteAccess = (%v, %v), want (true, nil)", ok, err)
	}
	ok, err = a.OrgMember(context.TODO(), "org", "u")
	if err != nil || ok {
		t.Errorf("OrgMember = (%v, %v), want (false, nil)", ok, err)
	}
	ok, err = a.PathExists(context.TODO(), "o", "r", "ref", "path")
	if err != nil || !ok {
		t.Errorf("PathExists = (%v, %v), want (true, nil)", ok, err)
	}
}
