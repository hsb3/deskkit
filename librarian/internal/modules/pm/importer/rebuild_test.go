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
