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

	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/desklib"
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

		existingList, ferr := txApp.FindRecordsByFilter("files", "", "", 0, 0, dbx.Params{})
		if ferr != nil {
			return ferr
		}
		existingByPath := make(map[string]*core.Record, len(existingList))
		for _, r := range existingList {
			existingByPath[r.GetString("path")] = r
		}

		now := time.Now().UTC()
		seen := make(map[string]bool, len(relPaths))
		for _, rel := range relPaths {
			row, serr := scanFile(root, rel, dirMap, cfg.SecretsDir, cfg.DeskName)
			if serr != nil {
				// Filesystem read errors on an individual file are recorded and skipped
				// (do not abort the sweep, spec §5.1 Errors).
				continue
			}
			seen[rel] = true
			old := existingByPath[rel]
			switch {
			case old == nil:
				rec := core.NewRecord(filesCollection)
				applyFileRow(rec, row)
				rec.Set("last_seen", now)
				if err := txApp.Save(rec); err != nil {
					return err
				}
				result.Created++
			case fileRowDiffers(old, row):
				applyFileRow(old, row)
				old.Set("last_seen", now)
				if err := txApp.Save(old); err != nil {
					return err
				}
				result.Updated++
			default:
				result.Unchanged++
			}
		}

		for p, old := range existingByPath {
			if !seen[p] && !old.GetBool("deleted") {
				old.Set("deleted", true)
				if err := txApp.Save(old); err != nil {
					return err
				}
				result.SoftDeleted++
			}
		}
		result.Total = len(seen)
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

// --- fileRow: the shared plain-Go shape of one `files` collection row (spec §4.2). Used by
// sweep (built from a filesystem scan), patrol, and query (read back from the DB) so that
// all rule/derivation logic is pure and testable without a PocketBase app. ---

type fileRow struct {
	ID            string
	Path          string
	Desk          string
	EntityType    string
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
}

// fileRowFromRecord reads a `files` collection *core.Record into the plain fileRow shape.
func fileRowFromRecord(r *core.Record) fileRow {
	return fileRow{
		ID:            r.Id,
		Path:          r.GetString("path"),
		Desk:          r.GetString("desk"),
		EntityType:    r.GetString("entity_type"),
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
	rec.Set("desk", row.Desk)
	rec.Set("entity_type", row.EntityType)
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
}

// fileRowDiffers implements COMPARE_FIELDS (spec §5.1): desk, entity_type, dir_kind, status,
// synopsis, origin, graduated_to, checksum, git_last_commit, fm_created, fm_updated, deleted.
// "path" and "last_seen" are excluded.
func fileRowDiffers(rec *core.Record, row fileRow) bool {
	return rec.GetString("desk") != row.Desk ||
		rec.GetString("entity_type") != row.EntityType ||
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
		row.EntityType = fmStr(fm, "type")
		row.Status = fmStr(fm, "status")
		row.Synopsis = fmStr(fm, "synopsis")
		row.FMCreated = fmStr(fm, "created")
		row.FMUpdated = fmStr(fm, "updated")
		if lineCount(text) <= 40 {
			if ref := issueRefFind(text); ref != "" {
				row.GraduatedTo = ref
			} else if url := ghURLRe.FindString(text); url != "" {
				row.GraduatedTo = url
			}
		}
	}
	return row, nil
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
