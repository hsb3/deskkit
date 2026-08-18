package pm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/pm/collections"
)

// TestNoSelfRegisteredMigrations is the spec §2.8a drift test: a PM migration written in the
// librarian's init()+m.Register style registers unconditionally at compile time and DEFEATS
// the feature gate. Three independent assertions:
//  1. every manifest entry is programmatic (SelfRegistered=false, real Up/Down);
//  2. no source file under modules/pm declares an init() (the self-registration vehicle);
//  3. importing this package (and everything under it reachable here) has NOT grown
//     PocketBase's global migration list with any pm basename.
func TestNoSelfRegisteredMigrations(t *testing.T) {
	for _, mig := range (&Mod{}).Migrations() {
		if mig.SelfRegistered {
			t.Errorf("pm migration %q is marked SelfRegistered — forbidden (§2.8a)", mig.Basename)
		}
		if mig.Up == nil || mig.Down == nil {
			t.Errorf("pm migration %q must carry real Up/Down for programmatic registration", mig.Basename)
		}
	}

	pmRoot := packageDir(t)
	err := filepath.WalkDir(pmRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Errorf("parse %s: %v", path, perr)
			return nil
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "init" {
				t.Errorf("%s declares func init() at %s — the self-registration vehicle is forbidden in modules/pm (§2.8a)",
					path, fset.Position(fn.Pos()))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk modules/pm: %v", err)
	}

	for _, item := range pbcore.AppMigrations.Items() {
		if strings.Contains(item.File, "_pm_") {
			t.Errorf("global migration list contains %q at plain import time — pm migrations must only "+
				"be registered programmatically when the module is enabled", item.File)
		}
	}
}

// TestMigrations_MatchOwnedCollections applies every pm migration to nothing (static check):
// the manifest's basenames are ordered, unique, and SchemaVersion pins the highest sequence;
// OwnedCollections and the collections the migrations create agree (drift guard mirroring the
// librarian module's).
func TestMigrations_MatchOwnedCollections(t *testing.T) {
	mod := &Mod{}
	migs := mod.Migrations()
	if len(migs) == 0 {
		t.Fatal("pm module declares no migrations")
	}
	highest := 0
	seen := map[string]bool{}
	for i, mig := range migs {
		if seen[mig.Basename] {
			t.Errorf("duplicate migration basename %q", mig.Basename)
		}
		seen[mig.Basename] = true
		if !strings.Contains(mig.Basename, "_pm_") {
			t.Errorf("pm migration basename %q must carry the _pm_ marker (stamp-by-observation "+
				"matches basenames; a librarian-colliding name would mis-stamp)", mig.Basename)
		}
		seq := leadingSequence(mig.Basename)
		if seq < 0 {
			t.Errorf("migration basename %q has no leading sequence", mig.Basename)
		}
		if seq != i+1 {
			t.Errorf("migration %q out of order: sequence %d at position %d", mig.Basename, seq, i)
		}
		if seq > highest {
			highest = seq
		}
	}
	if mod.SchemaVersion() != highest {
		t.Errorf("SchemaVersion()=%d must equal the highest migration sequence %d", mod.SchemaVersion(), highest)
	}
	if got, want := len(mod.OwnedCollections()), len(collections.Names()); got != want {
		t.Errorf("OwnedCollections mismatch: %d vs %d", got, want)
	}
}

// TestEnabled_Gate: nil config and default config are OFF; PMEnabled turns it on (§2.9).
func TestEnabled_Gate(t *testing.T) {
	mod := &Mod{}
	if mod.Enabled(nil) {
		t.Error("nil config must be off (fail closed)")
	}
	if mod.Enabled(&config.Config{}) {
		t.Error("default config must be off")
	}
	if !mod.Enabled(&config.Config{PMEnabled: true}) {
		t.Error("PMEnabled config must be on")
	}
}

// TestNoLibrarianImports is test lane §10.5's import-graph half: no PRODUCTION code under
// modules/pm may import modules/librarian — the PM module consumes documents only through the
// core/schema seam. _test.go files are exempt: the gatedon/gatedoff integration proofs
// deliberately assemble a full desk (librarian + pm), which is a test-harness concern, not a
// production dependency. (Transitive safety: modules/pm imports only internal/core packages,
// and no internal/core production package imports modules/librarian.)
func TestNoLibrarianImports(t *testing.T) {
	pmRoot := packageDir(t)
	err := filepath.WalkDir(pmRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, "internal/modules/librarian") {
				t.Errorf("%s imports %s — modules/pm must consume the core/schema seam, never the librarian (§10.5)",
					path, imp.Path.Value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk modules/pm: %v", err)
	}

	// The transitive leg: no internal/core package imports modules/librarian either.
	coreRoot := filepath.Join(pmRoot, "..", "..", "core")
	err = filepath.WalkDir(coreRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, "internal/modules/librarian") {
				t.Errorf("core package %s imports %s — core must stay module-free", path, imp.Path.Value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/core: %v", err)
	}
}

// --- helpers ---

func packageDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(thisFile)
}

// leadingSequence parses the NNNN prefix off a migration basename; -1 when absent.
func leadingSequence(basename string) int {
	i := 0
	for i < len(basename) && basename[i] >= '0' && basename[i] <= '9' {
		i++
	}
	if i == 0 {
		return -1
	}
	n, err := strconv.Atoi(basename[:i])
	if err != nil {
		return -1
	}
	return n
}
