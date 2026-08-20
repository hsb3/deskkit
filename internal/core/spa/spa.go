// Package spa serves the embedded single-page web UI — the binary's visual surface — at `/`
// on the embedded PocketBase serve (admin console stays at `/_/`, data API at `/api/`). The
// SPA is a read-plus-chat surface: browse screens read the store through PocketBase's own
// REST API with the operator's superuser token, and the chat screen drives the same
// stewardship session endpoints the pre-SPA standalone page used (see the sibling web
// package). This package itself opens no write path; the one desk-content write the browser
// can make is the web package's write-through route (`/desk/doc/write`), which
// runs the same record-original-first path the CLI uses — the SPA never writes store rows
// or disk directly.
//
// The dist tree is built by the repo's frontend build (`make build` runs it) and is NEVER
// committed — only dist/.gitkeep is tracked, so a plain `go build` still compiles. A binary
// built without the frontend serves a small placeholder page instead of the app.
//
// Auth model. Every domain collection keeps nil (superuser-only) API rules, so the SPA
// always needs a superuser token:
//
//   - Loopback bind: GET /desk/bootstrap mints a token server-side, so the local operator
//     never sees a login screen. This matches the loopback posture of the chat surface —
//     safety comes from the loopback bind (local, on-demand, single-operator), and the
//     origin guard closes the browser cross-site vector (a page on another origin cannot
//     read the token: a cross-origin fetch carries an Origin header and is refused).
//   - Public bind: the bootstrap route is NOT REGISTERED at all — fail closed by absence,
//     not by an in-handler check that could be bypassed. The SPA shows a login form and
//     authenticates against the superusers auth collection through the SDK.
//
// The static shell itself is served without auth in both modes, the same stance as the
// dependency's own admin console shell: the shell loads, the data behind it does not —
// every API call still requires a token in public mode. The shell must load
// unauthenticated for the login form to render at all.
package spa

import (
	"io/fs"
	"net/http"
	"net/url"

	"embed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// PathBootstrap is the loopback-only token-mint endpoint the SPA calls on load.
const PathBootstrap = "/desk/bootstrap"

// PathAdmin is the path operators type for the admin console; the dependency serves it at
// adminConsole. Without an explicit route the shell wildcard below swallows /admin and
// renders the SPA instead, which reads as "the console is gone" rather than "wrong path".
const PathAdmin = "/admin"

// adminConsole is the dependency's own admin console mount. Trailing slash: the console's
// asset URLs are relative, so redirecting to /_ (no slash) costs a second hop.
const adminConsole = "/_/"

//go:embed all:dist
var distFS embed.FS

// placeholderHTML is served when the binary was compiled without a frontend build (dist holds
// only the tracked .gitkeep). It keeps a bare `go build` binary honest about what it carries.
const placeholderHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>deskkit</title></head>
<body style="font-family:sans-serif;max-width:38rem;margin:4rem auto;line-height:1.5">
<h1>deskkit</h1>
<p>The web UI was not built into this binary. Build the binary with <code>make build</code>
(which runs the frontend build first), then restart <code>serve</code>.</p>
</body></html>`

// Register mounts the SPA on the serve router: the static shell on the root wildcard, the two
// small read endpoints the settings panel needs (the model catalog, open in both modes; the
// resolved-config report, superuser-gated in public mode), and — on a loopback bind only — the
// token bootstrap route. public mirrors the caller's derived exposure mode (see cmd/deskkit's
// isPublicBind).
func Register(r *router.Router[*core.RequestEvent], public bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// The dist directory is committed (via .gitkeep), so this cannot happen on a real build.
		panic("spa: embedded dist missing: " + err.Error())
	}
	// Registered before the wildcard and matched ahead of it: a literal path beats
	// /{path...} in the dependency's router, so this is the one /admin handler.
	r.GET(PathAdmin, func(e *core.RequestEvent) error {
		return e.Redirect(http.StatusFound, adminConsole)
	})
	r.GET(PathModels, models) // registered in both bind modes: no secrets, needed before login
	// Same posture as the catalog, and for the same reason: the doc-type vocabulary describes
	// the schema the binary ships, not this desk's contents, so it is open in both modes.
	r.GET(PathDoctypes, doctypes)
	// Registered in both modes, but gated in public: unlike the catalog this describes THIS
	// desk's configuration, so a public bind narrows it to the operator's own credential —
	// superusers only, not the member auth collection the chat surface also accepts.
	resolved := r.GET(PathSettingsResolved, settingsResolved)
	if public {
		resolved.Bind(apis.RequireAuth(core.CollectionNameSuperusers))
	}
	if _, statErr := fs.Stat(sub, "index.html"); statErr == nil {
		// indexFallback=true: unknown paths serve the shell, so the SPA's client-side routes
		// (and the pre-SPA /desk/chat URL) all land in the app.
		r.GET("/{path...}", apis.Static(sub, true))
	} else {
		r.GET("/{path...}", func(e *core.RequestEvent) error {
			return e.HTML(http.StatusOK, placeholderHTML)
		})
	}
	if !public {
		r.GET(PathBootstrap, bootstrap)
	}
}

// bootstrap mints a superuser auth token for the SPA. Loopback-only by registration (see
// Register); the origin guard below is the same browser cross-site defense the chat surface
// uses, so a page on a non-loopback origin cannot fetch the token even from the same machine.
//
// It prefers an administrable superuser account; a store that has only the dependency's
// first-run installer placeholder row (an ordinary local serve creates it) still gets a
// token — on a loopback bind that is equivalent to today's posture, where the unauthenticated
// chat surface already drives store-reading tools without any account.
func bootstrap(e *core.RequestEvent) error {
	if !loopbackOrigin(e.Request) {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "cross-origin request rejected: this surface accepts only same-origin browser requests",
		})
	}
	rec, err := findTokenRecord(e.App)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "no superuser account exists to mint a session for: " + err.Error(),
		})
	}
	token, err := rec.NewAuthToken()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, map[string]string{"token": token})
}

// findTokenRecord picks the superuser record to mint for: any administrable account first
// (excluding the installer placeholder, mirroring store.CountAdministrableSuperusers), then
// the placeholder as the empty-store fallback.
func findTokenRecord(app core.App) (*core.Record, error) {
	recs, err := app.FindAllRecords(core.CollectionNameSuperusers,
		dbx.Not(dbx.HashExp{"email": core.DefaultInstallerEmail}))
	if err == nil && len(recs) > 0 {
		return recs[0], nil
	}
	recs, err = app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, &noSuperuserError{}
	}
	return recs[0], nil
}

type noSuperuserError struct{}

func (*noSuperuserError) Error() string { return "the store holds no superuser records" }

// loopbackOrigin is the browser cross-site guard for the bootstrap route: a request with no
// Origin header (curl, same-origin navigation) passes; a present Origin must be a loopback
// host. Mirrors the loopback branch of the chat surface's origin guard — kept local because
// this route only ever exists on a loopback bind, so the public same-origin branch never
// applies.
func loopbackOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}
