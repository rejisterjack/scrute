package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{".env", ".env", true},
		{".env", "app/.env", true},
		{".env", ".env.local", false},
		{".env.*", ".env.local", true},
		{".env.*", "secrets/.env.production", true},
		{"**/*.pem", "keys/server.pem", true},
		{"**/*.pem", "server.pem", true},
		{"**/*.pem", "server.pem.bak", false},
		{"vendor/**", "vendor/mod.go", true},
		{"vendor/**", "vendor", true},
		{"vendor/**", "internal/vendor/mod.go", false},
		{"node_modules/**", "node_modules/leftpad/index.js", true},
		{"dist/**", "dist/app.js", true},
		{"dist/**", "src/dist.go", false},
	}
	for _, tc := range cases {
		if got := Match(tc.pattern, tc.path); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestLoadMissingUsesDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	def := Defaults()
	if cfg.Provider != def.Provider || cfg.FailOn != def.FailOn || len(cfg.Exclude) != len(def.Exclude) {
		t.Fatalf("%+v", cfg)
	}
}

func TestLoadReplacesExclude(t *testing.T) {
	dir := t.TempDir()
	body := "provider: openai\nstrictness: strict\nfail_on: none\nexclude:\n  - \"*.lock\"\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "openai" || cfg.Strictness != "strict" || cfg.FailOn != "none" {
		t.Fatalf("%+v", cfg)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "*.lock" {
		t.Fatalf("exclude %#v", cfg.Exclude)
	}
	if cfg.MaxDiffBytes != Defaults().MaxDiffBytes {
		t.Fatalf("max diff %d", cfg.MaxDiffBytes)
	}
}

func TestTemplateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(Template()), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	def := Defaults()
	if cfg.Provider != def.Provider || cfg.Strictness != def.Strictness || cfg.FailOn != def.FailOn || cfg.ContextLines != def.ContextLines || cfg.MaxDiffBytes != def.MaxDiffBytes {
		t.Fatalf("%+v", cfg)
	}
	if len(cfg.Exclude) != len(def.Exclude) {
		t.Fatalf("exclude %#v", cfg.Exclude)
	}
}

func TestRejectsUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("provider: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error")
	}
}
