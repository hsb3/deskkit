// Package bootstrap holds first-run, serve-time provisioning helpers that consume secrets
// from the environment (never from Config). It is the DB-facing analog of desklib's
// filesystem first-run auto-create (the .librarian-ignore boundary): idempotent, safe to
// run on every serve.
package store

import (
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
)

// ErrHalfConfiguredSuperuser is returned when exactly ONE of PB_SUPERUSER_EMAIL /
// PB_SUPERUSER_PASSWORD is set. Both-unset stays the documented silent no-op (spec §10.3: the
// identity-neutral binary never invents a credential), but a half-set pair is an operator
// mistake that would otherwise disable the account silently — an auth environment that is
// almost right must fail loudly, never degrade quietly. Callers on the serve path treat it as
// fatal (see CheckServeAuthPrereqs).
var ErrHalfConfiguredSuperuser = errors.New(
	"half-configured superuser environment: set BOTH PB_SUPERUSER_EMAIL and PB_SUPERUSER_PASSWORD, or neither")

// ValidateSuperuserEnv reports the half-configured-environment error described on
// ErrHalfConfiguredSuperuser, naming which of the pair is missing. Both set and both unset are
// both fine here; "both unset" is only fatal on a publicly-bound serve (CheckServeAuthPrereqs).
func ValidateSuperuserEnv(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	switch {
	case cfg.PBSuperuserEmail != "" && cfg.PBSuperuserPassword == "":
		return fmt.Errorf("%w (PB_SUPERUSER_PASSWORD is empty)", ErrHalfConfiguredSuperuser)
	case cfg.PBSuperuserEmail == "" && cfg.PBSuperuserPassword != "":
		return fmt.Errorf("%w (PB_SUPERUSER_EMAIL is empty)", ErrHalfConfiguredSuperuser)
	}
	return nil
}

// CountAdministrableSuperusers counts the superusers an operator can actually log in as.
//
// It EXCLUDES core.DefaultInstallerEmail. That row is not an account: the dependency's first-run
// installer writes it purely to mint the one-time `/_/#/pbinstall/<token>` setup link, and it
// carries no password anyone holds. A plain count therefore reports "a superuser exists" for a
// store nobody can administer — and since an ordinary loopback boot creates that row, a naive
// count would let one local `deskkit serve` permanently disarm the public gate below. The
// dependency's own installer check excludes the same address for the same reason
// (apis/installer.go, needInstallerSuperuser); this mirrors it deliberately, so re-audit that
// function on a dependency bump.
func CountAdministrableSuperusers(app core.App) (int64, error) {
	return app.CountRecords(core.CollectionNameSuperusers, dbx.Not(dbx.HashExp{
		"email": core.DefaultInstallerEmail,
	}))
}

// CheckServeAuthPrereqs is the fail-closed auth gate for `serve`. It runs at the TOP of the
// OnServe hook — which the dependency triggers BEFORE it opens the tcp listener — so a refusal
// here means the process never binds a port. It reports whether it created the superuser, so the
// caller can log that one operator-visible fact.
//
// public is derived from the RESOLVED bind addresses (see cmd/deskkit's isPublicBind), not from
// a flag: a "--public"-style opt-in can be forgotten while still binding 0.0.0.0, which fails
// OPEN; deriving the mode from the exposure cannot be forgotten.
//
//   - Every mode: a half-configured PB_SUPERUSER_* pair is fatal.
//   - Public mode: the gate VERIFIES THE END STATE rather than predicting it. It provisions the
//     PB_SUPERUSER_* account here and now — treating a provisioning failure as fatal — and then
//     re-counts administrable superusers. Only a count > 0 lets serve proceed.
//
// The end-state check is the whole point. An earlier version passed the gate on the env vars
// merely being non-empty, trusting a LATER, non-fatal EnsureSuperuser call to succeed; a password
// the dependency rejects then failed into a log line while the public listener came up with no
// administrable account at all. A gate that predicts is not a gate.
//
// A nil cfg (config did not resolve) carries no PB_SUPERUSER_* values, so in public mode it can
// only pass if the store already holds a real superuser — the correct fail-closed reading.
func CheckServeAuthPrereqs(app core.App, cfg *config.Config, public bool) (created bool, err error) {
	if err := ValidateSuperuserEnv(cfg); err != nil {
		return false, err
	}
	if !public {
		return false, nil
	}

	// Provision NOW, fatally. Idempotent, so the later serve-path call is a no-op.
	created, err = EnsureSuperuser(app, cfg)
	if err != nil {
		return false, fmt.Errorf(
			"refusing to serve on a non-loopback address: cannot provision the superuser named by "+
				"PB_SUPERUSER_EMAIL/PB_SUPERUSER_PASSWORD: %w", err)
	}

	// Then verify what actually landed, rather than what was expected to.
	n, cerr := CountAdministrableSuperusers(app)
	if cerr != nil {
		return created, fmt.Errorf(
			"cannot verify that a superuser exists before serving on a non-loopback address: %w", cerr)
	}
	if n > 0 {
		return created, nil
	}
	return created, errors.New(
		"refusing to serve on a non-loopback address with no superuser: the store holds no " +
			"administrable _superusers record (the first-run installer placeholder does not count) " +
			"and PB_SUPERUSER_EMAIL/PB_SUPERUSER_PASSWORD did not produce one. " +
			"Set both env vars (or create a superuser with `deskkit superuser create`), or bind " +
			"a loopback address (--http 127.0.0.1:<port>)")
}

// EnsureSuperuser creates the PocketBase superuser named by PB_SUPERUSER_EMAIL /
// PB_SUPERUSER_PASSWORD when BOTH are set (spec §10.3). It is idempotent: a superuser with
// that email already existing is a no-op — the existing credential is never overwritten.
// Upserting is deliberately NOT done: re-setting the password rotates tokenKey, which 401s
// every live token (incident, a sibling project).
//
// When BOTH vars are unset it does nothing and returns (false, nil): the identity-neutral
// binary never invents a credential (spec §10.3 decision — superuser-unset is non-fatal).
// When exactly ONE is set it returns ErrHalfConfiguredSuperuser rather than silently no-opping;
// see that variable for why.
//
// The password value is read from cfg (populated from the environment at Load time) and
// passed to SetPassword, which stores only a hash; the plaintext is never persisted.
func EnsureSuperuser(app core.App, cfg *config.Config) (created bool, err error) {
	if err := ValidateSuperuserEnv(cfg); err != nil {
		return false, err
	}
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
