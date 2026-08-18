package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hsb3/deskkit/internal/core/config"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// scratchDesk builds a throwaway desk with a _knowledge/profile.yaml holding body, and returns
// a config resolved to it — the SAME shape both real callers use (a desk root resolved from
// --dir/DESK_ROOT, never the process working directory).
func scratchDesk(t *testing.T, body string) (*config.Config, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, config.ProfileRootDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir knowledge root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	return &config.Config{DeskRoot: root, DeskName: "scratch"}, root
}

const validProfile = `schema_version: 1
desk:
  name: scratch
  root: "."
repos:
  default: owner/repo
identity:
  github:
    personal: handle
`

// TestProfileGet_ResolvesFromDeskRoot: discovery starts at the RESOLVED DESK ROOT, not at the
// process working directory — the binary is launched from an arbitrary cwd with --dir/DESK_ROOT.
func TestProfileGet_ResolvesFromDeskRoot(t *testing.T) {
	cfg, _ := scratchDesk(t, validProfile)
	out, err := ProfileGet(context.Background(), nil, cfg, &GetInput{Path: "  identity.github.personal  "})
	if err != nil {
		t.Fatalf("ProfileGet: %v", err)
	}
	if out.Path != "identity.github.personal" || out.Value != "handle" {
		t.Fatalf("got %+v, want the trimmed path and its scalar value", out)
	}
}

func TestProfileGet_Errors(t *testing.T) {
	cfg, _ := scratchDesk(t, validProfile)

	if _, err := ProfileGet(context.Background(), nil, cfg, &GetInput{Path: "   "}); err == nil {
		t.Error("an empty path must be a hard error")
	}

	// An absent key fails loud AND lists the keys available under the deepest resolved parent.
	_, err := ProfileGet(context.Background(), nil, cfg, &GetInput{Path: "identity.github.missing"})
	if err == nil {
		t.Fatal("an absent key must be a hard error, never an empty default")
	}
	if !strings.Contains(err.Error(), "identity.github") || !strings.Contains(err.Error(), "personal") {
		t.Errorf("error should name the deepest resolved parent and its available keys, got %q", err)
	}

	// A key that resolves to a MAP is not a scalar and is equally absent.
	if _, err := ProfileGet(context.Background(), nil, cfg, &GetInput{Path: "identity"}); err == nil {
		t.Error("a non-scalar key must be a hard error")
	}

	// No profile anywhere: the message names the directory actually searched.
	empty := &config.Config{DeskRoot: t.TempDir(), DeskName: "bare"}
	_, err = ProfileGet(context.Background(), nil, empty, &GetInput{Path: "desk.name"})
	if err == nil || !strings.Contains(err.Error(), empty.DeskRoot) {
		t.Errorf("a missing profile should name the searched directory %q, got %v", empty.DeskRoot, err)
	}
}

func TestProfileValidate(t *testing.T) {
	cfg, root := scratchDesk(t, validProfile)
	out, err := ProfileValidate(context.Background(), nil, cfg, &ValidateInput{})
	if err != nil {
		t.Fatalf("ProfileValidate: %v", err)
	}
	if !out.Valid || len(out.Errors) != 0 {
		t.Fatalf("a schema-clean profile should validate, got %+v", out)
	}
	if out.ProfilePath == nil || *out.ProfilePath != filepath.Join(root, config.ProfileRootDir, "profile.yaml") {
		t.Errorf("ProfilePath = %v, want the discovered profile path", out.ProfilePath)
	}

	// A schema violation is reported in errors, not raised.
	bad, _ := scratchDesk(t, "schema_version: 1\nnot_a_field: x\n")
	out, err = ProfileValidate(context.Background(), nil, bad, &ValidateInput{})
	if err != nil {
		t.Fatalf("ProfileValidate on an invalid profile: %v", err)
	}
	if out.Valid || len(out.Errors) == 0 || !strings.Contains(strings.Join(out.Errors, " "), "not_a_field") {
		t.Fatalf("an invalid profile should report the offending key, got %+v", out)
	}
}

// TestProfileValidate_NoProfileReturnsInvalid: absence RETURNS valid:false, it does not error —
// the tool is how a caller finds out a desk is unpersonalized.
func TestProfileValidate_NoProfileReturnsInvalid(t *testing.T) {
	bare := &config.Config{DeskRoot: t.TempDir(), DeskName: "bare"}
	out, err := ProfileValidate(context.Background(), nil, bare, &ValidateInput{})
	if err != nil {
		t.Fatalf("a missing profile must not be a tool error, got %v", err)
	}
	if out.Valid || len(out.Errors) == 0 || out.ProfilePath != nil {
		t.Fatalf("got %+v, want valid:false with a reason and a null profilePath", out)
	}
	if !strings.Contains(out.Errors[0], bare.DeskRoot) {
		t.Errorf("the reason should name the searched directory, got %q", out.Errors[0])
	}
	if !strings.Contains(mustJSON(t, out), `"profilePath":null`) {
		t.Errorf("profilePath must serialize as null, got %s", mustJSON(t, out))
	}
}

// TestProfileValidate_ExplicitPath: an explicit path bypasses discovery; an unparseable file is
// a returned result, not an error.
func TestProfileValidate_ExplicitPath(t *testing.T) {
	cfg, _ := scratchDesk(t, validProfile)
	other := filepath.Join(t.TempDir(), "elsewhere.yaml")
	if err := os.WriteFile(other, []byte("schema_version: 1\nbogus: y\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := ProfileValidate(context.Background(), nil, cfg, &ValidateInput{Path: other})
	if err != nil {
		t.Fatalf("ProfileValidate: %v", err)
	}
	if out.Valid || out.ProfilePath == nil || *out.ProfilePath != other {
		t.Fatalf("got %+v, want the explicit path validated (and invalid)", out)
	}

	unparseable := filepath.Join(t.TempDir(), "broken.yaml")
	if err := os.WriteFile(unparseable, []byte("\tnot: [valid\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err = ProfileValidate(context.Background(), nil, cfg, &ValidateInput{Path: unparseable})
	if err != nil {
		t.Fatalf("a parse failure must be a result, not a tool error: %v", err)
	}
	if out.Valid || len(out.Errors) == 0 {
		t.Fatalf("an unparseable profile should be invalid with a reason, got %+v", out)
	}
}

func TestTemplateRender(t *testing.T) {
	cfg, _ := scratchDesk(t, validProfile)
	t.Setenv("DESKKIT_TEST_TEMPLATE_VAR", "from-env")

	out, err := TemplateRender(context.Background(), nil, cfg, &RenderInput{
		Template: `desk={{profile.desk.name}} env={{env.DESKKIT_TEST_TEMPLATE_VAR}} opt={{profile.nope || "fallback"}}`,
	})
	if err != nil {
		t.Fatalf("TemplateRender: %v", err)
	}
	if out.Rendered != "desk=scratch env=from-env opt=fallback" {
		t.Fatalf("Rendered = %q", out.Rendered)
	}

	// A required placeholder with no default and no value is a hard error.
	if _, err := TemplateRender(context.Background(), nil, cfg, &RenderInput{Template: "{{profile.nope}}"}); err == nil {
		t.Error("an unresolved required placeholder must be a tool error")
	}

	// A MISSING profile is not an error here — it renders against an empty profile.
	bare := &config.Config{DeskRoot: t.TempDir(), DeskName: "bare"}
	out, err = TemplateRender(context.Background(), nil, bare, &RenderInput{Template: `x={{profile.desk.name || "none"}}`})
	if err != nil {
		t.Fatalf("a missing profile must not fail template_render: %v", err)
	}
	if out.Rendered != "x=none" {
		t.Errorf("Rendered = %q, want the default applied against an empty profile", out.Rendered)
	}
}

// TestKnowledgeIndexTool_RootIsProfileDir: the index root is the directory HOLDING the
// discovered profile (the personalization root), resolved from the desk root.
func TestKnowledgeIndexTool_RootIsProfileDir(t *testing.T) {
	cfg, root := scratchDesk(t, validProfile)
	knowledge := filepath.Join(root, config.ProfileRootDir)
	write(t, knowledge, "background.md", "one two three")

	idx, err := KnowledgeIndexTool(context.Background(), nil, cfg, &IndexInput{})
	if err != nil {
		t.Fatalf("KnowledgeIndexTool: %v", err)
	}
	if idx.Root != knowledge {
		t.Fatalf("Root = %q, want the profile's own directory %q", idx.Root, knowledge)
	}
	if idx.Budget != DefaultKnowledgeBudget {
		t.Errorf("an absent budget should fall back to the default, got %d", idx.Budget)
	}
	if idx.FileCount != 1 || idx.Entries[0].Path != "background.md" {
		t.Fatalf("entries = %+v, want just background.md (profile.yaml excluded)", idx.Entries)
	}

	// An explicit budget is honoured, including an explicit zero (include nothing).
	zero := 0
	idx, err = KnowledgeIndexTool(context.Background(), nil, cfg, &IndexInput{Budget: &zero})
	if err != nil {
		t.Fatalf("KnowledgeIndexTool: %v", err)
	}
	if idx.Budget != 0 || idx.BytesIncluded != 0 || idx.Entries[0].ContentIncluded {
		t.Fatalf("budget 0 should index metadata only, got %+v", idx)
	}

	// A negative budget is invalid input and falls back to the default.
	neg := -1
	idx, err = KnowledgeIndexTool(context.Background(), nil, cfg, &IndexInput{Budget: &neg})
	if err != nil {
		t.Fatalf("KnowledgeIndexTool: %v", err)
	}
	if idx.Budget != DefaultKnowledgeBudget {
		t.Errorf("a negative budget should fall back to the default, got %d", idx.Budget)
	}
}

// TestKnowledgeIndexTool_NothingResolves: no profile and no personalization root yields an
// empty index with an empty root, never an error.
func TestKnowledgeIndexTool_NothingResolves(t *testing.T) {
	bare := &config.Config{DeskRoot: t.TempDir(), DeskName: "bare"}
	idx, err := KnowledgeIndexTool(context.Background(), nil, bare, &IndexInput{})
	if err != nil {
		t.Fatalf("an unresolvable knowledge root must not error: %v", err)
	}
	if idx.Root != "" || idx.FileCount != 0 || len(idx.Entries) != 0 {
		t.Fatalf("got %+v, want an empty index rooted at \"\"", idx)
	}
}

// TestKnowledgeIndexTool_ProfilelessKnowledgeRoot: a desk with a personalization root but no
// profile file still indexes its background prose.
func TestKnowledgeIndexTool_ProfilelessKnowledgeRoot(t *testing.T) {
	root := t.TempDir()
	knowledge := filepath.Join(root, config.ProfileRootDir)
	write(t, knowledge, "notes.md", "alpha beta")
	cfg := &config.Config{DeskRoot: root, DeskName: "bare"}

	idx, err := KnowledgeIndexTool(context.Background(), nil, cfg, &IndexInput{})
	if err != nil {
		t.Fatalf("KnowledgeIndexTool: %v", err)
	}
	if idx.Root != knowledge || idx.FileCount != 1 {
		t.Fatalf("got %+v, want the personalization root indexed", idx)
	}
}

// TestDiscoveryFallsBackToWorkingDirectory: with no resolved desk root, discovery starts at the
// process working directory (the legacy launched-from-inside-the-desk case).
func TestDiscoveryFallsBackToWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, config.ProfileRootDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(validProfile), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Chdir(root)

	out, err := ProfileGet(context.Background(), nil, nil, &GetInput{Path: "desk.name"})
	if err != nil {
		t.Fatalf("ProfileGet with a nil config: %v", err)
	}
	if out.Value != "scratch" {
		t.Errorf("Value = %q, want the cwd-discovered profile's value", out.Value)
	}
}
