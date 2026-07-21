// Package librarian is the base desk module: it wraps the existing librarian collections,
// tools, and hooks behind the module.Module + schema.DocumentValidator interfaces (spec §2.7).
// It is always enabled (Enabled ignores cfg) so the binary stays behaviorally identical to
// pre-refactor pocket-librarian.
package librarian

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/migrate"
	"github.com/hsb3/desk-standard/librarian/internal/core/module"
	"github.com/hsb3/desk-standard/librarian/internal/core/schema"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/core/tuiview"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/desklib"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/tools"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/trigger"
)

// New constructs the librarian module.
func New() module.Module { return &Mod{} }

// Mod implements module.Module + schema.DocumentValidator + schema.FrontmatterReader (and
// module.Configurable to receive the resolved config the validator needs).
type Mod struct {
	cfg *config.Config // injected by module.Register via Configure; nil until then
}

// Configure implements module.Configurable: module.Register injects the resolved config so
// Verdict can resolve document pointers against DESK_ROOT. cfg may be nil (config.Load
// failed); Verdict fails closed on that.
func (m *Mod) Configure(cfg *config.Config) { m.cfg = cfg }

func (*Mod) Name() string { return "librarian" }

// SchemaVersion is the highest migration sequence the librarian module declares (0022).
func (*Mod) SchemaVersion() int { return 22 }

// Enabled is always true: librarian is the base module (spec §2.7).
func (*Mod) Enabled(*config.Config) bool { return true }

// OwnedCollections lists every collection created by the librarian's 0001..0021 migrations
// (enumerated from the migration bodies; see module_test.go's drift guard for the migrations
// side). Unchanged by 0021, which adds a field to `files` rather than a new collection.
func (*Mod) OwnedCollections() []string {
	return []string{
		"files", "patrol_findings", "patrol_log", "revisions", "adoption_log",
		"agent_runs", "messages", "tasks", "prompts", "feedback",
	}
}

// Tools returns the librarian's seven tool specs.
func (*Mod) Tools() []toolcore.ToolSpec { return tools.Specs() }

// TUIViews is nil for the librarian: the chat transcript IS its TUI surface (ADR 0004); it
// contributes no extra mounted views (spec §5.3 — the plug-point exists for other modules).
func (*Mod) TUIViews(core.App, *config.Config) []tuiview.View { return nil }

// Migrations lists the librarian's 0001..0021 migrations. All are SelfRegistered: their
// bodies still call PocketBase's m.Register via their own init() (blank-imported by main via
// internal/modules/librarian/collections), so Up/Down are nil here — this manifest exists for
// stamp-by-observation (core/migrate.StampModules) and the drift test below, not to re-wire
// registration.
func (*Mod) Migrations() []migrate.Migration {
	basenames := []string{
		"0001_files", "0002_patrol_findings", "0003_patrol_log", "0004_revisions",
		"0005_adoption_log", "0006_agent_runs", "0007_messages", "0008_tasks",
		"0009_prompts", "0010_patrol_findings_resolved", "0011_widen_content_fields",
		"0012_dir_kind_add_infra", "0013_feedback",
		"0014_patrol_findings_disposition", "0015_patrol_findings_drop_dismissed",
		"0016_patrol_findings_provenance", "0017_adoption_log_shrink_event",
		"0018_files_doc_id", "0019_files_doctype_rename", "0020_content_field_caps",
		"0021_files_content", "0022_agent_runs_archived",
	}
	out := make([]migrate.Migration, len(basenames))
	for i, b := range basenames {
		out[i] = migrate.Migration{Basename: b, SelfRegistered: true}
	}
	return out
}

// RegisterHooks wires the librarian's record hooks + cron (spec §2.4 wake layer). StartClaimer
// stays in main (it needs agentAction, which would otherwise pull internal/agent into
// internal/trigger) — see main.go wiring.
func (*Mod) RegisterHooks(app core.App, cfg *config.Config) error {
	if err := trigger.RegisterHooks(app, cfg); err != nil {
		return err
	}
	return trigger.RegisterCron(app, cfg)
}

// Verdict implements schema.DocumentValidator (spec §2.5, §4.4): a document is "filled" for a
// gate when it EXISTS at the pointer, its FRONTMATTER VALIDATES against schema v1 for the
// required type, and it carries the REQUIRED STATUS. The verdict is produced by the same
// engine family the patrol rules use — a direct desk-file read (DESK_ROOT-resolved pointer) +
// desklib.ParseFrontmatter + the core/schema doctypes vocabulary — NOT a `files`-collection
// lookup, so a verdict never depends on sweep freshness and needs no store handle (the gate
// engine's own store handle covers the PM collections; documents are files). Fails closed on
// unresolved config and on non-file pointers (a URL pointer can never satisfy a document gate).
func (m *Mod) Verdict(_ context.Context, pointer string, req schema.ArtifactRequirement) (schema.Verdict, error) {
	v := schema.Verdict{}
	fail := func(reason string) (schema.Verdict, error) {
		v.Missing = append(v.Missing, reason)
		return v, nil
	}
	if m.cfg == nil || m.cfg.DeskRoot == "" {
		return fail("document validator has no resolved desk config (DESK_ROOT); cannot verify " + pointer)
	}
	if pointer == "" {
		return fail(fmt.Sprintf("no document pointer set; a document (type=%s) is required", req.Type))
	}
	// A pointer may carry an advisory "§ heading" section anchor naming a heading INSIDE the
	// document (e.g. "notes.md § Decisions"). The heading is a human wayfinding hint, not part of
	// the file's identity: resolve and require only the FILE part to exist, and never check the
	// heading. This keeps a pointer resolving even after the document's headings are renamed, and
	// lets pointers that already carry such a suffix pass their first gated transition with no data
	// migration.
	file := sectionFilePart(pointer)
	if strings.Contains(file, "://") {
		return fail(fmt.Sprintf("pointer %q is not a desk file; a document gate needs a file path", pointer))
	}
	abs, ok := m.resolveDeskPath(file)
	if !ok {
		return fail(fmt.Sprintf("pointer %q resolves outside the desk root; a document gate reads desk files only", pointer))
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		msg := fmt.Sprintf("required document (type=%s) at %q does not exist", req.Type, pointer)
		// Only "§" delimits a section anchor (sectionFilePart), so a markdown-convention
		// "file.md#heading" pointer fails resolution looking for a file literally named that.
		// Name the likely cause instead of leaving a bare not-found for a file that may exist.
		if strings.ContainsRune(file, '#') {
			msg += `; note: "#heading" anchors are not stripped from pointers — write "file.md § Heading" instead`
		}
		return fail(msg)
	}
	v.Exists = true

	fm := desklib.ParseFrontmatter(string(b))
	if len(fm) == 0 {
		return fail(fmt.Sprintf("required document at %q has no valid frontmatter", pointer))
	}
	vocab, verr := schema.Vocab()
	if verr != nil {
		return schema.Verdict{}, verr // embedded-vocabulary parse failure is a build defect
	}
	docType, _ := fm["type"].(string)
	reasons := []string{}
	if docType != req.Type {
		reasons = append(reasons, fmt.Sprintf(
			"document at %q has type %q, gate requires type %q", pointer, docType, req.Type))
	}
	reasons = append(reasons, vocab.ValidateFrontmatter(fm, req.Type)...)
	if len(reasons) == 0 {
		v.FrontmatterValid = true
	} else {
		for _, r := range reasons {
			v.Missing = append(v.Missing, r)
		}
	}
	v.Status, _ = fm["status"].(string)

	statusOK := req.RequiredStatus == "" || v.Status == req.RequiredStatus
	if !statusOK {
		v.Missing = append(v.Missing, fmt.Sprintf(
			"required document (type=%s, status=%s) at %q is at status %q, needs %q",
			req.Type, req.RequiredStatus, pointer, v.Status, req.RequiredStatus))
	}
	v.Satisfied = v.Exists && v.FrontmatterValid && statusOK
	return v, nil
}

// Frontmatter implements schema.FrontmatterReader (spec §4.2 trait predicates): returns the
// parsed frontmatter of the pointed desk file, resolved the same way Verdict resolves
// pointers. A missing/unreadable/frontmatter-less file returns an empty map (the trait simply
// does not match), never an error the gate engine would have to interpret.
func (m *Mod) Frontmatter(_ context.Context, pointer string) (map[string]any, error) {
	if m.cfg == nil || m.cfg.DeskRoot == "" || pointer == "" {
		return map[string]any{}, nil
	}
	// Tolerate the same advisory "§ heading" section anchor Verdict does: resolve only the file
	// part, so a trait predicate over a section-anchored pointer reads the same document the gate
	// does (never a URL, never a heading).
	file := sectionFilePart(pointer)
	if strings.Contains(file, "://") {
		return map[string]any{}, nil
	}
	abs, ok := m.resolveDeskPath(file)
	if !ok {
		return map[string]any{}, nil
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return map[string]any{}, nil
	}
	return desklib.ParseFrontmatter(string(b)), nil
}

// sectionFilePart returns the FILE portion of a document pointer, dropping an advisory
// "§ heading" section anchor when one is present: "notes.md § Decisions" yields "notes.md",
// while a plain "notes.md" is returned unchanged. The text after "§" names a location inside the
// document for a human reader and is never part of the file's identity, so the gate resolves and
// requires only the file. Only "§" delimits a section anchor here; "#" is deliberately left
// untouched.
func sectionFilePart(pointer string) string {
	if i := strings.IndexRune(pointer, '§'); i >= 0 {
		return strings.TrimSpace(pointer[:i])
	}
	return pointer
}

// resolveDeskPath resolves a document pointer against DESK_ROOT and CONTAINS it there: a
// relative pointer joins the root, an absolute pointer is accepted only if already inside the
// root, and any `..` traversal escaping the root is refused (a gate whose purpose is document
// provenance must not read arbitrary filesystem paths).
func (m *Mod) resolveDeskPath(pointer string) (string, bool) {
	root := filepath.Clean(m.cfg.DeskRoot)
	abs := pointer
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, pointer)
	}
	abs = filepath.Clean(abs)
	if abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", false
	}
	return abs, true
}
