package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
)

// DisposeResult reports the outcome of a disposition change on a single patrol finding.
type DisposeResult struct {
	ID          string `json:"id"`
	Disposition string `json:"disposition"`
	Message     string `json:"message"`
}

// dispositionValues is the disposition lifecycle enum, ORTHOGONAL to a finding's `state`
// (flagged/dismissed/fixed/resolved): 'open' is the live default; acknowledged/triaged/wont_fix
// are the disposed states hidden from the default `query findings` view.
var dispositionValues = map[string]bool{
	"open":         true,
	"acknowledged": true,
	"triaged":      true,
	"wont_fix":     true,
}

// DisposeFinding sets a patrol finding's disposition and NOTHING else — the finding's `state`
// is deliberately untouched, so a disposed finding stays `flagged` and survives re-patrol
// (the next patrol dedupes the row on (path, rule, checksum) and its disposition rides along).
//
// It accepts the raw --as value, normalizes the CLI-friendly "wont-fix" to the stored
// "wont_fix", validates against {open, acknowledged, triaged, wont_fix}, and errors on an
// unknown finding id or an invalid disposition. ctx/cfg are part of the frozen tool signature
// (parity with the other tools) though this store-only write needs neither.
func DisposeFinding(ctx context.Context, app core.App, cfg *config.Config, findingID, disposition string) (*DisposeResult, error) {
	d := normalizeDisposition(disposition)
	if !dispositionValues[d] {
		return nil, fmt.Errorf("dispose: invalid disposition %q (one of: open acknowledged triaged wont_fix)", disposition)
	}

	rec, err := app.FindRecordById("patrol_findings", findingID)
	if err != nil {
		return nil, fmt.Errorf("dispose: unknown finding %q: %w", findingID, err)
	}

	rec.Set("disposition", d)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("dispose: save finding %q: %w", findingID, err)
	}

	return &DisposeResult{
		ID:          rec.Id,
		Disposition: d,
		Message:     fmt.Sprintf("finding %s disposition set to %s", rec.Id, d),
	}, nil
}

// normalizeDisposition trims surrounding whitespace and maps the hyphenated CLI spelling
// "wont-fix" onto the stored underscore form "wont_fix". Every other value passes through
// unchanged for the caller's enum validation to accept or reject.
func normalizeDisposition(raw string) string {
	d := strings.TrimSpace(raw)
	if d == "wont-fix" {
		d = "wont_fix"
	}
	return d
}
