// Package bootstrap holds first-run, serve-time provisioning helpers that consume secrets
// from the environment (never from Config). It is the DB-facing analog of desklib's
// filesystem first-run auto-create (the .librarian-ignore boundary): idempotent, safe to
// run on every serve.
package store

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
)

// EnsureSuperuser creates the PocketBase superuser named by PB_SUPERUSER_EMAIL /
// PB_SUPERUSER_PASSWORD when BOTH are set (spec §10.3). It is idempotent: a superuser with
// that email already existing is a no-op — the existing credential is never overwritten.
// When either var is unset it does nothing and returns (false, nil): the identity-neutral
// binary never invents a credential (spec §10.3 decision — superuser-unset is non-fatal).
//
// The password value is read from cfg (populated from the environment at Load time) and
// passed to SetPassword, which stores only a hash; the plaintext is never persisted.
func EnsureSuperuser(app core.App, cfg *config.Config) (created bool, err error) {
	if cfg == nil || cfg.PBSuperuserEmail == "" || cfg.PBSuperuserPassword == "" {
		return false, nil
	}

	col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return false, fmt.Errorf("bootstrap: fetch %q collection: %w", core.CollectionNameSuperusers, err)
	}

	// Idempotent: if the account already exists, do nothing (never re-set the password).
	if _, ferr := app.FindAuthRecordByEmail(col, cfg.PBSuperuserEmail); ferr == nil {
		return false, nil
	}

	rec := core.NewRecord(col)
	rec.SetEmail(cfg.PBSuperuserEmail)
	rec.SetPassword(cfg.PBSuperuserPassword)
	if serr := app.Save(rec); serr != nil {
		return false, fmt.Errorf("bootstrap: create superuser %q: %w", cfg.PBSuperuserEmail, serr)
	}
	return true, nil
}
