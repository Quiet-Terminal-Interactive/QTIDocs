package registry

import "testing"

func TestParseReserved(t *testing.T) {
	cfg, err := ParseReserved([]byte("reserved:\n  - www\n  - api\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Words) != 2 || cfg.Words[0] != "www" || cfg.Words[1] != "api" {
		t.Errorf("Words = %v", cfg.Words)
	}
}

func TestParseReserved_InvalidYAML(t *testing.T) {
	_, err := ParseReserved([]byte("not: valid: yaml: ["))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIsReserved_ExplicitWord(t *testing.T) {
	cfg := ReservedConfig{Words: []string{"www", "api"}}
	if !cfg.IsReserved("www") {
		t.Error("expected www to be reserved")
	}
}

func TestIsReserved_CaseInsensitive(t *testing.T) {
	cfg := ReservedConfig{Words: []string{"www"}}
	if !cfg.IsReserved("WWW") {
		t.Error("expected WWW to be reserved (case-insensitive)")
	}
}

func TestIsReserved_ShortLabel(t *testing.T) {
	cfg := ReservedConfig{}
	if !cfg.IsReserved("a") {
		t.Error("expected single-letter subdomain to be reserved")
	}
	if !cfg.IsReserved("ab") {
		t.Error("expected two-letter subdomain to be reserved")
	}
	if cfg.IsReserved("abc") {
		t.Error("expected three-letter subdomain to not be reserved by length alone")
	}
}

func TestIsReserved_PRPreviewPattern(t *testing.T) {
	cfg := ReservedConfig{}
	cases := []string{"pr-1", "pr-42", "PR-99", "Pr-123"}
	for _, c := range cases {
		if !cfg.IsReserved(c) {
			t.Errorf("expected %q to be reserved (pr-<number> pattern)", c)
		}
	}
}

func TestIsReserved_NotPRPattern(t *testing.T) {
	cfg := ReservedConfig{}
	cases := []string{"prefix", "pr-abc", "preview"}
	for _, c := range cases {
		if cfg.IsReserved(c) {
			t.Errorf("expected %q to not match the pr-<number> pattern", c)
		}
	}
}

func TestIsReserved_NotReserved(t *testing.T) {
	cfg := ReservedConfig{Words: []string{"www"}}
	if cfg.IsReserved("acme") {
		t.Error("expected acme to not be reserved")
	}
}
