// Package tools implements the profile module's four read-only tools over the desk's
// personalization surfaces: the `_knowledge/profile.{yaml,yml,json,md}` profile (surface i) and
// the `_knowledge/` background folder (surface ii). They are the Go home of the tool family that
// previously lived behind a separate stdio server, so a single binary now carries the whole
// surface; the profile grammar, discovery walk, and substitution rules are the SAME ones
// core/config already implements, never a second copy.
//
// Discovery start point (the one behavioural difference from a server launched inside the desk):
// the binary is routinely launched from an arbitrary working directory with the desk named by
// --dir / DESK_ROOT, so discovery starts at the RESOLVED DESK ROOT and only falls back to the
// process working directory when no desk root resolved. Every "no profile found" message names
// the directory actually searched, so it stays truthful under either path.
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	coreschema "github.com/hsb3/desk-standard/librarian/internal/core/schema"
)

// GetInput is profile_get's input: one dotted profile key.
type GetInput struct {
	Path string `json:"path" jsonschema:"description=Dotted profile key to resolve, e.g. \"repos.default\"."`
}

// GetOutput is profile_get's result.
type GetOutput struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

// ValidateInput is profile_validate's input: an optional explicit profile path.
type ValidateInput struct {
	Path string `json:"path,omitempty" jsonschema:"description=Optional explicit profile file path; omit to discover the desk's own profile."`
}

// ValidateOutput is profile_validate's result. ProfilePath is a pointer so an undiscovered
// profile serializes as an explicit null rather than an ambiguous empty string.
type ValidateOutput struct {
	Valid       bool     `json:"valid"`
	Errors      []string `json:"errors"`
	ProfilePath *string  `json:"profilePath"`
}

// RenderInput is template_render's input.
type RenderInput struct {
	Template string `json:"template" jsonschema:"description=Template text containing {{profile.<key>}} / {{env.<VAR>}} placeholders, each optionally carrying a || \"default\"."`
}

// RenderOutput is template_render's result.
type RenderOutput struct {
	Rendered string `json:"rendered"`
}

// IndexInput is knowledge_index's input. Budget is a POINTER so an explicit zero (index
// metadata only) stays distinguishable from an absent budget (use the default).
type IndexInput struct {
	Budget *int `json:"budget,omitempty" jsonschema:"description=Maximum total bytes of file content to include; omit for the 65536-byte default. Must not be negative."`
}

// ProfileGet resolves a dotted profile key to its scalar value. An absent-or-empty value is a
// hard ERROR, never a silent default: this tool exists so a caller learns what the desk does
// NOT declare. The failure names the keys available under the deepest parent that did resolve,
// which is what turns "not found" into a fixable message.
func ProfileGet(_ context.Context, _ core.App, cfg *config.Config, in *GetInput) (*GetOutput, error) {
	key := strings.TrimSpace(in.Path)
	if key == "" {
		return nil, fmt.Errorf("profile_get: path is required and must be a non-empty dotted key")
	}
	profile, path, err := loadDiscoveredProfile(cfg)
	if err != nil {
		return nil, fmt.Errorf("profile_get: %w", err)
	}
	value := config.ProfileScalar(profile, key)
	if value == "" {
		return nil, fmt.Errorf("profile_get: %s", absentKeyMessage(profile, path, key))
	}
	return &GetOutput{Path: key, Value: value}, nil
}

// ProfileValidate validates a profile against schema v1. Neither an undiscovered profile nor an
// unparseable one is a tool ERROR — both are RESULTS with valid:false, because "this desk has no
// usable profile" is exactly the answer a caller invokes this tool to get. Only a defect in the
// embedded schema itself (a build defect) is raised.
func ProfileValidate(_ context.Context, _ core.App, cfg *config.Config, in *ValidateInput) (*ValidateOutput, error) {
	start, startErr := discoveryStart(cfg)
	path := strings.TrimSpace(in.Path)
	if path == "" {
		if startErr != nil {
			return &ValidateOutput{Valid: false, Errors: []string{startErr.Error()}}, nil
		}
		found, ok := config.DiscoverProfile(start)
		if !ok {
			return &ValidateOutput{Valid: false, Errors: []string{noProfileMessage(start)}}, nil
		}
		path = found
	}
	profile, err := config.LoadProfile(path)
	if err != nil {
		return &ValidateOutput{Valid: false, Errors: []string{err.Error()}, ProfilePath: &path}, nil
	}
	res, err := coreschema.ValidateProfile(profile)
	if err != nil {
		return nil, fmt.Errorf("profile_validate: %w", err)
	}
	return &ValidateOutput{Valid: res.Valid, Errors: res.Errors, ProfilePath: &path}, nil
}

// TemplateRender substitutes {{profile.…}} / {{env.…}} placeholders in a template. A MISSING
// profile is not an error here — the template renders against an empty profile, so a template
// built entirely from env vars and defaults still works on an unpersonalized desk. An
// unresolved placeholder that carries no default IS an error (fail loud, never a silent empty
// substitution), raised by the shared substituter.
func TemplateRender(_ context.Context, _ core.App, cfg *config.Config, in *RenderInput) (*RenderOutput, error) {
	profile := map[string]any{}
	if start, err := discoveryStart(cfg); err == nil {
		if path, ok := config.DiscoverProfile(start); ok {
			if loaded, lerr := config.LoadProfile(path); lerr == nil {
				profile = loaded
			}
		}
	}
	rendered, err := config.Substitute(in.Template, profile)
	if err != nil {
		return nil, fmt.Errorf("template_render: %w", err)
	}
	return &RenderOutput{Rendered: rendered}, nil
}

// KnowledgeIndexTool indexes the desk's background folder. The root is the directory HOLDING
// the discovered profile (the personalization root itself); with no profile it falls back to
// the desk's personalization root directly, so a desk carrying background prose but no profile
// file is still indexed. When nothing resolves the result is an EMPTY index rooted at "" rather
// than an error — an unpersonalized desk has no knowledge, which is an answer, not a failure.
func KnowledgeIndexTool(_ context.Context, _ core.App, cfg *config.Config, in *IndexInput) (*KnowledgeIndexResult, error) {
	budget := DefaultKnowledgeBudget
	if in.Budget != nil && *in.Budget >= 0 {
		budget = *in.Budget
	}
	root := knowledgeRoot(cfg)
	if root == "" {
		return &KnowledgeIndexResult{Root: "", Budget: budget, Entries: []KnowledgeEntry{}}, nil
	}
	idx := KnowledgeIndex(root, budget)
	return &idx, nil
}

// knowledgeRoot resolves the directory to index, or "" if nothing resolves.
func knowledgeRoot(cfg *config.Config) string {
	start, err := discoveryStart(cfg)
	if err != nil {
		return ""
	}
	if path, ok := config.DiscoverProfile(start); ok {
		return filepath.Dir(path)
	}
	fallback := filepath.Join(start, config.ProfileRootDir)
	if fi, statErr := os.Stat(fallback); statErr == nil && fi.IsDir() {
		return fallback
	}
	return ""
}

// discoveryStart returns the directory the profile walk-up starts from: the resolved desk root
// when there is one, else the process working directory. The desk root comes first because the
// binary is normally launched from an ARBITRARY working directory with the desk named by
// --dir / DESK_ROOT — a cwd-only walk-up would find nothing in exactly the cases that matter.
func discoveryStart(cfg *config.Config) (string, error) {
	if cfg != nil && cfg.DeskRoot != "" {
		return cfg.DeskRoot, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("no desk root is resolved and the working directory is unreadable: %w", err)
	}
	return wd, nil
}

// loadDiscoveredProfile discovers and parses the desk's profile, returning it with the path it
// came from.
func loadDiscoveredProfile(cfg *config.Config) (map[string]any, string, error) {
	start, err := discoveryStart(cfg)
	if err != nil {
		return nil, "", err
	}
	path, ok := config.DiscoverProfile(start)
	if !ok {
		return nil, "", fmt.Errorf("%s", noProfileMessage(start))
	}
	profile, err := config.LoadProfile(path)
	if err != nil {
		return nil, path, err
	}
	return profile, path, nil
}

// noProfileMessage names the directory the walk-up actually started from, so the message stays
// truthful whether discovery ran from the desk root or from the working directory.
func noProfileMessage(start string) string {
	return fmt.Sprintf("no %s/profile.{yaml,yml,json,md} found searching up from %s",
		config.ProfileRootDir, start)
}

// absentKeyMessage builds the fail-loud message for a key that resolved to nothing, listing the
// keys available under the DEEPEST parent that did resolve so the caller can correct the path
// in one step instead of probing.
func absentKeyMessage(profile map[string]any, profilePath, dotted string) string {
	parts := strings.Split(dotted, ".")
	var reached []string
	container := profile
	for _, p := range parts {
		next, ok := config.IndexTree(container, []string{p}).(map[string]any)
		if !ok {
			break
		}
		reached = append(reached, p)
		container = next
	}

	under := "the top level"
	if len(reached) > 0 {
		under = strconv.Quote(strings.Join(reached, "."))
	}
	keys := make([]string, 0, len(container))
	for k := range container {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	msg := fmt.Sprintf("key %q is absent or empty in %s", dotted, profilePath)
	if len(keys) > 0 {
		msg += fmt.Sprintf("; keys available under %s: %s", under, strings.Join(keys, ", "))
	}
	return msg
}
