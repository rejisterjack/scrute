// Package config loads .scrute.yaml from the repository root.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = ".scrute.yaml"

// Config is the repository review configuration.
type Config struct {
	Provider     string   `yaml:"provider"`
	Model        string   `yaml:"model"`
	Strictness   string   `yaml:"strictness"`
	ContextLines int      `yaml:"context_lines"`
	MaxDiffBytes int      `yaml:"max_diff_bytes"`
	FailOn       string   `yaml:"fail_on"`
	Exclude      []string `yaml:"exclude"`
}

// Defaults is the configuration used when .scrute.yaml is absent.
func Defaults() Config {
	return Config{
		Provider:     "anthropic",
		Model:        "",
		Strictness:   "balanced",
		ContextLines: 40,
		MaxDiffBytes: 200000,
		FailOn:       "warning",
		Exclude: []string{
			".env",
			".env.*",
			"**/*.pem",
			"vendor/**",
			"node_modules/**",
			"dist/**",
		},
	}
}

// Template is the file written by `scrute init`.
func Template() string {
	return `# Scrute reviews staged changes with your own API key.
# https://github.com/rejisterjack/scrute

provider: anthropic
model: ""
strictness: balanced
context_lines: 40
max_diff_bytes: 200000
fail_on: warning
exclude:
  - ".env"
  - ".env.*"
  - "**/*.pem"
  - "vendor/**"
  - "node_modules/**"
  - "dist/**"
`
}

// Load reads .scrute.yaml from the work tree root. A missing file returns Defaults.
func Load(root string) (Config, error) {
	cfg := Defaults()
	path := filepath.Join(root, FileName)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// ApplyFlags overwrites configuration with set command flags.
func (c *Config) ApplyFlags(providerName, model, strictness string) error {
	if providerName != "" {
		c.Provider = providerName
	}
	if model != "" {
		c.Model = model
	}
	if strictness != "" {
		c.Strictness = strictness
	}
	c.normalize()
	return c.Validate()
}

func (c *Config) normalize() {
	c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
	c.Strictness = strings.ToLower(strings.TrimSpace(c.Strictness))
	c.FailOn = strings.ToLower(strings.TrimSpace(c.FailOn))
	c.Model = strings.TrimSpace(c.Model)
}

// Validate checks enumerations and limits.
func (c Config) Validate() error {
	switch c.Provider {
	case "anthropic", "openai":
	default:
		return fmt.Errorf("provider must be anthropic or openai, got %q", c.Provider)
	}
	switch c.Strictness {
	case "lenient", "balanced", "strict":
	default:
		return fmt.Errorf("strictness must be lenient, balanced, or strict, got %q", c.Strictness)
	}
	switch c.FailOn {
	case "none", "suggestion", "warning", "critical":
	default:
		return fmt.Errorf("fail_on must be none, suggestion, warning, or critical, got %q", c.FailOn)
	}
	if c.ContextLines < 0 {
		return fmt.Errorf("context_lines must be zero or greater")
	}
	if c.MaxDiffBytes <= 0 {
		return fmt.Errorf("max_diff_bytes must be greater than zero")
	}
	return nil
}

// Excluded reports whether path matches any exclusion pattern.
func Excluded(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if Match(pattern, path) {
			return true
		}
	}
	return false
}

// Match reports whether path matches a scrute exclusion pattern.
// Patterns use forward slashes. ** matches across directories.
func Match(pattern, path string) bool {
	pattern = strings.Trim(filepath.ToSlash(pattern), "/")
	path = strings.Trim(filepath.ToSlash(path), "/")
	if pattern == "" || path == "" {
		return false
	}
	if pattern == "**" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.HasPrefix(pattern, "**/") {
		rest := strings.TrimPrefix(pattern, "**/")
		if matchPath(rest, path) {
			return true
		}
		parts := strings.Split(path, "/")
		for i := range parts {
			if matchPath(rest, strings.Join(parts[i:], "/")) {
				return true
			}
		}
		return false
	}
	if matchPath(pattern, path) {
		return true
	}
	if !strings.Contains(pattern, "/") {
		return matchPath(pattern, pathBase(path))
	}
	return false
}

func matchPath(pattern, path string) bool {
	ok, err := filepath.Match(pattern, path)
	return err == nil && ok
}

func pathBase(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
