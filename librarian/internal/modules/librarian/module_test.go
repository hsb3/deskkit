package librarian

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/schema"
)

// TestMigrations_MatchesCollectionsDir is the §2.8 manifest-vs-disk drift guard: the .go
// basenames actually present in modules/librarian/collections (excluding _test.go) must equal
// the Basenames declared in (*Mod).Migrations(), so a migration file added without updating the
// manifest fails loud here instead of silently missing stamp-by-observation / the drift check.
func TestMigrations_MatchesCollectionsDir(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	collectionsDir := filepath.Join(filepath.Dir(thisFile), "collections")
	entries, err := os.ReadDir(collectionsDir)
	if err != nil {
		t.Fatalf("read collections dir: %v", err)
	}

	onDisk := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		onDisk[strings.TrimSuffix(name, ".go")] = true
	}

	manifest := map[string]bool{}
	for _, mig := range (&Mod{}).Migrations() {
		manifest[mig.Basename] = true
	}

	for b := range onDisk {
		if !manifest[b] {
			t.Errorf("collections/%s.go exists on disk but is missing from (*Mod).Migrations()", b)
		}
	}
	for b := range manifest {
		if !onDisk[b] {
			t.Errorf("(*Mod).Migrations() declares %q but collections/%s.go does not exist", b, b)
		}
	}
}

// TestSchemaVersion_MatchesHighestMigration guards the GuardDowngrade lockout that a
// three-migration change makes newly likely: SchemaVersion() must equal the highest migration
// sequence the module declares. A migration whose author bumps the basenames manifest but forgets
// SchemaVersion() leaves a migrated store stamped AHEAD of what the binary claims, and
// GuardDowngrade (core/migrate) then refuses to start on the next run — even for the very binary
// that just applied the migration. TestMigrations_MatchesCollectionsDir above checks the manifest
// against disk but never SchemaVersion() itself; this closes that gap, mirroring the PM module's
// TestMigrations_MatchOwnedCollections SchemaVersion assertion (whose comment already claimed to
// mirror "the librarian module's" check — a claim this test finally makes true).
func TestSchemaVersion_MatchesHighestMigration(t *testing.T) {
	highest := 0
	for _, mig := range (&Mod{}).Migrations() {
		// basenames are "NNNN_slug"; the leading zero-padded integer is the sequence number.
		seq := 0
		for i := 0; i < len(mig.Basename) && mig.Basename[i] >= '0' && mig.Basename[i] <= '9'; i++ {
			seq = seq*10 + int(mig.Basename[i]-'0')
		}
		if seq > highest {
			highest = seq
		}
	}
	if got := (&Mod{}).SchemaVersion(); got != highest {
		t.Errorf("SchemaVersion() = %d but the highest declared migration sequence is %d — bump "+
			"SchemaVersion() to match, or GuardDowngrade refuses the store on the next start", got, highest)
	}
}

// --- D3: the DocumentValidator wiring (spec §2.5/§4.4; test lane §10.1's verdict half) ---

func verdictEnv(t *testing.T) *Mod {
	t.Helper()
	m := &Mod{}
	m.Configure(&config.Config{DeskRoot: t.TempDir(), DeskName: "verdict-desk"})
	return m
}

func writeDeskFile(t *testing.T, m *Mod, rel, content string) {
	t.Helper()
	abs := filepath.Join(m.cfg.DeskRoot, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestVerdict_AbsentInvalidWrongStatusSatisfied drives one document through the four §10.1
// verdict states: absent -> invalid frontmatter -> wrong status -> satisfied.
func TestVerdict_AbsentInvalidWrongStatusSatisfied(t *testing.T) {
	m := verdictEnv(t)
	ctx := context.Background()
	req := schema.ArtifactRequirement{Type: "decision", RequiredStatus: "accepted"}
	rel := "_structure/decisions/0001-test.md"

	// Absent.
	v, err := m.Verdict(ctx, rel, req)
	if err != nil {
		t.Fatal(err)
	}
	if v.Exists || v.Satisfied || len(v.Missing) == 0 {
		t.Fatalf("absent doc: %+v", v)
	}

	// Invalid frontmatter (no fence at all).
	writeDeskFile(t, m, rel, "just a body, no frontmatter\n")
	v, _ = m.Verdict(ctx, rel, req)
	if !v.Exists || v.FrontmatterValid || v.Satisfied {
		t.Fatalf("frontmatter-less doc: %+v", v)
	}

	// Valid frontmatter, wrong status.
	writeDeskFile(t, m, rel, `---
type: decision
status: proposed
created: 2026-07-18
updated: 2026-07-18
tags: [pm]
decided_by: owner
affects_workstreams: [pm]
---
body
`)
	v, _ = m.Verdict(ctx, rel, req)
	if !v.Exists || !v.FrontmatterValid || v.Satisfied || v.Status != "proposed" {
		t.Fatalf("wrong-status doc: %+v", v)
	}
	joined := strings.Join(v.Missing, "; ")
	if !strings.Contains(joined, `"proposed"`) || !strings.Contains(joined, `"accepted"`) {
		t.Fatalf("refusal must name actual vs required status: %v", v.Missing)
	}

	// Satisfied.
	writeDeskFile(t, m, rel, `---
type: decision
status: accepted
created: 2026-07-18
updated: 2026-07-18
tags: [pm]
decided_by: owner
affects_workstreams: [pm]
---
body
`)
	v, _ = m.Verdict(ctx, rel, req)
	if !v.Satisfied || len(v.Missing) != 0 {
		t.Fatalf("satisfied doc: %+v", v)
	}

	// No required status: existence + validity suffices even at status proposed.
	v, _ = m.Verdict(ctx, rel, schema.ArtifactRequirement{Type: "decision"})
	if !v.Satisfied {
		t.Fatalf("existence-only requirement should be satisfied: %+v", v)
	}
}

// TestVerdict_TypeMismatchAndMissingFields: a doc of another type, or missing the required
// type-specific fields, is not FrontmatterValid for the gate's required type.
func TestVerdict_TypeMismatchAndMissingFields(t *testing.T) {
	m := verdictEnv(t)
	ctx := context.Background()
	writeDeskFile(t, m, "analyses/a.md", `---
type: analysis
status: approved
created: 2026-07-18
updated: 2026-07-18
tags: [x]
author: someone
---
`)
	v, _ := m.Verdict(ctx, "analyses/a.md", schema.ArtifactRequirement{Type: "decision", RequiredStatus: "accepted"})
	if v.FrontmatterValid || v.Satisfied {
		t.Fatalf("type-mismatched doc must not satisfy a decision gate: %+v", v)
	}

	// Missing decision-required fields (decided_by, affects_workstreams).
	writeDeskFile(t, m, "d.md", `---
type: decision
status: accepted
created: 2026-07-18
updated: 2026-07-18
tags: [x]
---
`)
	v, _ = m.Verdict(ctx, "d.md", schema.ArtifactRequirement{Type: "decision", RequiredStatus: "accepted"})
	if v.FrontmatterValid || v.Satisfied {
		t.Fatalf("doc missing required fields must not validate: %+v", v)
	}
}

// TestVerdict_FailsClosed: nil config and URL pointers can never satisfy a gate.
func TestVerdict_FailsClosed(t *testing.T) {
	ctx := context.Background()
	unconfigured := &Mod{}
	v, err := unconfigured.Verdict(ctx, "x.md", schema.ArtifactRequirement{Type: "decision"})
	if err != nil || v.Satisfied || len(v.Missing) == 0 {
		t.Fatalf("unconfigured validator must fail closed: %+v %v", v, err)
	}

	m := verdictEnv(t)
	v, _ = m.Verdict(ctx, "https://example.com/tracker/issues/1", schema.ArtifactRequirement{Type: "decision"})
	if v.Satisfied || len(v.Missing) == 0 {
		t.Fatalf("URL pointer must fail a document gate: %+v", v)
	}
	if v, _ = m.Verdict(ctx, "", schema.ArtifactRequirement{Type: "decision"}); v.Satisfied {
		t.Fatalf("empty pointer must fail: %+v", v)
	}
}

// TestVerdict_ToleratesSectionAnchorSuffix: a pointer carrying an advisory "§ heading" section
// anchor resolves by its FILE part — the file must exist and validate, but the heading itself is
// never required to exist. This is the seeded-pointer robustness case: an item whose pointer was
// seeded with a "§ Some Heading" suffix must not fail its first gated transition with "document
// not found".
func TestVerdict_ToleratesSectionAnchorSuffix(t *testing.T) {
	m := verdictEnv(t)
	ctx := context.Background()
	req := schema.ArtifactRequirement{Type: "decision", RequiredStatus: "accepted"}

	writeDeskFile(t, m, "somedoc.md", `---
type: decision
status: accepted
created: 2026-07-18
updated: 2026-07-18
tags: [pm]
decided_by: owner
affects_workstreams: [pm]
---
## Some Heading
body
`)

	// The file exists and validates; the "§ Some Heading" suffix must be tolerated and ignored
	// for existence, so the gate is satisfied.
	v, err := m.Verdict(ctx, "somedoc.md § Some Heading", req)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Exists {
		t.Fatalf("section-anchor pointer must resolve to its file part and exist: %+v", v)
	}
	if !v.Satisfied || len(v.Missing) != 0 {
		t.Fatalf("section-anchor pointer over a valid+accepted doc must satisfy the gate: %+v", v)
	}

	// The heading need not exist for the gate: a suffix naming an absent heading still resolves,
	// because the heading is advisory and never checked.
	v, _ = m.Verdict(ctx, "somedoc.md § A Heading That Is Not In The File", req)
	if !v.Satisfied {
		t.Fatalf("an absent heading must not fail the gate (heading is advisory): %+v", v)
	}

	// A section anchor over a genuinely MISSING file still fails closed.
	v, _ = m.Verdict(ctx, "nope.md § Some Heading", req)
	if v.Exists || v.Satisfied || len(v.Missing) == 0 {
		t.Fatalf("section anchor over a missing file must still fail: %+v", v)
	}

	// A URL pointer carrying a section anchor is still rejected (the URL guard runs on the file
	// part, so a "://"-bearing file part can never satisfy a document gate).
	v, _ = m.Verdict(ctx, "https://example.com/doc § Some Heading", req)
	if v.Exists || v.Satisfied || len(v.Missing) == 0 {
		t.Fatalf("URL pointer with a section anchor must still fail a document gate: %+v", v)
	}

	// Regression: a plain pointer (no anchor) still resolves and satisfies.
	if v, _ = m.Verdict(ctx, "somedoc.md", req); !v.Satisfied {
		t.Fatalf("plain pointer must still satisfy: %+v", v)
	}
}

// TestVerdict_HashAnchorNotStripped pins the deliberate non-handling of "#"-style anchors:
// only "§" delimits a section anchor (sectionFilePart), so a markdown-convention
// "file.md#heading" pointer does NOT resolve to its file part — it fails closed, and the
// failure names the supported "§ heading" form so the fix is actionable.
func TestVerdict_HashAnchorNotStripped(t *testing.T) {
	m := verdictEnv(t)
	ctx := context.Background()
	req := schema.ArtifactRequirement{Type: "decision", RequiredStatus: "accepted"}

	writeDeskFile(t, m, "somedoc.md", `---
type: decision
status: accepted
created: 2026-07-18
updated: 2026-07-18
tags: [pm]
decided_by: owner
affects_workstreams: [pm]
---
## Some Heading
body
`)

	v, err := m.Verdict(ctx, "somedoc.md#some-heading", req)
	if err != nil {
		t.Fatal(err)
	}
	if v.Exists || v.Satisfied {
		t.Fatalf("a #-anchored pointer must not resolve to its file part (only § is stripped): %+v", v)
	}
	if len(v.Missing) != 1 || !strings.Contains(v.Missing[0], "§") {
		t.Fatalf("the failure must hint at the supported § anchor form, got %+v", v.Missing)
	}
}

// TestFrontmatter_Reader: the trait-predicate companion seam returns the pointed doc's
// frontmatter, and an empty map (never an error) for anything unreadable.
func TestFrontmatter_Reader(t *testing.T) {
	m := verdictEnv(t)
	ctx := context.Background()
	writeDeskFile(t, m, "doc.md", "---\ngoverns: desk-operations\ntype: decision\n---\n")
	fm, err := m.Frontmatter(ctx, "doc.md")
	if err != nil || fm["governs"] != "desk-operations" {
		t.Fatalf("Frontmatter: %v %v", fm, err)
	}
	if fm, err := m.Frontmatter(ctx, "absent.md"); err != nil || len(fm) != 0 {
		t.Fatalf("absent file must yield an empty map, got %v %v", fm, err)
	}
}

// TestVerdict_RefusesTraversalOutsideDeskRoot: a pointer that escapes DESK_ROOT (via `..` or
// an absolute path elsewhere) can never satisfy a gate, and Frontmatter returns nothing for it.
func TestVerdict_RefusesTraversalOutsideDeskRoot(t *testing.T) {
	m := verdictEnv(t)
	ctx := context.Background()

	outside := filepath.Join(filepath.Dir(m.cfg.DeskRoot), "outside.md")
	if err := os.WriteFile(outside, []byte("---\ntype: decision\nstatus: accepted\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, ptr := range []string{"../outside.md", "a/../../outside.md", outside} {
		v, err := m.Verdict(ctx, ptr, schema.ArtifactRequirement{Type: "decision"})
		if err != nil || v.Exists || v.Satisfied {
			t.Errorf("pointer %q escaping DESK_ROOT must fail closed: %+v %v", ptr, v, err)
		}
		if fm, _ := m.Frontmatter(ctx, ptr); len(fm) != 0 {
			t.Errorf("Frontmatter(%q) must return nothing outside DESK_ROOT, got %v", ptr, fm)
		}
	}
}
