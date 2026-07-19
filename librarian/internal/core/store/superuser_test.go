package store

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/example/pocket-librarian/internal/core/config"

	// Blank-import registers this project's Go migrations so tests.NewTestApp applies them.
	_ "github.com/example/pocket-librarian/internal/modules/librarian/collections"
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

func TestEnsureSuperuser_NoopWhenUnset(t *testing.T) {
	app := newBootstrapApp(t)
	before := len(func() []*core.Record {
		recs, _ := app.FindAllRecords(core.CollectionNameSuperusers)
		return recs
	}())

	cases := []*config.Config{
		nil,
		{},
		{PBSuperuserEmail: testSuperEmail},   // password missing
		{PBSuperuserPassword: testSuperPass}, // email missing
	}
	for _, cfg := range cases {
		created, err := EnsureSuperuser(app, cfg)
		if err != nil {
			t.Fatalf("EnsureSuperuser must not error when env is unset: %v", err)
		}
		if created {
			t.Fatalf("expected created=false when either var is unset")
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
