package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// centralAppDirName is the XDG application subdirectory that owns the machine-wide config
// file. It mirrors the store's application dir name so a desk's config and its store live
// under the same application namespace in their respective XDG homes.
const centralAppDirName = "deskkit"

// Central is the machine-wide config: $XDG_CONFIG_HOME/deskkit/config.yaml (falling back to
// ~/.config/deskkit/config.yaml when XDG_CONFIG_HOME is unset or empty). It sits BELOW the
// per-desk profile in the precedence chain (env > profile > central > default) and is the one
// place the LLM API key may be stored at rest, so the file is written 0600 in a 0700 dir.
//
// Identity-neutral: the file is created by the operator on their own machine; nothing about it
// ships in the binary but the path shape and the key names.
type Central struct {
	LLM struct {
		Provider string `yaml:"provider,omitempty"`
		Model    string `yaml:"model,omitempty"`
		APIKey   string `yaml:"api_key,omitempty"`
	} `yaml:"llm,omitempty"`
	DefaultDesk string `yaml:"default_desk,omitempty"`
}

// configHome resolves the XDG config home: $XDG_CONFIG_HOME, falling back to ~/.config when
// unset or empty (per the XDG base-dir convention).
func configHome() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: resolve home directory for the central config location: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return base, nil
}

// CentralPath resolves the central config file path. It creates nothing — a caller that wants
// the file on disk goes through SaveCentral.
func CentralPath() (string, error) {
	base, err := configHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, centralAppDirName, "config.yaml"), nil
}

// LoadCentral reads the central config. A MISSING file is not an error: it returns a
// zero-value Central and nil, so "no central config" and "empty central config" behave the
// same. An unreadable or malformed file DOES error — a silently ignored bad config is worse
// than a loud one.
func LoadCentral() (*Central, error) {
	path, err := CentralPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Central{}, nil
		}
		return nil, fmt.Errorf("config: read central config %s: %w", path, err)
	}
	var c Central
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("config: parse central config %s: %w", path, err)
	}
	return &c, nil
}

// SaveCentral writes the central config, creating the parent directory 0700 and the file 0600
// (it may hold an API key). An existing file's mode is tightened back to 0600 on every write.
func SaveCentral(c *Central) error {
	path, err := CentralPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: create central config dir: %w", err)
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: encode central config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("config: write central config %s: %w", path, err)
	}
	// WriteFile's perm applies only on create; tighten an existing widened file too.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("config: set central config mode %s: %w", path, err)
	}
	return nil
}

// CentralKeys lists the settable dotted keys, in display order.
func CentralKeys() []string {
	return []string{"llm.provider", "llm.model", "llm.api_key", "default_desk"}
}

// Set assigns a dotted key. An unknown key errors, naming the valid keys.
func (c *Central) Set(key, value string) error {
	switch key {
	case "llm.provider":
		c.LLM.Provider = value
	case "llm.model":
		c.LLM.Model = value
	case "llm.api_key":
		c.LLM.APIKey = value
	case "default_desk":
		c.DefaultDesk = value
	default:
		return fmt.Errorf("config: unknown key %q (valid keys: %s)", key, strings.Join(CentralKeys(), ", "))
	}
	return nil
}

// Get returns the value for a dotted key; ok is false for an unknown key.
func (c *Central) Get(key string) (string, bool) {
	switch key {
	case "llm.provider":
		return c.LLM.Provider, true
	case "llm.model":
		return c.LLM.Model, true
	case "llm.api_key":
		return c.LLM.APIKey, true
	case "default_desk":
		return c.DefaultDesk, true
	}
	return "", false
}

// APIKeyEnvName returns the NAME of the env var that would hold the LLM API key: the profile's
// secrets_ref.llm_api_key indirection (surfaced as LLMAPIKeyEnv) when set, else the default var
// for the resolved provider. Only the name is decided here; ResolveAPIKey reads the value.
//
// This mapping lives here, beside ResolveAPIKey, because BOTH the provider adapter and the
// config display surface need it. A second copy of this switch would drift silently: a display
// naming the wrong var is a config command that lies about where the key comes from.
func APIKeyEnvName(cfg *Config) string {
	if cfg.LLMAPIKeyEnv != "" {
		return cfg.LLMAPIKeyEnv
	}
	switch cfg.LLMProvider {
	case "openai":
		return "OPENAI_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	default:
		return "ANTHROPIC_API_KEY"
	}
}

// ResolveAPIKey resolves the LLM API key given the NAME of the env var that may hold it (the
// per-provider default, or the profile's secrets_ref.llm_api_key indirection target): the env
// var first, then the central config's llm.api_key. source is SourceEnv, SourceCentral, or ""
// when unresolved. An unreadable central config is treated as absent — Load reports that once
// at startup rather than every construction.
//
// This is the ONE place the key precedence lives: the provider adapter and any display surface
// both call it, so neither can drift from the other. A display surface must render the returned
// key through MaskSecret — never raw.
func ResolveAPIKey(envName string) (key, source string) {
	if v := os.Getenv(envName); v != "" {
		return v, SourceEnv
	}
	if c, err := LoadCentral(); err == nil && c.LLM.APIKey != "" {
		return c.LLM.APIKey, SourceCentral
	}
	return "", ""
}

// MaskSecret renders a secret for display. It NEVER returns the full secret: an empty value
// renders "(unset)", a value shorter than 8 characters renders "(set)" with no tail at all
// (too short for a 4-char tail to be non-identifying), and anything longer exposes only its
// last 4 characters.
func MaskSecret(s string) string {
	switch {
	case s == "":
		return "(unset)"
	case len(s) < 8:
		return "(set)"
	default:
		return "(set, …" + s[len(s)-4:] + ")"
	}
}
