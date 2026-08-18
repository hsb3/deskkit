package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// isLoopbackAddr classifies ONE resolved listen address as loopback-only or publicly exposed.
// It is the atom of "public mode": everything the binary hardens when it is reachable from off-box
// is derived from the exposure itself, never from a separate opt-in flag. A flag can be forgotten
// while still binding 0.0.0.0 — which fails OPEN — whereas a mode derived from the bind address
// cannot be forgotten. Callers classify the FULL address set via isPublicBind, not this alone.
//
// Rules, in order:
//   - empty  -> loopback. PocketBase's own --http default ("127.0.0.1:8090") is applied inside
//     the dependency's serve command, so an empty value here means "the dependency's default",
//     which is loopback. (An empty value is also what a programmatic Serve with no addr yields.)
//   - a bare ":8090" (no host) -> PUBLIC: an empty host binds every interface.
//   - "localhost" -> loopback.
//   - a parseable IP -> whatever net.IP.IsLoopback says, so the whole 127.0.0.0/8 range and ::1
//     both classify as loopback and 0.0.0.0 / :: / any routable address do not.
//   - anything else (a hostname we cannot prove is loopback) -> PUBLIC. Fail closed: an
//     unresolvable name is treated as exposure, never as safety.
func isLoopbackAddr(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return true
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// No port component (or a malformed one): treat the whole value as the host and let the
		// checks below decide. A genuinely malformed address falls through to the public default.
		host = addr
	}
	// SplitHostPort already strips the brackets of an "[::1]:8090" form; trim them anyway so a
	// bracketed host passed without a port ("[::1]") is handled identically.
	host = strings.Trim(host, "[]")

	if host == "" {
		return false // ":8090" — an empty host binds all interfaces
	}
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// effectiveServeAddrs mirrors the dependency's own address defaulting so BOTH listeners are
// classified, not just one.
//
// Why this is needed: the dependency sets its ServeEvent server address to the HTTPS address when
// --https is given, and only to the HTTP address otherwise. That single value — the only address
// visible on the serve event — therefore HIDES the other listener. `--https 127.0.0.1:8443 --http
// 0.0.0.0:8090` presents a loopback address on the event while genuinely binding 0.0.0.0. Today
// the extra listener only serves the certificate manager's ACME/redirect handler rather than the
// app router, so it is not a live route bypass, but "public mode is derived from the resolved bind
// address" must be true of every address, not the one that happens to be reported.
//
// The defaulting rules are copied from the dependency's serve command and must stay in sync with
// it (re-audit on a dependency bump; the table test pins the current behavior):
//   - with certificate-domain positional args: an empty --http means 0.0.0.0:80 and an empty
//     --https means 0.0.0.0:443 — both public.
//   - without them: an empty --http means 127.0.0.1:8090, and an empty --https means NO TLS
//     listener at all (not an address to classify).
func effectiveServeAddrs(httpAddr, httpsAddr string, hasDomainArgs bool) []string {
	if hasDomainArgs {
		if httpAddr == "" {
			httpAddr = "0.0.0.0:80"
		}
		if httpsAddr == "" {
			httpsAddr = "0.0.0.0:443"
		}
	} else if httpAddr == "" {
		httpAddr = "127.0.0.1:8090"
	}
	addrs := []string{httpAddr}
	if httpsAddr != "" {
		addrs = append(addrs, httpsAddr)
	}
	return addrs
}

// isPublicBind reports whether ANY of the addresses the server will listen on is non-loopback.
// One exposed listener makes the whole process public — the hardening is per-process (auth
// prerequisites, route auth, CORS), so it cannot be applied to half a server. Empty entries are
// skipped rather than treated as loopback, so a caller may pass a partially-known set safely.
func isPublicBind(addrs ...string) bool {
	for _, a := range addrs {
		if strings.TrimSpace(a) == "" {
			continue
		}
		if !isLoopbackAddr(a) {
			return true
		}
	}
	return false
}

// serveAddrsAndOrigins reads the invoked `serve` command's own parsed flags — the --http/--https
// addresses and the --origins allowlist — so the exposure decision sees the full picture.
//
// Cobra's parsed flag values are read here rather than argv being re-scanned: by the time OnServe
// fires, the serve command has parsed, so this is the authoritative value with no second parser to
// drift. serverAddr (the address the dependency put on the serve event) is always included, so an
// absent or unreadable serve command degrades to exactly the previous single-address behavior
// rather than to "assume loopback".
func serveAddrsAndOrigins(root *cobra.Command, serverAddr string) (addrs []string, origins []string) {
	addrs = []string{serverAddr}
	for _, c := range root.Commands() {
		if c.Name() != "serve" {
			continue
		}
		httpAddr, _ := c.Flags().GetString("http")
		httpsAddr, _ := c.Flags().GetString("https")
		// Positional args to `serve` are certificate domains, which is what flips the dependency's
		// address defaults to 0.0.0.0.
		addrs = append(addrs, effectiveServeAddrs(httpAddr, httpsAddr, len(c.Flags().Args()) > 0)...)
		origins, _ = c.Flags().GetStringSlice("origins")
		break
	}
	return addrs, origins
}

// isWildcardOrigins reports whether an --origins value leaves the door open to every origin, i.e.
// whether it is the dependency's own default rather than an operator-chosen allowlist.
//
// Empty counts as wildcard because the dependency substitutes ["*"] for an empty list. A literal
// "*" entry counts even alongside other entries — this stays fail-closed (strip the middleware)
// for the mixed case, purely as defense in depth: on a public bind ValidatePublicOrigins refuses
// that input outright, so hardenPublicCORS never legitimately sees it. If the two are ever
// reordered, this direction errs toward removing headers rather than serving a live wildcard.
//
// A glob PATTERN such as "https://*.example.com" is NOT a wildcard — the dependency treats those
// globs as a real constraint, and an operator who wrote one meant it.
func isWildcardOrigins(origins []string) bool {
	if len(origins) == 0 {
		return true
	}
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}

// splitOrigins separates an --origins value into its bare-wildcard entries and its explicit ones.
// Blank entries are ignored (a trailing comma is a typo, not an instruction).
func splitOrigins(origins []string) (wildcards, explicit []string) {
	for _, o := range origins {
		switch t := strings.TrimSpace(o); t {
		case "":
			// skip
		case "*":
			wildcards = append(wildcards, t)
		default:
			explicit = append(explicit, t)
		}
	}
	return wildcards, explicit
}

// ValidatePublicOrigins refuses a self-contradictory --origins value on a public bind: a bare "*"
// mixed with explicit origins.
//
// `--origins "*,https://frontend.example"` is a CONTRADICTORY SECURITY INSTRUCTION, not a
// preference to resolve. Honoring the wildcard ignores the allowlist; honoring the allowlist
// ignores the wildcard. Either way half of an operator's explicit input at a trust boundary is
// silently discarded — the exact defect class this binary has now been bitten by twice (a
// half-configured PB_SUPERUSER_* pair that quietly disabled auth; a wildcard CORS policy that
// survived unnoticed because nothing asserted on it). Two states only: it works, or it refuses and
// says why. The operator has to state what they meant.
//
// Loopback is deliberately exempt: local development is not a trust boundary and must not start
// failing over a flag combination that has always been tolerated.
func ValidatePublicOrigins(public bool, origins []string) error {
	if !public {
		return nil
	}
	wildcards, explicit := splitOrigins(origins)
	if len(wildcards) == 0 || len(explicit) == 0 {
		return nil
	}
	return fmt.Errorf(
		"refusing to serve on a non-loopback address: --origins mixes the wildcard %q with explicit "+
			"origin(s) %s, which contradict each other — the wildcard allows every origin the list "+
			"was written to exclude. Pass either --origins %q (allow all) or --origins %q (allow only "+
			"those), never both",
		"*", strings.Join(explicit, ", "), "*", strings.Join(explicit, ","))
}

// hardenPublicCORS removes the dependency's DEFAULT WILDCARD CORS middleware when the server is
// publicly bound.
//
// The dependency binds `CORS(CORSConfig{AllowOrigins: ["*"]})` onto the serve router itself
// (apis/serve.go: the AllowedOrigins default is "*" and the bind happens right after the router is
// built), which is BEFORE it triggers the OnServe hook — and that bind is the router this function
// receives, so removing the middleware here removes it from every route: the health endpoint, the
// record API, and this binary's own /desk/chat routes alike. It is a router-wide concern, which is
// why it lives here and not inside the web surface's Register.
//
// Removal, not replacement — but ONLY of the default wildcard. The session page is served from the
// same origin it posts to, so it needs no CORS headers at all, and with none emitted a browser
// refuses a cross-origin read by default. Inventing an allowlist here would be speculative
// configuration for a caller that does not exist.
//
// An operator-supplied `--origins` allowlist IS preserved (origins non-wildcard -> this is a
// no-op). The dependency binds that value into the very same middleware id, so an unconditional
// unbind silently threw away the one mechanism an operator has for legitimately serving a separate
// frontend origin. Discarding an explicit security setting without a word is its own defect, so the
// wildcard case is the only one removed.
//
// Loopback mode is deliberately untouched: the wildcard is the dependency's long-standing local
// behavior and local-dev tooling may rely on it. Only the exposed path hardens.
func hardenPublicCORS(r *router.Router[*core.RequestEvent], public bool, origins []string) {
	if !public || !isWildcardOrigins(origins) {
		return
	}
	r.Unbind(apis.DefaultCorsMiddlewareId)
}
