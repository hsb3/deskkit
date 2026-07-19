package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/librarian/desklib"
)

// Patrol — §5.2: run rules R1–R6 over the non-deleted files (filtered by Path), file new
// patrol_findings (dedupe on (path, rule, checksum)) + one patrol_log row. Never writes the
// filesystem. R6 (HANDOFF staleness) runs only when cfg.HandoffPath is in the filtered set.
// ISSUE_REF_RE's lookbehind is reimplemented in sweep.go (RE2 has no lookbehind, §5.2).
func Patrol(ctx context.Context, app core.App, cfg *config.Config, in *PatrolInput) (*PatrolResult, error) {
	root := cfg.DeskRoot
	dirMap := cfg.EntityDirMap()
	runID := "patrol-" + time.Now().UTC().Format("20060102T150405Z")

	result := &PatrolResult{RunID: runID, ByRule: map[string]int{}}

	txErr := app.RunInTransaction(func(txApp core.App) error {
		allFileRecs, err := txApp.FindRecordsByFilter("files", "deleted = false", "", 0, 0, dbx.Params{})
		if err != nil {
			return err
		}
		allFiles := fileRowsFromRecords(allFileRecs)

		var filtered []fileRow
		if in.Path == "" {
			filtered = allFiles
		} else {
			for _, row := range allFiles {
				if pathOrSubtree(row.Path, in.Path) {
					filtered = append(filtered, row)
				}
			}
		}
		result.FilesSwept = len(filtered)

		// open findings -> dedupe keys (path, rule, checksum), keyed off the finding's OWN
		// stored checksum (checksum at flag time), not the file's current checksum.
		openFindingRecs, err := txApp.FindRecordsByFilter("patrol_findings", "state = 'flagged'", "", 0, 0, dbx.Params{})
		if err != nil {
			return err
		}
		filesByID := make(map[string]fileRow, len(allFiles))
		for _, row := range allFiles {
			filesByID[row.ID] = row
		}
		openKeys := make(map[findingKey]bool, len(openFindingRecs))
		openFindings := make([]openFinding, 0, len(openFindingRecs))
		for _, f := range openFindingRecs {
			p := ""
			if row, ok := filesByID[f.GetString("file")]; ok {
				p = row.Path
			}
			rule := f.GetString("rule")
			openKeys[findingDedupeKey(p, rule, f.GetString("checksum"))] = true
			openFindings = append(openFindings, openFinding{rec: f, path: p, rule: rule})
		}

		// fired records every (path, rule) whose rule detection hit this run — recorded BEFORE
		// the dedupe check so a still-firing finding that dedupes still counts as fired (and so
		// is never resolved). Any open finding whose (path, rule) is absent here did not re-fire.
		fired := make(map[ruleKey]bool)

		findingsCollection, err := txApp.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return err
		}

		fileFinding := func(row fileRow, rule, severity, detail, proposedFix string) error {
			fired[ruleKey{row.Path, rule}] = true
			if isDuplicateFinding(openKeys, row.Path, rule, row.Checksum) {
				return nil
			}
			fr := core.NewRecord(findingsCollection)
			fr.Set("file", row.ID)
			fr.Set("rule", rule)
			fr.Set("severity", severity)
			fr.Set("detail", detail)
			fr.Set("proposed_fix", proposedFix)
			fr.Set("state", "flagged")
			fr.Set("patrol_run", runID)
			fr.Set("checksum", row.Checksum)
			if err := txApp.Save(fr); err != nil {
				return err
			}
			result.FindingsNew++
			result.ByRule[rule]++
			return nil
		}

		for _, rule := range mechanicalRules {
			for _, row := range filtered {
				detail, fix, hit := runMechanicalCheck(root, rule, row, dirMap)
				if hit {
					if err := fileFinding(row, rule, "mechanical", detail, fix); err != nil {
						return err
					}
				}
			}
		}
		// R4 is JUDGMENT (spec §5.2): its detection is mechanical (an invalid/empty decision
		// status), but choosing WHICH of {proposed, accepted, rejected, superseded} an empty
		// status should become is a semantic call the librarian cannot make. So propose_fix
		// never auto-fixes R4 (it is absent from FIXABLE_RULES) and patrol files it as judgment,
		// not mechanical — a supervisor picks the value.
		for _, row := range filtered {
			if detail, fix, hit := checkR4(row); hit {
				if err := fileFinding(row, "R4", "judgment", detail, fix); err != nil {
					return err
				}
			}
		}
		for _, row := range filtered {
			if detail, fix, hit := checkR5(root, row); hit {
				if err := fileFinding(row, "R5", "judgment", detail, fix); err != nil {
					return err
				}
			}
		}

		// R6 runs only when HANDOFF_PATH is a member of the filtered set (spec §5.2).
		for _, row := range filtered {
			if row.Path == cfg.HandoffPath {
				if detail, fix, hit := checkR6(root, row); hit {
					if err := fileFinding(row, "R6", "judgment", detail, fix); err != nil {
						return err
					}
				}
				break
			}
		}

		// Resolution (deterministic; same transaction as the patrol write). Any open finding
		// whose (path, rule) did NOT re-fire this run transitions to state=resolved with the
		// resolving run id recorded. A FULL-desk patrol (no path restriction) considers every
		// open finding; a SCOPED patrol (--path) resolves only findings within its scope, so a
		// stale finding outside the scope is left flagged.
		findingsResolved := 0
		for _, of := range openFindings {
			if in.Path != "" && !pathOrSubtree(of.path, in.Path) {
				continue // out of scope for a scoped patrol
			}
			if fired[ruleKey{of.path, of.rule}] {
				continue // still firing this run — stays flagged
			}
			of.rec.Set("state", "resolved")
			of.rec.Set("resolved_run", runID)
			if err := txApp.Save(of.rec); err != nil {
				return err
			}
			findingsResolved++
		}

		logCollection, err := txApp.FindCollectionByNameOrId("patrol_log")
		if err != nil {
			return err
		}
		started := time.Now().UTC()
		logRec := core.NewRecord(logCollection)
		logRec.Set("run_id", runID)
		logRec.Set("desk", cfg.DeskName)
		logRec.Set("started", started)
		logRec.Set("finished", time.Now().UTC())
		logRec.Set("files_swept", result.FilesSwept)
		logRec.Set("findings_new", result.FindingsNew)
		logRec.Set("summary", fmt.Sprintf("%s resolved=%d", patrolSummary(result), findingsResolved))
		return txApp.Save(logRec)
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

// mechanicalRules is the fixed MECHANICAL check order (spec §5.2 point 3): sorted
// MECHANICAL (R1,R2,R3), then the JUDGMENT rules (R4, R5) run separately below, then R6 last.
// R4 detects mechanically but remediates by judgment, so it is filed as judgment (see the R4
// block in Patrol) and is not in this list.
var mechanicalRules = []string{"R1", "R2", "R3"}

func runMechanicalCheck(root, rule string, row fileRow, dirMap map[string]string) (string, string, bool) {
	switch rule {
	case "R1":
		return checkR1(root, row)
	case "R2":
		return checkR2(row)
	case "R3":
		return checkR3(row, dirMap)
	default:
		return "", "", false
	}
}

func patrolSummary(result *PatrolResult) string {
	var parts []string
	for _, rule := range []string{"R1", "R2", "R3", "R4", "R5", "R6"} {
		if n, ok := result.ByRule[rule]; ok {
			parts = append(parts, fmt.Sprintf("%s=%d", rule, n))
		}
	}
	detail := "none"
	if len(parts) > 0 {
		detail = strings.Join(parts, ", ")
	}
	return fmt.Sprintf("files=%d findings_new=%d (%s)", result.FilesSwept, result.FindingsNew, detail)
}

// --- dedupe key (spec §5.2 point 4: key = (path, rule, checksum)) ---

type findingKey struct{ path, rule, checksum string }

// ruleKey identifies a (path, rule) pair independent of checksum — the granularity at which a
// finding "re-fires". A finding is resolved when its ruleKey did not fire in the current run.
type ruleKey struct{ path, rule string }

// openFinding is an in-scope flagged finding carried through the run for the resolution pass:
// the record to patch plus its resolved (path, rule).
type openFinding struct {
	rec  *core.Record
	path string
	rule string
}

func findingDedupeKey(path, rule, checksum string) findingKey {
	return findingKey{path, rule, checksum}
}

func isDuplicateFinding(open map[findingKey]bool, path, rule, checksum string) bool {
	return open[findingDedupeKey(path, rule, checksum)]
}

// --- rule detection (spec §5.2 table, PoC-verbatim text) ---

var entityDirKinds = map[string]bool{"decisions": true, "tasks": true, "analyses": true, "journal": true}

func isEntityDoc(row fileRow) bool {
	// An index README.md inside an entity dir is a directory index, not an entity record —
	// exempt it so R1 (and R5) never fire on it.
	if filepath.Base(row.Path) == "README.md" {
		return false
	}
	return entityDirKinds[row.DirKind] && strings.HasSuffix(row.Path, ".md")
}

var universalFMKeys = []string{"type", "created", "updated", "tags", "synopsis"}

// r1MissingKeys is the pure core of R1: which UNIVERSAL_FM_KEYS are absent from fm.
func r1MissingKeys(fm map[string]any) []string {
	var missing []string
	for _, k := range universalFMKeys {
		if _, ok := fm[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}

func r1Detail(missing []string) (string, string) {
	return "missing universal frontmatter: " + strings.Join(missing, ", "),
		"add the missing keys per the headcase frontmatter contract"
}

func checkR1(root string, row fileRow) (string, string, bool) {
	if !isEntityDoc(row) {
		return "", "", false
	}
	fm := map[string]any{}
	if text, ok := readText(filepath.Join(root, row.Path)); ok {
		fm = desklib.ParseFrontmatter(text)
	}
	missing := r1MissingKeys(fm)
	if len(missing) == 0 {
		return "", "", false
	}
	detail, fix := r1Detail(missing)
	return detail, fix, true
}

var journalNameRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-.+\.md$`)

// r2Check is the pure core of R2.
func r2Check(dirKind, relPath string) (string, string, bool) {
	if dirKind != "journal" {
		return "", "", false
	}
	name := basename(relPath)
	if journalNameRe.MatchString(name) {
		return "", "", false
	}
	return "journal filename not yyyy-mm-dd-*.md: " + name,
		"rename to yyyy-mm-dd-subject.md", true
}

func checkR2(row fileRow) (string, string, bool) { return r2Check(row.DirKind, row.Path) }

// r3Check is the pure core of R3. dirMap is TYPE_DIR_MAP (entity_type -> configured PATH,
// spec §5.2 — NOT dir_kind labels).
func r3Check(entityType, dirKind, relPath string, dirMap map[string]string) (string, string, bool) {
	expected, ok := dirMap[entityType]
	if !ok || expected == "" {
		return "", "", false
	}
	if pathOrSubtree(relPath, expected) {
		return "", "", false
	}
	return fmt.Sprintf("type '%s' but file lives under '%s' (expected %s/)", entityType, dirKind, expected),
		fmt.Sprintf("move the doc to %s/ or fix its type", expected), true
}

func checkR3(row fileRow, dirMap map[string]string) (string, string, bool) {
	return r3Check(row.EntityType, row.DirKind, row.Path, dirMap)
}

var decisionStatuses = map[string]bool{"proposed": true, "accepted": true, "rejected": true, "superseded": true}

// r4Check is the pure core of R4 (judgment, flag-only, no fixer — the fix requires choosing a
// status value, which is a supervisor's semantic call; filed as judgment in Patrol).
func r4Check(dirKind, relPath, status string) (string, string, bool) {
	if dirKind != "decisions" || !strings.HasSuffix(relPath, ".md") {
		return "", "", false
	}
	// A decisions-dir README.md is a directory index, not a decision record — never flag it
	// for a missing decision status.
	if filepath.Base(relPath) == "README.md" {
		return "", "", false
	}
	if decisionStatuses[status] {
		return "", "", false
	}
	shown := status
	if shown == "" {
		shown = "(empty)"
	}
	return fmt.Sprintf("decision status %s not in {proposed, accepted, rejected, superseded}", shown),
		"set a valid status", true
}

func checkR4(row fileRow) (string, string, bool) { return r4Check(row.DirKind, row.Path, row.Status) }

// r5Check is the pure core of R5 (judgment, flag-only), given already-read text. Decisions
// are excluded (append-only records, never collapse targets).
func r5Check(dirKind string, text string) (string, string, bool) {
	if dirKind == "decisions" {
		return "", "", false
	}
	lines := lineCount(text)
	if lines <= 40 {
		return "", "", false
	}
	ref := issueRefFind(text)
	if ref == "" {
		return "", "", false
	}
	return fmt.Sprintf("references filed issue %s but is %d lines — a graduated doc should be a short pointer stub", ref, lines),
		fmt.Sprintf("collapse the doc to a pointer at %s", ref), true
}

func checkR5(root string, row fileRow) (string, string, bool) {
	if !isEntityDoc(row) {
		return "", "", false
	}
	text, ok := readText(filepath.Join(root, row.Path))
	if !ok {
		return "", "", false
	}
	return r5Check(row.DirKind, text)
}

var handoffDateRe = regexp.MustCompile(`Last updated:\s*(\d{4}-\d{2}-\d{2})`)

// r6Check is the pure core of R6 (judgment, handled separately — operates on the handoff
// record only), given the handoff's already-read text and the newest desk-commit date.
func r6Check(text, newest string) (string, string, bool) {
	if newest == "" {
		return "", "", false
	}
	fm := desklib.ParseFrontmatter(text)
	docDate := fmStr(fm, "updated")
	if docDate == "" {
		if m := handoffDateRe.FindStringSubmatch(text); m != nil {
			docDate = m[1]
		}
	}
	if docDate == "" || docDate < newest {
		shown := docDate
		if shown == "" {
			shown = "(unknown)"
		}
		return fmt.Sprintf("HANDOFF.md last updated %s but newest desk commit is %s — handoff may be stale", shown, newest),
			"refresh _meta/HANDOFF.md at the next session boundary", true
	}
	return "", "", false
}

func checkR6(root string, row fileRow) (string, string, bool) {
	text, ok := readText(filepath.Join(root, row.Path))
	if !ok {
		return "", "", false
	}
	// A git failure degrades to "no R6 finding" (desklib.GitNewestCommit returns "" on any
	// git error), matching the PoC's `newest` empty -> None (spec §5.2 Errors).
	newest := desklib.GitNewestCommit(root)
	return r6Check(text, newest)
}

// readText is a best-effort UTF-8 text read: ("", false) on any read error or invalid UTF-8
// (mirrors the PoC's read_text, which returns None on OSError/UnicodeDecodeError).
func readText(abs string) (string, bool) {
	raw, err := os.ReadFile(abs)
	if err != nil || !utf8.Valid(raw) {
		return "", false
	}
	return string(raw), true
}
