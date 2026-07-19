package librarian

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
