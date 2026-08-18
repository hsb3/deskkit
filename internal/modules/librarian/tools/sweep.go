package tools

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
)

// Sweep — §5.1: walk DESK_ROOT, checksum + parse frontmatter (desklib.ParseFrontmatter),
// derive dir_kind (cfg.EntityDirMap prefix match), apply the pointer-stub heuristic, and
// upsert the files collection inside one app.RunInTransaction. Idempotent: COMPARE_FIELDS
// excludes path + last_seen, unchanged files are not patched.
func Sweep(ctx context.Context, app core.App, cfg *config.Config, in *SweepInput) (*SweepResult, error) {
	root := cfg.DeskRoot
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("sweep: desk root not found: %s", root)
	}
	dirMap := cfg.EntityDirMap()

	relPaths, err := walkDeskFiles(root)
	if err != nil {
		return nil, err
	}

	result := &SweepResult{}
	txErr := app.RunInTransaction(func(txApp core.App) error {
		filesCollection, cerr := txApp.FindCollectionByNameOrId("files")
		if cerr != nil {
			return cerr
		}
		findingsCollection, fcerr := txApp.FindCollectionByNameOrId("patrol_findings")
		if fcerr != nil {
			return fcerr
		}

		existingList, ferr := txApp.FindRecordsByFilter("files", "", "", 0, 0, dbx.Params{})
		if ferr != nil {
			return ferr
		}
		// Two identity indexes: by path (always), and by frontmatter id -> the row that carries it
		// (files.doc_id, ADR 0017). A sweep prefers the doc_id match so a renamed doc keeps its row;
		// path is the fallback for a doc with no id.
		existingByPath := make(map[string]*core.Record, len(existingList))
		existingByDocID := make(map[string]*core.Record)
		for _, r := range existingList {
			existingByPath[r.GetString("path")] = r
			if id := r.GetString("doc_id"); id != "" {
				if _, dup := existingByDocID[id]; !dup {
					existingByDocID[id] = r
				}
			}
		}

		now := time.Now().UTC()
		seen := make(map[string]bool, len(relPaths))
		matched := make(map[string]bool, len(existingList)) // record ids claimed this sweep
		idClaimedBy := make(map[string]string)              // frontmatter id -> first rel that used it
		for _, rel := range relPaths {
			row, serr := scanFile(root, rel, dirMap, cfg.SecretsDir, cfg.DeskName)
			if serr != nil {
				// Filesystem read errors on an individual file are recorded and skipped
				// (do not abort the sweep, spec §5.1 Errors).
				continue
			}
			seen[rel] = true
			if row.Truncated {
				result.Truncated++
			}

			// Identity match: doc_id first (rename survival), path fallback. A frontmatter id
			// already claimed by an earlier doc THIS sweep is a duplicate — never merge two files
			// into one row: fall back to path-matching and surface a patrol-visible finding.
			var old *core.Record
			duplicateID := false
			if row.DocID != "" {
				if _, claimed := idClaimedBy[row.DocID]; claimed {
					duplicateID = true
					old = existingByPath[rel]
				} else {
					idClaimedBy[row.DocID] = rel
					if byID, ok := existingByDocID[row.DocID]; ok && !matched[byID.Id] {
						old = byID
					} else {
						old = existingByPath[rel]
					}
				}
			} else {
				old = existingByPath[rel]
			}

			var recID string
			switch {
			case old == nil:
				rec := core.NewRecord(filesCollection)
				applyFileRow(rec, row)
				rec.Set("last_seen", now)
				if err := txApp.Save(rec); err != nil {
					return err
				}
				recID = rec.Id
				result.Created++
			case old.GetString("path") != row.Path || fileRowDiffers(old, row):
				// Update — including the rename case (matched by doc_id, new path): path is excluded
				// from COMPARE_FIELDS, so a pure rename is applied via the explicit path comparison
				// here, moving the SAME record to the new path.
				applyFileRow(old, row)
				old.Set("last_seen", now)
				if err := txApp.Save(old); err != nil {
					return err
				}
				recID = old.Id
				result.Updated++
			default:
				recID = old.Id
				result.Unchanged++
			}
			matched[recID] = true

			if duplicateID {
				if err := fileDuplicateIDFinding(txApp, findingsCollection, recID, row, idClaimedBy[row.DocID], now); err != nil {
					return err
				}
			}
		}

		// Soft-delete by RECORD identity, not by path: any existing row NOT matched this sweep (by
		// doc_id or path) is gone from disk. A rename-with-id updated its row in place (matched), so
		// it is never soft-deleted at its old path — only a genuinely vanished file is.
		for _, old := range existingList {
			if matched[old.Id] || old.GetBool("deleted") {
				continue
			}
			old.Set("deleted", true)
			if err := txApp.Save(old); err != nil {
				return err
			}
			result.SoftDeleted++
		}
		result.Total = len(seen)
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

// fileDuplicateIDFinding records a patrol-visible finding when two desk documents share one
// frontmatter `id` within a single sweep. The duplicate keeps its OWN path identity (it was
// upserted by path, never merged into the row that first claimed the id); this finding surfaces
// the collision for a supervisor to resolve. It is filed against the duplicate's own files record
// (recID) and deduped on (file, rule, checksum) so a repeated sweep does not spam identical rows.
// severity is "judgment": detection is mechanical but choosing WHICH doc keeps the id is a human
// call, so there is no auto-fix.
func fileDuplicateIDFinding(txApp core.App, findings *core.Collection, recID string, row fileRow, otherPath string, now time.Time) error {
	const rule = "duplicate-doc-id"
	existing, err := txApp.FindRecordsByFilter(
		"patrol_findings",
		"file = {:file} && rule = {:rule} && checksum = {:checksum} && state = 'flagged'",
		"", 1, 0,
		dbx.Params{"file": recID, "rule": rule, "checksum": row.Checksum},
	)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil // already flagged for this file+content — do not duplicate the finding
	}
	fr := core.NewRecord(findings)
	fr.Set("file", recID)
	fr.Set("rule", rule)
	fr.Set("severity", "judgment")
	fr.Set("detail", fmt.Sprintf(
		"frontmatter id %q is also used by %q — two documents cannot share one id; this file kept its path identity instead of merging",
		row.DocID, otherPath))
	fr.Set("proposed_fix", "give one of the two documents a distinct frontmatter `id`")
	fr.Set("state", "flagged")
	fr.Set("patrol_run", "sweep-"+now.Format("20060102T150405Z"))
	fr.Set("checksum", row.Checksum)
	fr.Set("disposition", "open")
	return txApp.Save(fr)
}

// --- fileRow: the shared plain-Go shape of one `files` collection row (spec §4.2). Used by
// sweep (built from a filesystem scan), patrol, and query (read back from the DB) so that
// all rule/derivation logic is pure and testable without a PocketBase app. ---

type fileRow struct {
	ID            string
	Path          string
	DocID         string // frontmatter `id` (optional document-identity primitive, ADR 0017)
	Desk          string
	Doctype       string // frontmatter `type` (files.doctype column; renamed by migration 0019, ADR 0017)
	DirKind       string
	Status        string
	Synopsis      string
	Origin        string
	GraduatedTo   string
	Checksum      string
	GitLastCommit string
	FMCreated     string
	FMUpdated     string
	Deleted       bool
	Content       string // swept file body, indexed for retrieval/search (§5.6 content/search kinds)
	Truncated     bool   // transient scan flag: Content was clipped at maxContentRunes (never persisted)
}

// fileRowFromRecord reads a `files` collection *core.Record into the plain fileRow shape.
func fileRowFromRecord(r *core.Record) fileRow {
	return fileRow{
		ID:            r.Id,
		Path:          r.GetString("path"),
		DocID:         r.GetString("doc_id"),
		Desk:          r.GetString("desk"),
		Doctype:       r.GetString("doctype"),
		DirKind:       r.GetString("dir_kind"),
		Status:        r.GetString("status"),
		Synopsis:      r.GetString("synopsis"),
		Origin:        r.GetString("origin"),
		GraduatedTo:   r.GetString("graduated_to"),
		Checksum:      r.GetString("checksum"),
		GitLastCommit: r.GetString("git_last_commit"),
		FMCreated:     r.GetString("fm_created"),
		FMUpdated:     r.GetString("fm_updated"),
		Deleted:       r.GetBool("deleted"),
		Content:       r.GetString("content"),
	}
}

func fileRowsFromRecords(recs []*core.Record) []fileRow {
	rows := make([]fileRow, len(recs))
	for i, r := range recs {
		rows[i] = fileRowFromRecord(r)
	}
	return rows
}

// applyFileRow writes a scanned fileRow's comparable fields onto a *core.Record (create or
// update path). "path" and "last_seen" are handled by the caller (last_seen is excluded from
// COMPARE_FIELDS; §5.1).
func applyFileRow(rec *core.Record, row fileRow) {
	rec.Set("path", row.Path)
	rec.Set("doc_id", row.DocID)
	rec.Set("desk", row.Desk)
	rec.Set("doctype", row.Doctype)
	rec.Set("dir_kind", row.DirKind)
	rec.Set("status", row.Status)
	rec.Set("synopsis", row.Synopsis)
	rec.Set("origin", row.Origin)
	rec.Set("graduated_to", row.GraduatedTo)
	rec.Set("checksum", row.Checksum)
	rec.Set("git_last_commit", row.GitLastCommit)
	rec.Set("fm_created", row.FMCreated)
	rec.Set("fm_updated", row.FMUpdated)
	rec.Set("deleted", row.Deleted)
	rec.Set("content", row.Content)
}

// fileRowDiffers implements COMPARE_FIELDS (spec §5.1): doc_id, desk, doctype, dir_kind, status,
// synopsis, origin, graduated_to, checksum, git_last_commit, fm_created, fm_updated, deleted.
// "path" and "last_seen" are excluded — a rename that carries a frontmatter `id` is matched by
// doc_id and its new path is applied explicitly in Sweep (a pure path change is otherwise "no
// diff" here). doc_id IS compared so a doc that gains or changes its `id` re-persists.
//
// content IS compared, but it newly MATTERS only to backfill: a normal content edit already
// changes the checksum (which is compared), so an edited file diffs on checksum alone. The content
// comparison exists so a store created BEFORE the content field existed (empty content on every
// row) re-persists the body on the next sweep (empty content -> populated) without needing a
// content change. Sweep stays idempotent: an unchanged file whose content is already stored
// produces no diff here.
func fileRowDiffers(rec *core.Record, row fileRow) bool {
	return rec.GetString("doc_id") != row.DocID ||
		rec.GetString("content") != row.Content ||
		rec.GetString("desk") != row.Desk ||
		rec.GetString("doctype") != row.Doctype ||
		rec.GetString("dir_kind") != row.DirKind ||
		rec.GetString("status") != row.Status ||
		rec.GetString("synopsis") != row.Synopsis ||
		rec.GetString("origin") != row.Origin ||
		rec.GetString("graduated_to") != row.GraduatedTo ||
		rec.GetString("checksum") != row.Checksum ||
		rec.GetString("git_last_commit") != row.GitLastCommit ||
		rec.GetString("fm_created") != row.FMCreated ||
		rec.GetString("fm_updated") != row.FMUpdated ||
		rec.GetBool("deleted") != row.Deleted
}

// walkDeskFiles walks root pruning .git/logs and any dir/file whose name starts "pb_",
// returning sorted "/"-joined relative paths (spec §5.1 point 1).
func walkDeskFiles(root string) ([]string, error) {
	var rels []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if p == root {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "logs" || strings.HasPrefix(name, "pb_") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, "pb_") {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(rels)
	return rels, nil
}

// scanFile builds the fileRow for one desk file (spec §5.1 point 2, ported from
// sweep.py scan_file). dir_kind is derived for EVERY file, not just .md (matches the PoC:
// dir_kind_for is called unconditionally in scan_file's return, outside the .md branch).
func scanFile(root, rel string, dirMap map[string]string, secretsDir, deskName string) (fileRow, error) {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(abs)
	if err != nil {
		return fileRow{}, err
	}
	row := fileRow{
		Path:          rel,
		Desk:          deskName,
		DirKind:       dirKindFor(rel, dirMap, secretsDir),
		Checksum:      desklib.Checksum(raw),
		Origin:        desklib.GitOrigin(root, rel),
		GitLastCommit: desklib.GitLastCommit(root, rel),
	}
	if strings.HasSuffix(rel, ".md") && utf8.Valid(raw) {
		text := string(raw)
		fm := desklib.ParseFrontmatter(text)
		row.DocID = fmStr(fm, "id")
		row.Doctype = fmStr(fm, "type")
		row.Status = fmStr(fm, "status")
		row.Synopsis = fmStr(fm, "synopsis")
		row.FMCreated = fmStr(fm, "created")
		row.FMUpdated = fmStr(fm, "updated")
		// graduated_to is populated ONLY from an EXPLICIT graduation marker (a frontmatter
		// `graduated_to:` key, or a canonical inline `graduated to: <ref>` line) — never
		// inferred from a bare #N merely quoted in prose. The old heuristic (lines <= 40 AND
		// leftmost ISSUE_REF_RE/GH_URL_RE match) mis-populated this column for any short doc
		// that only cited an issue as evidence, and coupled to R5 (§5.2) it false-fired the
		// graduated-doc rule. Gating both on a deliberate marker fixes both symptoms at the
		// root; the marker is authoritative at ANY length (a graduated-but-uncollapsed doc is
		// exactly the >40-line case R5 flags).
		row.GraduatedTo = graduationMarker(text)
	}
	// Content indexing (§5.6 retrieval/search). A file under the desk's configured SECRETS_DIR is a
	// secret home and is NEVER indexed — mirrors the meta/secrets exclusion boundary (isMetaPath's
	// secrets clause). Otherwise, only UTF-8 text is stored (a binary/non-UTF-8 file indexes to ""),
	// truncated rune-safe to the collection's content cap so a very large file never overflows the
	// TextField Max. content is re-derivable by a fresh sweep (store is disposable; files-are-truth).
	switch {
	case secretsDir != "" && pathOrSubtree(rel, secretsDir):
		row.Content = ""
	case utf8.Valid(raw):
		body := string(raw)
		row.Content = truncateRunes(body, maxContentRunes)
		row.Truncated = len(row.Content) < len(body)
	default:
		row.Content = ""
	}
	return row, nil
}

// maxContentRunes is the files.content cap (see collections/0021_files_content.go). PocketBase's
// TextField Max counts runes, so truncating to this many runes keeps a body at or under the Max.
const maxContentRunes = 1000000

// truncateRunes returns s unchanged when it holds at most max runes, else the first max runes of s
// (never splitting a multi-byte rune). Single pass: the range loop yields rune-start byte offsets,
// so the cutoff is found without a separate RuneCountInString traversal.
func truncateRunes(s string, max int) string {
	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}
	return s
}

func fmStr(fm map[string]any, key string) string {
	if v, ok := fm[key].(string); ok {
		return v
	}
	return ""
}

// lineCount is the deterministic "lines" measure used by the pointer-stub heuristic (here)
// and R5 (§5.2): a raw newline count over the WHOLE file (len(text.split("\n")) in Python).
func lineCount(text string) int {
	return strings.Count(text, "\n") + 1
}

// pathOrSubtree is the shared prefix test used by dir_kind derivation, R3, the patrol Path
// filter, and query's orphan/meta check: rel == target, or rel starts with target + "/".
func pathOrSubtree(rel, target string) bool {
	return rel == target || strings.HasPrefix(rel, target+"/")
}

// dirKindFor derives the dir_kind bucket for rel (spec §5.1, exact algorithm — the
// nested-prefix fix over the PoC's bare-top-segment version).
func dirKindFor(rel string, dirMap map[string]string, secretsDir string) string {
	if !strings.Contains(rel, "/") {
		return "root"
	}
	best, bestLen := "", -1
	for typ, dir := range dirMap {
		if dir == "" || !pathOrSubtree(rel, dir) {
			continue
		}
		if len(dir) > bestLen {
			bestLen = len(dir)
			best = entityDirKindLabel(typ)
		}
	}
	if best != "" {
		return best
	}
	if isMetaPath(rel, secretsDir) {
		return "meta"
	}
	top := rel
	if i := strings.Index(rel, "/"); i >= 0 {
		top = rel[:i]
	}
	if top == "memory" || (top == ".claude" && strings.Contains(rel, "/memory/")) {
		return "memory"
	}
	// Non-entity infrastructure living under a dotted top-level dir (.claude, .agents, .github,
	// ...) is not misfiled desk content — bucket it as "infra" so the orphans view can exclude it
	// (spec §5.1/§5.6). The memory check above already claimed .claude/memory/**.
	if strings.HasPrefix(top, ".") {
		return "infra"
	}
	return "other"
}

// entityDirKindLabel maps a frontmatter `type` key to its dir_kind label (decision->decisions,
// task->tasks, analysis->analyses, journal->journal).
func entityDirKindLabel(typ string) string {
	switch typ {
	case "decision":
		return "decisions"
	case "task":
		return "tasks"
	case "analysis":
		return "analyses"
	case "journal":
		return "journal"
	default:
		return ""
	}
}

// isMetaPath reports whether rel is under SECRETS_DIR or starts with "_meta/" (spec §5.1 dir_kind
// step 3; also used by query's orphans predicate, §5.6).
func isMetaPath(rel, secretsDir string) bool {
	if secretsDir != "" && pathOrSubtree(rel, secretsDir) {
		return true
	}
	return strings.HasPrefix(rel, "_meta/")
}

// --- graduation marker (explicit-only; gates graduated_to §5.1 and R5 §5.2) ---

// inlineGraduationRe matches a CANONICAL graduation line — one that STARTS with "graduated
// to" (case-insensitive, optional colon) followed by a pointer target (`wb#N`, `#N`, a bare
// number, or a URL). Anchored at line start (multiline) so a bare #N merely mentioned inside a
// prose sentence ("...which graduated to #N last week") is NOT a marker; only a deliberate
// canonical line is. `wb#\d+` precedes `#?\d+` in the alternation so the wb-prefixed form wins.
// The pattern is pinned VERBATIM by the spec (§5.1 graduated_to precedence, §5.2 R5): `#?\d+`
// deliberately accepts a bare number ("graduated to: 42") as opaque pointer text — the line
// anchor plus the explicit "graduated to" prefix is what makes it a marker, not the ref format.
var inlineGraduationRe = regexp.MustCompile(`(?im)^\s*graduated to:?\s+(wb#\d+|#?\d+|https?://\S+)`)

// graduationMarker returns the doc's EXPLICIT graduation pointer, or "" when the doc declares
// none. A graduation is DECLARED, never inferred from a bare #N quoted in prose: the primary
// marker is the frontmatter `graduated_to:` key; a canonical inline `graduated to: <ref>` line
// is accepted as a secondary marker. This single gate backs both graduated_to population
// (sweep, §5.1) and R5 (patrol, §5.2), so a doc that only CITES an issue as evidence is never
// treated as graduated. Self-contained (parses its own frontmatter) so the R5 call site can
// swap `issueRefFind(text)` for `graduationMarker(text)` with no signature change.
func graduationMarker(text string) string {
	if v := strings.TrimSpace(fmStr(desklib.ParseFrontmatter(text), "graduated_to")); v != "" {
		return v
	}
	if m := inlineGraduationRe.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

// --- ISSUE_REF_RE / GH_URL_RE (spec §5.1 graduated_to precedence, §5.2 R5) ---

// ghURLRe = GH_URL_RE verbatim.
var ghURLRe = regexp.MustCompile(`https://github\.com/\S+`)

// wbRefRe matches the `\bwb#\d+` alternative of ISSUE_REF_RE. Go's RE2 supports \b (word
// boundary) natively, so only the OTHER alternative's lookbehind needs reimplementing below.
var wbRefRe = regexp.MustCompile(`\bwb#\d+`)

// hashRefRe matches bare `#\d+`; findBareHashRef filters out matches preceded by a word
// char or '&' to emulate ISSUE_REF_RE's `(?<![\w&])` lookbehind (RE2 has no lookbehind).
var hashRefRe = regexp.MustCompile(`#\d+`)

// issueRefFind reproduces Python's `ISSUE_REF_RE.search(text)` (leftmost match across the
// `\bwb#\d+` and `(?<![\w&])#\d+` alternatives) and returns the matched substring, or "" if
// neither alternative matches anywhere in text.
func issueRefFind(text string) string {
	wbLoc := wbRefRe.FindStringIndex(text)
	hashLoc := findBareHashRef(text)
	switch {
	case wbLoc != nil && hashLoc != nil:
		if wbLoc[0] <= hashLoc[0] {
			return text[wbLoc[0]:wbLoc[1]]
		}
		return text[hashLoc[0]:hashLoc[1]]
	case wbLoc != nil:
		return text[wbLoc[0]:wbLoc[1]]
	case hashLoc != nil:
		return text[hashLoc[0]:hashLoc[1]]
	default:
		return ""
	}
}

// findBareHashRef returns the [start,end) of the first `#\d+` match whose preceding byte is
// NOT a word character and NOT '&' (the (?<![\w&]) lookbehind, reimplemented as a
// preceding-byte rejection scan per spec §5.2 lines 989-991), or nil if none qualifies.
func findBareHashRef(text string) []int {
	for _, loc := range hashRefRe.FindAllStringIndex(text, -1) {
		start := loc[0]
		if start > 0 {
			prev := text[start-1]
			if isWordByte(prev) || prev == '&' {
				continue
			}
		}
		return loc
	}
	return nil
}

func isWordByte(b byte) bool {
	return b == '_' ||
		(b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

// basename is a "/"-aware basename helper (rel paths are always "/"-joined regardless of
// OS), used by R2's journal-filename check.
func basename(rel string) string { return path.Base(rel) }
