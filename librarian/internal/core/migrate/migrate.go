// Package migrate is the module-schema-versioning framework (spec §2.7/§2.8). Each module
// (librarian, pm, ...) declares its migrations as a manifest of Migration entries; the
// librarian's are SelfRegistered (they already call PocketBase's m.Register via init() +
// blank-import — that path is left untouched, D2 is a pure move) while a future module may
// supply real Up/Down functions registered programmatically here (RegisterProgrammatic).
//
// After migrations run, StampModules reads PocketBase's own applied-migrations ledger
// (_migrations) and, for each enabled module, upserts a module_schema_versions row recording
// the highest migration sequence that module has applied — "stamp by observation" rather than
// a module owning its own version bookkeeping. GuardDowngrade compares a module's declared
// SchemaVersion() against its stamped row and refuses to proceed if the STORE is ahead of the
// BINARY (an old binary opened a store a newer one already migrated).
package migrate

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migration describes one of a module's schema migrations. Basename is the migration's file
// stem (e.g. "0001_files"), which PocketBase's runner records as the `file` column in its own
// `_migrations` ledger table with a numeric prefix + ".go" — StampModules matches on the
// leading NNNN sequence, not the full recorded string, so it is insensitive to that exact
// suffix format. SelfRegistered marks a librarian-style migration that already calls
// pocketbase's m.Register via its own init() (blank-imported by main); Up/Down are nil for
// those. A non-self-registered migration supplies real Up/Down and is wired by
// RegisterProgrammatic.
type Migration struct {
	Basename       string
	Up, Down       func(app core.App) error
	SelfRegistered bool
}

// sequence parses the leading NNNN digits off a migration basename (e.g. "0013" from
// "0013_feedback"). Returns -1 if the basename does not start with digits.
func sequence(basename string) int {
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

// Module is the narrow slice of module.Module this package needs, expressed locally to avoid
// an import cycle (core/module already imports core/migrate): any module.Module value
// satisfies this interface, so module.Registry adapts its enabled set to []migrate.Module for
// RegisterProgrammatic/StampModules/GuardDowngrade.
type Module interface {
	Name() string
	SchemaVersion() int
	Migrations() []Migration
}

// RegisterProgrammatic wires each enabled module's non-self-registered migrations onto
// PocketBase's migration runner via m.Register(up, down, basename), using the SAME explicit
// basename so stamp-by-observation (StampModules) matches it later. D2: every module's
// migrations are either empty (pm) or SelfRegistered (librarian), so this is a no-op in
// practice — but the code path must exist and compile for D3+ modules that supply real
// Up/Down.
func RegisterProgrammatic(mods []Module) {
	for _, mod := range mods {
		for _, mig := range mod.Migrations() {
			if mig.SelfRegistered {
				continue
			}
			m.Register(mig.Up, mig.Down, mig.Basename)
		}
	}
}

// StampModules reads PocketBase's own applied-migrations ledger (_migrations) and, for each
// module, upserts its module_schema_versions row with the highest applied sequence among that
// module's declared Migrations(). Non-fatal on error (logs via the returned error to the
// caller, which logs and continues) — this is observational bookkeeping, never a gate that
// blocks serve/requireConfig.
func StampModules(app core.App, mods []Module) error {
	appliedFiles, err := appliedMigrationFiles(app)
	if err != nil {
		return fmt.Errorf("migrate: read applied migrations: %w", err)
	}
	applied := map[string]bool{}
	for _, f := range appliedFiles {
		applied[f] = true
	}

	for _, mod := range mods {
		highest := -1
		for _, mig := range mod.Migrations() {
			if !migrationApplied(applied, mig.Basename) {
				continue
			}
			if seq := sequence(mig.Basename); seq > highest {
				highest = seq
			}
		}
		if highest < 0 {
			continue // this module has no applied migrations yet
		}
		if err := upsertModuleVersion(app, mod.Name(), highest); err != nil {
			return fmt.Errorf("migrate: stamp module %q: %w", mod.Name(), err)
		}
	}
	return nil
}

// GuardDowngrade refuses to proceed if a module's STORED schema version (from a prior, newer
// binary) exceeds the version this binary's module declares — an old binary opening a store a
// newer one already migrated further. D2: librarian's SchemaVersion()==13 matches its highest
// migration, so this never trips in the zero-change envelope; the guard exists for future
// migrations.
func GuardDowngrade(app core.App, mods []Module) error {
	for _, mod := range mods {
		stored, ok, err := storedModuleVersion(app, mod.Name())
		if err != nil {
			return fmt.Errorf("migrate: read stored version for module %q: %w", mod.Name(), err)
		}
		if ok && stored > mod.SchemaVersion() {
			return fmt.Errorf(
				"migrate: store schema for module %q is at version %d but this binary only knows version %d; refusing to downgrade",
				mod.Name(), stored, mod.SchemaVersion())
		}
	}
	return nil
}

// appliedMigrationFiles returns every `file` recorded in PocketBase's `_migrations` ledger
// table (the table migratecmd/RunAppMigrations maintains).
func appliedMigrationFiles(app core.App) ([]string, error) {
	var files []string
	err := app.DB().Select("file").From("_migrations").Column(&files)
	return files, err
}

// migrationApplied reports whether basename (e.g. "0013_feedback") was recorded as applied.
// PocketBase records the file with a ".go" suffix appended (e.g. "0013_feedback.go"), so this
// matches on basename equality OR basename+".go" for robustness against that exact suffix
// convention.
func migrationApplied(applied map[string]bool, basename string) bool {
	if applied[basename] {
		return true
	}
	return applied[basename+".go"]
}

// moduleSchemaVersionsCollection is the meta collection's stable name (created by
// 0000_module_schema_versions.go, core-owned).
const moduleSchemaVersionsCollection = "module_schema_versions"

// upsertModuleVersion writes (or updates) the module_schema_versions row for module, recording
// version and the current time.
func upsertModuleVersion(app core.App, module string, version int) error {
	col, err := app.FindCollectionByNameOrId(moduleSchemaVersionsCollection)
	if err != nil {
		return err
	}
	rec, ferr := app.FindFirstRecordByFilter(moduleSchemaVersionsCollection, "module = {:m}", map[string]any{"m": module})
	if ferr != nil {
		// Not found (sql.ErrNoRows, wrapped) is the expected first-stamp case: create the row.
		rec = core.NewRecord(col)
		rec.Set("module", module)
	}
	rec.Set("version", version)
	rec.Set("applied_at", time.Now())
	return app.Save(rec)
}

// storedModuleVersion reads the currently-stamped version for module, if any.
func storedModuleVersion(app core.App, module string) (version int, found bool, err error) {
	rec, ferr := app.FindFirstRecordByFilter(moduleSchemaVersionsCollection, "module = {:m}", map[string]any{"m": module})
	if ferr != nil {
		return 0, false, nil // not found (or collection not yet migrated) is not an error here
	}
	return rec.GetInt("version"), true, nil
}
