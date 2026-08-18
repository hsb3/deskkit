package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/config"

	// Blank-import registers this project's Go migrations so tests.NewTestApp applies them.
	_ "github.com/hsb3/deskkit/internal/modules/librarian/collections"
)

const (
	testSuperEmail = "admin@desk.test"
	testSuperPass  = "a-sufficiently-long-password"
)

func newBootstrapApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func countSuperusers(t *testing.T, app core.App, email string) int {
	t.Helper()
	recs, err := app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("list superusers: %v", err)
	}
	n := 0
	for _, r := range recs {
		if r.Email() == email {
			n++
		}
	}
	return n
}

func TestEnsureSuperuser_CreatesWhenBothSet(t *testing.T) {
	app := newBootstrapApp(t)
	cfg := &config.Config{PBSuperuserEmail: testSuperEmail, PBSuperuserPassword: testSuperPass}

	created, err := EnsureSuperuser(app, cfg)
	if err != nil {
		t.Fatalf("EnsureSuperuser: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true on first run")
	}
	if got := countSuperusers(t, app, testSuperEmail); got != 1 {
		t.Fatalf("expected exactly 1 superuser %q, got %d", testSuperEmail, got)
	}
}

func TestEnsureSuperuser_Idempotent(t *testing.T) {
	app := newBootstrapApp(t)
	cfg := &config.Config{PBSuperuserEmail: testSuperEmail, PBSuperuserPassword: testSuperPass}

	if _, err := EnsureSuperuser(app, cfg); err != nil {
		t.Fatalf("first EnsureSuperuser: %v", err)
	}
	// Second call must be a no-op: no error, created=false, still exactly one account.
	created, err := EnsureSuperuser(app, cfg)
	if err != nil {
		t.Fatalf("second EnsureSuperuser must not error when the account exists: %v", err)
	}
	if created {
		t.Fatalf("expected created=false on the idempotent second run")
	}
	if got := countSuperusers(t, app, testSuperEmail); got != 1 {
		t.Fatalf("idempotency broken: expected exactly 1 superuser, got %d", got)
	}
}

// TestEnsureSuperuser_NoopWhenBothUnset pins the one silent case that must survive: BOTH vars
// unset is today's normal local UX (an identity-neutral binary never invents a credential, spec
// §10.3) and must stay a no-op, not an error. The half-configured case is an error, asserted
// separately below.
func TestEnsureSuperuser_NoopWhenBothUnset(t *testing.T) {
	app := newBootstrapApp(t)
	before := len(func() []*core.Record {
		recs, _ := app.FindAllRecords(core.CollectionNameSuperusers)
		return recs
	}())

	for _, cfg := range []*config.Config{nil, {}} {
		created, err := EnsureSuperuser(app, cfg)
		if err != nil {
			t.Fatalf("EnsureSuperuser must not error when BOTH vars are unset: %v", err)
		}
		if created {
			t.Fatalf("expected created=false when both vars are unset")
		}
	}

	after, err := app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("list superusers: %v", err)
	}
	if len(after) != before {
		t.Fatalf("superuser count changed on a no-op: before=%d after=%d", before, len(after))
	}
}

// TestEnsureSuperuser_HalfConfiguredEnvFailsLoudly: exactly one of the pair set is an operator
// mistake. It previously returned (false, nil) — a silently disabled superuser, i.e. an auth
// environment that looks configured but is not. It must now be a loud, typed failure, and it must
// still create nothing.
func TestEnsureSuperuser_HalfConfiguredEnvFailsLoudly(t *testing.T) {
	app := newBootstrapApp(t)
	before := len(func() []*core.Record {
		recs, _ := app.FindAllRecords(core.CollectionNameSuperusers)
		return recs
	}())

	cases := map[string]*config.Config{
		"password missing": {PBSuperuserEmail: testSuperEmail},
		"email missing":    {PBSuperuserPassword: testSuperPass},
	}
	for name, cfg := range cases {
		created, err := EnsureSuperuser(app, cfg)
		if !errors.Is(err, ErrHalfConfiguredSuperuser) {
			t.Fatalf("%s: err = %v, want ErrHalfConfiguredSuperuser", name, err)
		}
		if created {
			t.Fatalf("%s: expected created=false on a half-configured environment", name)
		}
	}

	after, err := app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("list superusers: %v", err)
	}
	if len(after) != before {
		t.Fatalf("a half-configured environment must create nothing: before=%d after=%d", before, len(after))
	}
}

// TestValidateSuperuserEnv names the missing half in its message, so an operator reading the
// refusal knows which var to set rather than being told "one of these two".
func TestValidateSuperuserEnv_NamesTheMissingVar(t *testing.T) {
	if err := ValidateSuperuserEnv(nil); err != nil {
		t.Fatalf("nil config must validate: %v", err)
	}
	if err := ValidateSuperuserEnv(&config.Config{}); err != nil {
		t.Fatalf("both-unset must validate: %v", err)
	}
	both := &config.Config{PBSuperuserEmail: testSuperEmail, PBSuperuserPassword: testSuperPass}
	if err := ValidateSuperuserEnv(both); err != nil {
		t.Fatalf("both-set must validate: %v", err)
	}

	err := ValidateSuperuserEnv(&config.Config{PBSuperuserEmail: testSuperEmail})
	if err == nil || !strings.Contains(err.Error(), "PB_SUPERUSER_PASSWORD") {
		t.Fatalf("email-only error = %v, want it to name PB_SUPERUSER_PASSWORD", err)
	}
	err = ValidateSuperuserEnv(&config.Config{PBSuperuserPassword: testSuperPass})
	if err == nil || !strings.Contains(err.Error(), "PB_SUPERUSER_EMAIL") {
		t.Fatalf("password-only error = %v, want it to name PB_SUPERUSER_EMAIL", err)
	}
}

// TestCheckServeAuthPrereqs is the fail-closed gate for a non-loopback serve. Loopback keeps
// today's behavior in every case (that is the "local UX unchanged" half); public refuses unless an
// ADMINISTRABLE superuser actually exists once the gate has run.
func TestCheckServeAuthPrereqs(t *testing.T) {
	both := &config.Config{PBSuperuserEmail: testSuperEmail, PBSuperuserPassword: testSuperPass}

	t.Run("loopback tolerates an empty environment and an empty store", func(t *testing.T) {
		app := newBootstrapApp(t)
		for _, cfg := range []*config.Config{nil, {}, both} {
			if _, err := CheckServeAuthPrereqs(app, cfg, false); err != nil {
				t.Fatalf("loopback must not require auth prerequisites: %v", err)
			}
		}
	})

	t.Run("loopback creates nothing", func(t *testing.T) {
		// The gate provisions only on the public path; a local serve must reach the existing
		// (non-fatal) EnsureSuperuser call further down the hook exactly as it always did.
		app := newBootstrapApp(t)
		created, err := CheckServeAuthPrereqs(app, both, false)
		if err != nil || created {
			t.Fatalf("loopback gate must be inert: created=%v err=%v", created, err)
		}
	})

	t.Run("loopback still rejects a half-configured environment", func(t *testing.T) {
		app := newBootstrapApp(t)
		_, err := CheckServeAuthPrereqs(app, &config.Config{PBSuperuserEmail: testSuperEmail}, false)
		if !errors.Is(err, ErrHalfConfiguredSuperuser) {
			t.Fatalf("err = %v, want ErrHalfConfiguredSuperuser even on a loopback bind", err)
		}
	})

	t.Run("public refuses with no superuser and no environment", func(t *testing.T) {
		app := newBootstrapApp(t)
		emptySuperusers(t, app)
		for _, cfg := range []*config.Config{nil, {}} {
			_, err := CheckServeAuthPrereqs(app, cfg, true)
			if err == nil {
				t.Fatal("public bind with no superuser and no PB_SUPERUSER_* env must refuse to serve")
			}
			if !strings.Contains(err.Error(), "non-loopback") {
				t.Fatalf("refusal must name the exposure that caused it, got: %v", err)
			}
		}
	})

	t.Run("public accepts and PROVISIONS when both env vars are set", func(t *testing.T) {
		app := newBootstrapApp(t)
		emptySuperusers(t, app)
		created, err := CheckServeAuthPrereqs(app, both, true)
		if err != nil {
			t.Fatalf("both PB_SUPERUSER_* set must satisfy the public gate: %v", err)
		}
		if !created {
			t.Fatal("the gate must create the account itself, not defer to a later non-fatal call")
		}
		// The account is real and present the moment the gate returns.
		if n := administrableCount(t, app); n != 1 {
			t.Fatalf("administrable superusers after the gate = %d, want 1", n)
		}
	})

	t.Run("public accepts when the store already holds a superuser", func(t *testing.T) {
		app := newBootstrapApp(t)
		emptySuperusers(t, app)
		if _, err := EnsureSuperuser(app, both); err != nil {
			t.Fatalf("EnsureSuperuser: %v", err)
		}
		// The env is now empty but the account persists — the second-boot case.
		created, err := CheckServeAuthPrereqs(app, &config.Config{}, true)
		if err != nil {
			t.Fatalf("an existing superuser must satisfy the public gate: %v", err)
		}
		if created {
			t.Fatal("nothing should be created when the account already exists")
		}
	})
}

// TestCheckServeAuthPrereqs_InstallerPlaceholderDoesNotCount pins that the installer placeholder
// row never satisfies the public-serve gate.
//
// The dependency's first-run installer writes a `__pbinstaller@example.com` superuser row purely
// to mint its one-time setup link — it is not an account anyone can log in as. A plain
// CountRecords over _superusers therefore reports "a superuser exists" for a store nobody can
// administer, and because ONE ordinary loopback `deskkit serve` creates that row, the naive count
// let a single local boot permanently disarm this gate: the same store could then be hosted
// publicly with zero administrable accounts.
//
// This test models exactly that store — the placeholder and NOTHING else — so it must NOT use a
// helper that wipes the placeholder too, or it would pass vacuously.
func TestCheckServeAuthPrereqs_InstallerPlaceholderDoesNotCount(t *testing.T) {
	app := newBootstrapApp(t)
	emptySuperusers(t, app)
	seedInstallerPlaceholder(t, app)

	// Sanity: the store really does hold a row, so a naive count would see one.
	if n, err := app.CountRecords(core.CollectionNameSuperusers); err != nil || n != 1 {
		t.Fatalf("setup: raw superuser count = %d (err=%v), want exactly the placeholder", n, err)
	}
	if n := administrableCount(t, app); n != 0 {
		t.Fatalf("the installer placeholder must not count as administrable, got %d", n)
	}

	_, err := CheckServeAuthPrereqs(app, &config.Config{}, true)
	if err == nil {
		t.Fatal("a store holding ONLY the installer placeholder must not satisfy the public gate — " +
			"one ordinary loopback serve would otherwise disarm it forever")
	}
	if !strings.Contains(err.Error(), "administrable") {
		t.Fatalf("refusal should explain that the placeholder does not count, got: %v", err)
	}
}

// TestCheckServeAuthPrereqs_BadPasswordIsFatal pins that the public gate verifies the provisioned
// END STATE, never the env input: a gate that passes on the vars merely being non-empty would let
// a store-rejected password fail into a log row while the public listener came up with no
// administrable account. The gate must provision here, fatally, then re-count.
func TestCheckServeAuthPrereqs_BadPasswordIsFatal(t *testing.T) {
	app := newBootstrapApp(t)
	emptySuperusers(t, app)

	bad := &config.Config{PBSuperuserEmail: testSuperEmail, PBSuperuserPassword: "short"}
	created, err := CheckServeAuthPrereqs(app, bad, true)
	if err == nil {
		t.Fatal("a password the store rejects must FAIL the public gate, not be logged and ignored")
	}
	if created {
		t.Fatal("created must be false when provisioning failed")
	}
	if !strings.Contains(err.Error(), "non-loopback") {
		t.Fatalf("refusal must name the exposure that caused it, got: %v", err)
	}
	if n := administrableCount(t, app); n != 0 {
		t.Fatalf("no account should exist after a rejected password, got %d", n)
	}

	// A malformed email is the same class of failure and must also be fatal.
	if _, err := CheckServeAuthPrereqs(app,
		&config.Config{PBSuperuserEmail: "not-an-email", PBSuperuserPassword: testSuperPass}, true); err == nil {
		t.Fatal("an email the store rejects must FAIL the public gate")
	}
}

// administrableCount is the count the gate actually consults.
func administrableCount(t *testing.T, app core.App) int64 {
	t.Helper()
	n, err := CountAdministrableSuperusers(app)
	if err != nil {
		t.Fatalf("CountAdministrableSuperusers: %v", err)
	}
	return n
}

// emptySuperusers removes EVERY superuser row, placeholder included, so a test can build the exact
// store state it wants from scratch. It goes through raw SQL rather than app.Delete because the
// record layer refuses to remove the last superuser — which is exactly the state under test (a
// store the operator has never provisioned).
//
// Deliberately NOT used alone by the placeholder regression above: wiping the placeholder is what
// would make that test pass vacuously, so it seeds the placeholder back explicitly.
func emptySuperusers(t *testing.T, app core.App) {
	t.Helper()
	if _, err := app.DB().NewQuery("DELETE FROM {{_superusers}}").Execute(); err != nil {
		t.Fatalf("empty superusers: %v", err)
	}
	n, err := app.CountRecords(core.CollectionNameSuperusers)
	if err != nil || n != 0 {
		t.Fatalf("superusers not emptied: count=%d err=%v", n, err)
	}
}

// seedInstallerPlaceholder writes the throwaway row the dependency's first-run installer creates,
// reproducing the state a single ordinary loopback boot leaves behind.
func seedInstallerPlaceholder(t *testing.T, app core.App) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("find superusers: %v", err)
	}
	rec := core.NewRecord(col)
	rec.SetEmail(core.DefaultInstallerEmail)
	rec.SetRandomPassword()
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed installer placeholder: %v", err)
	}
}
