package importer

import (
	"context"
	"strings"
	"testing"
)

// TestRebuildReproducibility is test lane §10.8 (spec §8.2): a fresh store → migrate → import
// yields a graph that is a pure function of the manifest. Two independent rebuilds (store A and
// store B, each freshly migrated on the SAME desk) produce byte-identical graphs — record ids
// included, because the ids are derived from (desk, key). This is the PM-collections analogue
// of the librarian's file-checksum rebuild check in verify.sh §10.
func TestRebuildReproducibility(t *testing.T) {
	ctx := context.Background()
	m := sampleManifest()

	engA := newEngine(t, "repro-desk")
	if _, err := Import(ctx, engA, m); err != nil {
		t.Fatalf("import into store A: %v", err)
	}
	snapA, err := GraphSnapshot(ctx, engA)
	if err != nil {
		t.Fatal(err)
	}

	// Store B: a wholly independent fresh store, same desk name, same manifest.
	engB := newEngine(t, "repro-desk")
	if _, err := Import(ctx, engB, m); err != nil {
		t.Fatalf("import into store B: %v", err)
	}
	snapB, err := GraphSnapshot(ctx, engB)
	if err != nil {
		t.Fatal(err)
	}

	if snapA.Canonical() != snapB.Canonical() {
		t.Fatalf("two rebuilds must be byte-identical (§8.2):\n--- A ---\n%s\n--- B ---\n%s",
			snapA.Canonical(), snapB.Canonical())
	}

	// The stable-id property is the golden anchor: every item's record id equals the
	// deterministic ItemID for its manifest key. Assert it directly so a regression that made
	// ids non-deterministic (e.g. dropping the ID pin) would fail even if A and B still matched
	// by luck within one run.
	if len(snapA.Items) != len(m.Items) {
		t.Fatalf("snapshot has %d items, manifest has %d", len(snapA.Items), len(m.Items))
	}
	wantID := map[string]bool{}
	for _, it := range m.Items {
		wantID[ItemID("repro-desk", it.Key)] = true
	}
	for _, ip := range snapA.Items {
		if !wantID[ip.ID] {
			t.Errorf("item id %q is not a deterministic ItemID of any manifest key", ip.ID)
		}
	}

	// A different desk name produces a different graph (different ids) from the same manifest —
	// desk-scoping holds, so two desks' imports never collide.
	engC := newEngine(t, "other-desk")
	if _, err := Import(ctx, engC, m); err != nil {
		t.Fatal(err)
	}
	snapC, err := GraphSnapshot(ctx, engC)
	if err != nil {
		t.Fatal(err)
	}
	if snapA.Canonical() == snapC.Canonical() {
		t.Fatal("a different desk must produce a different graph (desk-scoped ids)")
	}
	// Sanity: the canonical form is non-trivial (guards against an empty-snapshot false pass).
	if !strings.Contains(snapA.Canonical(), "\"items\"") || len(snapA.Deps) != 2 {
		t.Fatalf("snapshot looks empty/wrong: %s", snapA.Canonical())
	}
}

// TestDepSnapshotKindTiebreak guards the dep-snapshot sort's Kind tiebreaker (issue 71).
// e_createDep dedups edges by the (from,to,kind) triple, so two edges sharing the SAME (From,To)
// pair but different Kind legitimately coexist. If the snapshot sort compares only From then To,
// those two edges keep their store-retrieval order — which follows dep INSERTION order — so the
// same graph rebuilt with the edges inserted in a different order yields a different Canonical(),
// a false-failure the §8.2 reproducibility oracle would trip on non-deterministically.
//
// TestRebuildReproducibility cannot catch this: it imports the SAME manifest in the SAME order
// into both stores, so any insertion-order dependence cancels out. This test perturbs the
// insertion order between the two stores and asserts Canonical() is byte-stable regardless.
// It is RED-able: remove the Kind comparator from GraphSnapshot's sort.Slice and it fails.
func TestDepSnapshotKindTiebreak(t *testing.T) {
	ctx := context.Background()

	// A local two-item manifest with two edges between the SAME pair (task-a -> task-b): a
	// gating "blocks" edge and an informational "relates-to" edge. Defined inline so the shared
	// sampleManifest() (which other tests pin by shape/count) is untouched. depsForward/Reversed
	// differ ONLY in the order the two same-pair edges are inserted.
	items := []ManifestItem{
		{Key: "task-a", Title: "Item A", Type: "task", Court: "desk", Priority: 1},
		{Key: "task-b", Title: "Item B", Type: "task", Court: "crew", Priority: 2},
	}
	blocksEdge := ManifestDep{From: "task-a", To: "task-b", Kind: "blocks", UnblockAt: "review", Cascade: "auto"}
	relatesEdge := ManifestDep{From: "task-a", To: "task-b", Kind: "relates-to"}
	forward := Manifest{Items: items, Deps: []ManifestDep{blocksEdge, relatesEdge}}
	reversed := Manifest{Items: items, Deps: []ManifestDep{relatesEdge, blocksEdge}}

	engFwd := newEngine(t, "tiebreak-desk")
	if _, err := Import(ctx, engFwd, forward); err != nil {
		t.Fatalf("import forward: %v", err)
	}
	snapFwd, err := GraphSnapshot(ctx, engFwd)
	if err != nil {
		t.Fatal(err)
	}

	engRev := newEngine(t, "tiebreak-desk")
	if _, err := Import(ctx, engRev, reversed); err != nil {
		t.Fatalf("import reversed: %v", err)
	}
	snapRev, err := GraphSnapshot(ctx, engRev)
	if err != nil {
		t.Fatal(err)
	}

	// Guard the fixture actually exercises the case: two edges on the same (From,To) pair.
	if len(snapFwd.Deps) != 2 {
		t.Fatalf("fixture must produce 2 same-pair edges, got %d", len(snapFwd.Deps))
	}
	if snapFwd.Deps[0].From != snapFwd.Deps[1].From || snapFwd.Deps[0].To != snapFwd.Deps[1].To {
		t.Fatalf("fixture edges must share the same (From,To) pair, got %+v", snapFwd.Deps)
	}

	// The heart of the Kind tiebreaker: insertion order must not leak into the canonical bytes. Without the Kind
	// tiebreaker in GraphSnapshot's sort, the two same-pair edges appear in insertion order and
	// these two canonicals diverge.
	if snapFwd.Canonical() != snapRev.Canonical() {
		t.Fatalf("dep-snapshot must be byte-stable across dep insertion order (Kind tiebreaker):\n--- forward ---\n%s\n--- reversed ---\n%s",
			snapFwd.Canonical(), snapRev.Canonical())
	}
}
