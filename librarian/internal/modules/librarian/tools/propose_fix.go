package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/librarian/desklib"
	"github.com/example/pocket-librarian/templates"
)

// ProposeFix — §5.3: load the ignore list FIRST (fail closed via desklib.Ignored), then for
// each flagged mechanical R1/R2/R3 finding run the guards in EXACT order
// (ignore → missing → read → staleness → plan → RECORD-ORIGINAL-FIRST) and create a
// revisions row. No filesystem write. A failed original-record is tolerated per-file (an
// "error" outcome that records NO revision row) and the batch continues — the safety
// boundary holds because no filesystem write may ever follow a finding without a revision.
func ProposeFix(ctx context.Context, app core.App, cfg *config.Config, in *ProposeFixInput) (*ProposeFixResult, error) {
	result := &ProposeFixResult{RunID: in.RunID, Proposed: []ProposedFix{}}

	ruleSet := effectiveRuleSet(in.Rules)

	filter := "state = 'flagged' && severity = 'mechanical'"
	params := dbx.Params{}
	if in.RunID != "" {
		filter += " && patrol_run = {:run}"
		params["run"] = in.RunID
	}
	findings, err := app.FindRecordsByFilter("patrol_findings", filter, "", 0, 0, params)
	if err != nil {
		return nil, fmt.Errorf("propose_fix: load findings: %w", err)
	}

	type candidate struct {
		finding *core.Record
		file    *core.Record
		path    string
	}
	var candidates []candidate
	for _, f := range findings {
		rule := f.GetString("rule")
		if !ruleSet[rule] {
			continue
		}
		var fileRec *core.Record
		if fid := f.GetString("file"); fid != "" {
			if fr, ferr := app.FindRecordById("files", fid); ferr == nil {
				fileRec = fr
			}
		}
		path := ""
		if fileRec != nil {
			path = fileRec.GetString("path")
		}
		candidates = append(candidates, candidate{finding: f, file: fileRec, path: path})
	}
	sort.Slice(candidates, func(i, j int) bool {
		ri, rj := candidates[i].finding.GetString("rule"), candidates[j].finding.GetString("rule")
		if ri != rj {
			return ri < rj
		}
		return candidates[i].path < candidates[j].path
	})

	// Load the ignore list FIRST, ahead of any read/plan (§10.1). If it is absent or
	// unreadable, fail closed: abort the whole operation and return an `ignored`
	// outcome for every candidate — never proceed with an empty ignore list.
	ignoreList, ignoreErr := desklib.LoadIgnoreList(cfg.IgnoreConfig)
	if ignoreErr != nil {
		for _, c := range candidates {
			result.Proposed = append(result.Proposed, ProposedFix{
				FindingID: c.finding.Id,
				Path:      c.path,
				Rule:      c.finding.GetString("rule"),
				Outcome:   "ignored",
			})
		}
		return result, nil
	}

	// Resolve the revisions collection ONCE, ahead of the loop, rather than per-file inside
	// proposeOne (it is a pure schema lookup, invariant across candidates). A failure here is
	// carried into proposeOne so its step-6 record-original guard reports a per-file "error"
	// — earlier guards (ignore/missing/stale/noop) still resolve normally.
	revCol, revColErr := app.FindCollectionByNameOrId("revisions")

	for _, c := range candidates {
		// Per-file tolerance: a failure recording one file's original (e.g. a store error)
		// yields an "error" outcome for that finding and the run continues to the rest —
		// one bad file never aborts the batch. The safety boundary is unweakened: an errored
		// finding records NO revision row, so no filesystem write can ever follow it.
		result.Proposed = append(result.Proposed, proposeOne(app, cfg, c.finding, c.file, c.path, ignoreList, in.RunID, revCol, revColErr))
	}
	return result, nil
}

// fixableRules is FIXABLE_RULES (spec §5.2/§5.3): the mechanical rules that have a fixer.
var fixableRules = map[string]bool{"R1": true, "R2": true, "R3": true}

// effectiveRuleSet intersects the caller's optional rule filter with FIXABLE_RULES; an
// empty filter defaults to the full fixable set.
func effectiveRuleSet(requested []string) map[string]bool {
	if len(requested) == 0 {
		out := make(map[string]bool, len(fixableRules))
		for k, v := range fixableRules {
			out[k] = v
		}
		return out
	}
	out := map[string]bool{}
	for _, r := range requested {
		if fixableRules[r] {
			out[r] = true
		}
	}
	return out
}

// proposeOne runs the per-finding guard chain (§5.3, EXACT order): ignore check FIRST,
// then missing, read+checksum, staleness, plan, record-original-first. It never returns an
// error: a failure at any step is folded into an "error" outcome so the caller can continue
// the batch. An "error" (or any non-"recorded") outcome creates NO revision row, preserving
// the boundary that no filesystem write may ever follow a failed original-record.
func proposeOne(app core.App, cfg *config.Config, finding, fileRec *core.Record, path string, ignoreList []string, runID string, revCol *core.Collection, revColErr error) ProposedFix {
	rule := finding.GetString("rule")
	out := ProposedFix{FindingID: finding.Id, Path: path, Rule: rule}

	// 1. Ignore check FIRST. No read, no write.
	if desklib.IsIgnored(path, ignoreList) {
		out.Outcome = "ignored"
		return out
	}

	// 2. File must exist as a regular file on disk.
	if fileRec == nil {
		out.Outcome = "missing"
		return out
	}
	abs := filepath.Join(cfg.DeskRoot, path)
	fi, statErr := os.Stat(abs)
	if statErr != nil || !fi.Mode().IsRegular() {
		out.Outcome = "missing"
		return out
	}

	// 3. Read the original: raw bytes + checksum.
	raw, rerr := os.ReadFile(abs)
	if rerr != nil {
		out.Outcome = "missing"
		return out
	}
	checksum := desklib.Checksum(raw)

	// 4. Staleness guard: finding.checksum vs the current on-disk checksum.
	findingChecksum := finding.GetString("checksum")
	if findingChecksum != "" && checksum != findingChecksum {
		out.Outcome = "stale"
		return out
	}

	// 5. Compute the plan (pure function of rule + file record + original bytes).
	plan, perr := computePlan(cfg, rule, fileRec, raw)
	if perr != nil {
		out.Outcome = "error"
		out.Error = fmt.Sprintf("compute plan for %s: %v", path, perr)
		return out
	}
	if plan == nil {
		out.Outcome = "noop"
		return out
	}
	out.Action = plan.Action
	out.NewPath = plan.NewPath

	// 6. RECORD ORIGINAL FIRST (Boundary 1, decision 0014). If this create fails, report a
	// per-file error and record NOTHING — no filesystem write may ever follow a failed
	// original-record, and the caller carries on with the remaining findings.
	if revColErr != nil {
		out.Outcome = "error"
		out.Error = fmt.Sprintf("revisions collection for %s: %v", path, revColErr)
		return out
	}
	rev := core.NewRecord(revCol)
	rev.Set("path", path)
	rev.Set("action", plan.Action)
	rev.Set("original_content", string(raw))
	rev.Set("original_checksum", checksum)
	rev.Set("new_path", plan.NewPath)
	rev.Set("finding", finding.Id)
	rev.Set("applied", false)
	rev.Set("restored", false)
	rev.Set("run_id", runID)
	if err := app.Save(rev); err != nil {
		out.Outcome = "error"
		out.Error = fmt.Sprintf("record original for %s: %v", path, err)
		return out
	}
	out.RevisionID = rev.Id
	out.Outcome = "recorded"
	return out
}

// --- planners (ported verbatim, §5.3) ---

// universalFMKeys (UNIVERSAL_FM_KEYS, §5.2) is declared in patrol.go — shared package-level.

// reverseTypeMap is REVERSE_TYPE_MAP (§5.3, exact): dir_kind label -> frontmatter type.
// Independent of TYPE_DIR_MAP (§5.2 R3), which maps type -> the configured directory PATH.
var reverseTypeMap = map[string]string{
	"decisions": "decision",
	"tasks":     "task",
	"analyses":  "analysis",
	"journal":   "journal",
}

var journalNameRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-.+\.md$`)
var slugNonAlnumRE = regexp.MustCompile(`[^a-z0-9]+`)

// fixPlan is the materialized plan a planner computes: either an "edit" (NewContent
// replaces the file in place) or a "move" (NewPath is the destination; Stub, if set, is
// left at the old path).
type fixPlan struct {
	Action     string // edit | move
	NewPath    string
	NewContent string
	Stub       string
}

// computePlan dispatches to the rule's planner. Planners are pure functions of
// (rule, file record, original bytes) so apply_fix can re-derive the same plan
// deterministically (spec §5.3 implementation note; default: re-derive, no proposed_*
// fields on the revisions row).
func computePlan(cfg *config.Config, rule string, fileRec *core.Record, original []byte) (*fixPlan, error) {
	switch rule {
	case "R1":
		return planR1(cfg, fileRec, original), nil
	case "R2":
		return planR2(cfg, fileRec), nil
	case "R3":
		return planR3(cfg, fileRec), nil
	default:
		return nil, fmt.Errorf("no planner for rule %s", rule)
	}
}

// planR1 — insert only the missing universal frontmatter keys before the closing fence;
// if there is no valid fence, prepend the whole template block. Only type/created/updated
// are computed; tags/synopsis are fixed template literals (spec §5.3).
func planR1(cfg *config.Config, fileRec *core.Record, original []byte) *fixPlan {
	text := string(original)
	fm := desklib.ParseFrontmatter(text)

	var missing []string
	for _, k := range universalFMKeys {
		if _, ok := fm[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	path := fileRec.GetString("path")
	created := gitDateOnly(desklib.GitOrigin(cfg.DeskRoot, path))
	if created == "" {
		created = today()
	}
	updated := gitDateOnly(desklib.GitLastCommit(cfg.DeskRoot, path))
	if updated == "" {
		updated = created
	}
	typ := reverseTypeMap[fileRec.GetString("dir_kind")]
	if typ == "" {
		typ = "note"
	}

	block := templates.Render(templates.FrontmatterUniversal, map[string]string{
		"type": typ, "created": created, "updated": updated,
	})
	blockLines := strings.Split(strings.Trim(block, "\n"), "\n") // ---, k: v, ..., ---
	keyLines := map[string]string{}
	for _, line := range blockLines[1 : len(blockLines)-1] {
		key, _, _ := strings.Cut(line, ":")
		keyLines[strings.TrimSpace(key)] = line
	}

	var newText string
	if len(fm) > 0 {
		// Existing valid fence: insert the missing key-lines before the close.
		lines := strings.Split(text, "\n")
		closeIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				closeIdx = i
				break
			}
		}
		if closeIdx == -1 {
			// Should not happen if fm is non-empty, but fall back safely.
			newText = strings.Join(blockLines, "\n") + "\n" + text
		} else {
			insert := make([]string, 0, len(missing))
			for _, k := range missing {
				insert = append(insert, keyLines[k])
			}
			newLines := make([]string, 0, len(lines)+len(insert))
			newLines = append(newLines, lines[:closeIdx]...)
			newLines = append(newLines, insert...)
			newLines = append(newLines, lines[closeIdx:]...)
			newText = strings.Join(newLines, "\n")
		}
	} else {
		// No (valid) frontmatter: prepend the whole template block.
		newText = strings.Join(blockLines, "\n") + "\n" + text
	}
	return &fixPlan{Action: "edit", NewContent: newText}
}

// planR2 — rename a journal file to <dir>/<date>-<slug>.md; never clobber an existing
// destination (spec §5.3).
func planR2(cfg *config.Config, fileRec *core.Record) *fixPlan {
	path := fileRec.GetString("path")
	name := filepath.Base(path)
	if journalNameRE.MatchString(name) {
		return nil
	}
	date := gitDateOnly(desklib.GitOrigin(cfg.DeskRoot, path))
	if date == "" {
		date = today()
	}
	stem := strings.TrimSuffix(name, ".md")
	slug := strings.Trim(slugNonAlnumRE.ReplaceAllString(strings.ToLower(stem), "-"), "-")
	if slug == "" {
		slug = "entry"
	}
	newPath := filepath.Join(filepath.Dir(path), fmt.Sprintf("%s-%s.md", date, slug))
	if _, err := os.Stat(filepath.Join(cfg.DeskRoot, newPath)); err == nil {
		return nil // never clobber
	}
	return &fixPlan{Action: "move", NewPath: newPath}
}

// planR3 — move the file to <expected>/<basename> and leave a pointer stub at the old
// path; never clobber an existing destination. expected is the configured entity-dir PATH
// (TYPE_DIR_MAP, spec §5.2 R3), and "under it" uses the same prefix test as R3 detection —
// NOT a dir_kind-label equality check (dir_kind is a fixed label; TYPE_DIR_MAP values are
// configurable paths that need not match it syntactically even when correctly placed).
func planR3(cfg *config.Config, fileRec *core.Record) *fixPlan {
	doctype := fileRec.GetString("doctype")
	expected := cfg.EntityDirMap()[doctype]
	path := fileRec.GetString("path")
	if expected == "" || isUnderDir(path, expected) {
		return nil
	}
	newPath := expected + "/" + filepath.Base(path)
	if _, err := os.Stat(filepath.Join(cfg.DeskRoot, newPath)); err == nil {
		return nil // never clobber
	}
	stub := templates.Render(templates.PointerStub, map[string]string{
		"date": today(), "new_path": newPath,
	})
	return &fixPlan{Action: "move", NewPath: newPath, Stub: stub}
}

// isUnderDir reports whether rel == dir or rel is inside dir (same prefix test as
// dir_kind_for, spec §5.1/§5.2 R3).
func isUnderDir(rel, dir string) bool {
	dir = strings.TrimSuffix(dir, "/")
	return rel == dir || strings.HasPrefix(rel, dir+"/")
}

// gitDateOnly extracts the yyyy-mm-dd half of a "<hash>|<date>" desklib git-meta string
// ("" on either an empty input or a malformed one).
func gitDateOnly(meta string) string {
	if meta == "" {
		return ""
	}
	_, date, ok := strings.Cut(meta, "|")
	if !ok {
		return ""
	}
	return date
}

func today() string { return time.Now().Format("2006-01-02") }
