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
