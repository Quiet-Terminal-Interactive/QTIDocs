package main

import (
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
)

func TestVersionFlags_Set_Valid(t *testing.T) {
	var v versionFlags
	if err := v.Set("v1=main"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := v.Set("v2=release-branch"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := versionFlags{{Name: "v1", Ref: "main"}, {Name: "v2", Ref: "release-branch"}}
	if len(v) != len(want) {
		t.Fatalf("v = %+v, want %+v", v, want)
	}
	for i := range want {
		if v[i] != want[i] {
			t.Errorf("v[%d] = %+v, want %+v", i, v[i], want[i])
		}
	}
}

func TestVersionFlags_Set_MissingEquals(t *testing.T) {
	var v versionFlags
	if err := v.Set("v1-no-equals"); err == nil {
		t.Error("expected error for missing '=', got nil")
	}
}

func TestVersionFlags_Set_EmptyName(t *testing.T) {
	var v versionFlags
	if err := v.Set("=main"); err == nil {
		t.Error("expected error for empty name, got nil")
	}
}

func TestVersionFlags_Set_EmptyRef(t *testing.T) {
	var v versionFlags
	if err := v.Set("v1="); err == nil {
		t.Error("expected error for empty ref, got nil")
	}
}

func TestVersionFlags_String(t *testing.T) {
	var v versionFlags
	if v.String() != "" {
		t.Errorf("String() = %q, want empty", v.String())
	}
}

func TestVersionFlags_RefWithEqualsSign(t *testing.T) {
	var v versionFlags
	if err := v.Set("v1=feature=branch"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := build.Version{Name: "v1", Ref: "feature=branch"}
	if v[0] != want {
		t.Errorf("v[0] = %+v, want %+v", v[0], want)
	}
}
