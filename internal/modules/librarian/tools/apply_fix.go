package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
)

// ApplyFix — §5.4: the write path. Reload the ignore list and re-run the ignore + staleness
// guards, re-derive the plan from (rec, original_content), write byte-exact
// (desklib.WriteExact / os.Rename), then patch revisions.applied + patrol_findings.state and
// append one adoption_log row. The ONLY tool that mutates the desk tree — its registration
// for the autonomous agent is gated by LIBRARIAN_AUTONOMOUS_WRITES (registry.AgentTools, §5.4).
func ApplyFix(ctx context.Context, app core.App, cfg *config.Config, in *ApplyFixInput) (*ApplyFixResult, error) {
	result := &ApplyFixResult{RunID: in.RunID, Outcomes: []ApplyOutcome{}}

	items, err := loadOutstandingRevisions(app, in)
	if err != nil {
		return nil, fmt.Errorf("apply_fix: load revisions: %w", err)
	}

	// 1. Reload the ignore list ahead of any write; re-run the ignore check. Fail closed:
	// if the ignore file is absent or unreadable, refuse the WHOLE batch (§10.1) — every
	// revision returns `ignored` and no filesystem write occurs.
	ignoreList, ignoreErr := desklib.LoadIgnoreList(cfg.IgnoreConfig)
	if ignoreErr != nil {
		for _, it := range items {
			result.Outcomes = append(result.Outcomes, ApplyOutcome{
				RevisionID: it.rev.Id, Path: it.rev.GetString("path"), Rule: it.rule(), Outcome: "ignored",
			})
		}
		if err := recordAdoptionLog(app, cfg, in.RunID, result.Outcomes); err != nil {
			return nil, fmt.Errorf("apply_fix: adoption log: %w", err)
		}
		return result, nil
	}

	for _, it := range items {
		result.Outcomes = append(result.Outcomes, applyOne(app, cfg, it, ignoreList))
	}

	if err := recordAdoptionLog(app, cfg, in.RunID, result.Outcomes); err != nil {
		return nil, fmt.Errorf("apply_fix: adoption log: %w", err)
	}
	return result, nil
}

// revItem pairs a revisions row with its (possibly nil) originating finding.
type revItem struct {
	rev     *core.Record
	finding *core.Record
}

func (it revItem) rule() string {
	if it.finding == nil {
		return ""
	}
	return it.finding.GetString("rule")
}

// loadOutstandingRevisions loads revisions where applied == false && restored == false,
// scoped by RunID or explicit RevisionIDs (spec §5.4).
func loadOutstandingRevisions(app core.App, in *ApplyFixInput) ([]revItem, error) {
	all, err := app.FindRecordsByFilter("revisions", "applied = false && restored = false", "+created", 0, 0)
	if err != nil {
		return nil, err
	}

	var filtered []*core.Record
	switch {
	case len(in.RevisionIDs) > 0:
		want := map[string]bool{}
		for _, id := range in.RevisionIDs {
			want[id] = true
		}
		for _, r := range all {
			if want[r.Id] {
				filtered = append(filtered, r)
			}
		}
	case in.RunID != "":
		for _, r := range all {
			if r.GetString("run_id") == in.RunID {
				filtered = append(filtered, r)
			}
		}
	default:
		filtered = all
	}

	items := make([]revItem, 0, len(filtered))
	for _, r := range filtered {
		var finding *core.Record
		if fid := r.GetString("finding"); fid != "" {
			if f, ferr := app.FindRecordById("patrol_findings", fid); ferr == nil {
				finding = f
			}
		}
		items = append(items, revItem{rev: r, finding: finding})
	}
	return items, nil
}

// applyOne runs the write-time guard chain (§5.4, EXACT order): re-check ignore, confirm
// the file still exists, re-run the staleness guard against finding.checksum, re-derive the
// plan deterministically, write byte-exact, then mark applied/fixed inside one transaction
// (the FS write happens OUTSIDE the DB transaction — see the half-applied-move note below).
func applyOne(app core.App, cfg *config.Config, it revItem, ignoreList []string) ApplyOutcome {
	rev := it.rev
	path := rev.GetString("path")
	out := ApplyOutcome{RevisionID: rev.Id, Path: path, Rule: it.rule()}

	// 1. Ignore check (defense in depth: the list may have changed since propose_fix).
	if desklib.IsIgnored(path, ignoreList) {
		out.Outcome = "ignored"
		return out
	}

	// 2. Confirm the file still exists.
	abs := filepath.Join(cfg.DeskRoot, path)
	fi, statErr := os.Stat(abs)
	if statErr != nil || !fi.Mode().IsRegular() {
		out.Outcome = "missing"
		return out
	}

	// 3. Re-run the staleness guard: current bytes vs finding.checksum (NOT
	// original_checksum) — the authority is the checksum recorded at flag time.
	current, rerr := os.ReadFile(abs)
	if rerr != nil {
		out.Outcome = "error"
		out.Error = rerr.Error()
		return out
	}
	if it.finding != nil {
		if fc := it.finding.GetString("checksum"); fc != "" && desklib.Checksum(current) != fc {
			out.Outcome = "stale"
			return out
		}
	}

	// 4. Re-derive the plan deterministically from (rec, original) — the file record
	// backing the finding, plus the CURRENT on-disk bytes (equal to the recorded original
	// bytes at this point, since staleness passed).
	if it.finding == nil {
		out.Outcome = "error"
		out.Error = "revision " + rev.Id + ": no originating finding"
		return out
	}
	var fileRec *core.Record
	if fid := it.finding.GetString("file"); fid != "" {
		if fr, ferr := app.FindRecordById("files", fid); ferr == nil {
			fileRec = fr
		}
	}
	if fileRec == nil {
		out.Outcome = "error"
		out.Error = "revision " + rev.Id + ": file record not found"
		return out
	}
	plan, perr := computePlan(cfg, it.rule(), fileRec, current)
	if perr != nil {
		out.Outcome = "error"
		out.Error = perr.Error()
		return out
	}
	if plan == nil {
		out.Outcome = "noop"
		return out
	}

	// 5. Write to the tree, byte-exact, no newline translation.
	switch plan.Action {
	case "edit":
		if err := desklib.WriteExact(abs, []byte(plan.NewContent)); err != nil {
			out.Outcome = "error"
			out.Error = err.Error()
			return out
		}
	case "move":
		destAbs := filepath.Join(cfg.DeskRoot, plan.NewPath)
		if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
			out.Outcome = "error"
			out.Error = err.Error()
			return out
		}
		if err := os.Rename(abs, destAbs); err != nil {
			out.Outcome = "error"
			out.Error = err.Error()
			return out
		}
		if plan.Stub != "" {
			if err := desklib.WriteExact(abs, []byte(plan.Stub)); err != nil {
				// Half-applied-move window (spec §5.4): the move already happened but the
				// stub write failed. Log loudly; `restore --by-path` can recover.
				app.Logger().Warn("half-applied move: stub write failed after rename",
					"revision", rev.Id, "path", path, "new_path", plan.NewPath, "err", err)
				out.Outcome = "error"
				out.Error = err.Error()
				return out
			}
		}
	}

	// 6. Mark applied + fixed, atomically. The FS write above is outside this transaction
	// (the filesystem is not transactional); if this patch fails after a successful write,
	// the revision stays applied=false — log a WARNING naming the revision + moved path so
	// `restore --by-path` can complete or roll it back (spec §5.4 half-applied recovery).
	txErr := app.RunInTransaction(func(txApp core.App) error {
		rev.Set("applied", true)
		if err := txApp.Save(rev); err != nil {
			return err
		}
		it.finding.Set("state", "fixed")
		return txApp.Save(it.finding)
	})
	if txErr != nil {
		movedPath := path
		if plan.Action == "move" {
			movedPath = plan.NewPath
		}
		app.Logger().Warn("half-applied write: DB patch failed after fs write",
			"revision", rev.Id, "path", movedPath, "err", txErr)
		out.Outcome = "error"
		out.Error = txErr.Error()
		return out
	}

	out.Outcome = "applied"
	return out
}

// recordAdoptionLog appends the single post-batch adoption_log row (spec §5.4): "run <id>:
// <outcome counts>", sorted by outcome name, or "nothing to fix" when there were none.
func recordAdoptionLog(app core.App, cfg *config.Config, runID string, outcomes []ApplyOutcome) error {
	col, err := app.FindCollectionByNameOrId("adoption_log")
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, o := range outcomes {
		counts[o.Outcome]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, counts[k]))
	}
	summary := strings.Join(parts, ", ")
	if summary == "" {
		summary = "nothing to fix"
	}

	rec := core.NewRecord(col)
	rec.Set("date", time.Now())
	rec.Set("desk", cfg.DeskName)
	rec.Set("event", "fix")
	rec.Set("detail", fmt.Sprintf("run %s: %s", runID, summary))
	return app.Save(rec)
}
