package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
)

// errStoreNotInitialized is the actionable error Query surfaces in place of pocketbase's bare
// sql.ErrNoRows when the store's collections have never been created. The app migrations that
// create the files/findings/adoption collections run only under `serve` and `migrate up` — a
// plain tool command against a store that has never had either applied finds no collection at
// all, so every query kind hits the same underlying condition. A migrated-but-unswept store,
// by contrast, degrades cleanly to empty results; the remedy is `migrate up`, not `sweep`.
var errStoreNotInitialized = errors.New("store is not initialized — run `librarian migrate up` first")

// Query — §5.6: read-only queries over files/findings/adoption. Kind selects one of
// live_files | recent | orphans | uncollapsed | findings | summary | adoption; the returned
// JSON document echoes kind + count (except `summary`, which mirrors the /api/desk/summary
// aggregate shape verbatim, per the concrete examples in §5.6) plus a kind-specific body.
// Never writes.
func Query(ctx context.Context, app core.App, cfg *config.Config, in *QueryInput) (json.RawMessage, error) {
	var (
		raw json.RawMessage
		err error
	)
	switch in.Kind {
	case "live_files":
		raw, err = queryLiveFiles(app)
	case "recent":
		days := in.Days
		if days <= 0 {
			days = 7
		}
		raw, err = queryRecent(app, days)
	case "orphans":
		raw, err = queryOrphans(app, cfg, in.ShowIndex)
	case "uncollapsed":
		raw, err = queryUncollapsed(app)
	case "findings":
		raw, err = queryFindings(app, in.IncludeDisposed)
	case "summary":
		raw, err = querySummary(app)
	case "adoption":
		raw, err = queryAdoption(app)
	case "feedback":
		raw, err = queryFeedback(app)
	case "search":
		raw, err = querySearch(app, in.Term, in.Limit)
	case "content":
		raw, err = queryContent(app, in.Path)
	default:
		return nil, fmt.Errorf("query: unknown kind %q (one of: live_files recent orphans uncollapsed findings summary adoption feedback search content)", in.Kind)
	}
	if err != nil {
		return nil, translateUninitializedStoreError(err)
	}
	return raw, nil
}

// translateUninitializedStoreError rewrites the collection-not-found error pocketbase's record/
// collection lookups return — the bare database/sql sentinel sql.ErrNoRows, indistinguishable
// at this call site from a genuinely empty (but existing) result set — into errStoreNotInitialized.
// Any other error (an invalid filter expression, a real driver failure) passes through as-is.
func translateUninitializedStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errStoreNotInitialized
	}
	return err
}

// --- output shapes (spec §5.6 "Return JSON shape per kind (concrete)") ---

type fileBrief struct {
	Path          string `json:"path"`
	DirKind       string `json:"dir_kind"`
	Doctype       string `json:"doctype"`
	Status        string `json:"status"`
	GraduatedTo   string `json:"graduated_to"`
	GitLastCommit string `json:"git_last_commit"`
}

type orphanBrief struct {
	Path    string `json:"path"`
	DirKind string `json:"dir_kind"`
}

// findingBrief is one open (or, with --include-disposed, any) patrol finding as the
// `findings`/`uncollapsed` queries return it: the record id (so a caller can act on it via
// `findings dispose <id>`, matching feedbackBrief below) plus path + detail.
type findingBrief struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

type adoptionRow struct {
	Date   string `json:"date"`
	Event  string `json:"event"`
	Detail string `json:"detail"`
}

// findingRow is the plain shape a patrol_findings record is reduced to for query purposes.
type findingRow struct {
	ID       string
	Path     string
	Rule     string
	Detail   string
	Severity string
}

type liveFilesResult struct {
	Kind  string      `json:"kind"`
	Count int         `json:"count"`
	Files []fileBrief `json:"files"`
}

type recentResult struct {
	Kind  string      `json:"kind"`
	Days  int         `json:"days"`
	Count int         `json:"count"`
	Files []fileBrief `json:"files"`
}

type orphansResult struct {
	Kind  string        `json:"kind"`
	Count int           `json:"count"`
	Files []orphanBrief `json:"files"`
}

type uncollapsedResult struct {
	Kind     string         `json:"kind"`
	Count    int            `json:"count"`
	Findings []findingBrief `json:"findings"`
}

type findingsResult struct {
	Kind   string                    `json:"kind"`
	Count  int                       `json:"count"`
	ByRule map[string][]findingBrief `json:"by_rule"`
}

type summaryResult struct {
	Kind                   string         `json:"kind"`
	FilesTotal             int            `json:"files_total"`
	FilesByDirKind         map[string]int `json:"files_by_dir_kind"`
	OpenFindingsTotal      int            `json:"open_findings_total"`
	OpenFindingsByRule     map[string]int `json:"open_findings_by_rule"`
	OpenFindingsBySeverity map[string]int `json:"open_findings_by_severity"`
}

type adoptionResult struct {
	Kind  string        `json:"kind"`
	Count int           `json:"count"`
	Rows  []adoptionRow `json:"rows"`
}

// feedbackBrief is one open feedback-log entry as the `feedback` query returns it: identity +
// summary + source + status + created, with the full detail body included (spec: read-back
// carries detail; source lets a harvest pass split agent-observed problems from user asks).
type feedbackBrief struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	Detail  string `json:"detail"`
	Source  string `json:"source"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

type feedbackResult struct {
	Kind    string          `json:"kind"`
	Count   int             `json:"count"`
	Entries []feedbackBrief `json:"entries"`
}

// searchMatch is one hit of the `search` query: the file's path + dir_kind and a short snippet of
// context around the first case-insensitive occurrence of the term in that file's indexed body.
type searchMatch struct {
	Path    string `json:"path"`
	DirKind string `json:"dir_kind"`
	Snippet string `json:"snippet"`
}

type searchResult struct {
	Kind    string        `json:"kind"`
	Term    string        `json:"term"`
	Count   int           `json:"count"`
	Matches []searchMatch `json:"matches"`
}

// contentResult is the `content` query: the full stored body for one desk-relative path, or
// found=false when no live row exists at that path.
type contentResult struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Found   bool   `json:"found"`
	Content string `json:"content"`
}

// --- pure helpers (unit-testable with hand-built fileRow/findingRow fixtures) ---

func toFileBrief(row fileRow) fileBrief {
	return fileBrief{
		Path:          row.Path,
		DirKind:       row.DirKind,
		Doctype:       row.Doctype,
		Status:        row.Status,
		GraduatedTo:   row.GraduatedTo,
		GitLastCommit: row.GitLastCommit,
	}
}

// splitCommitDate extracts the "yyyy-mm-dd" half of a "<hash>|<date>" git-meta string;
// "" if there is no "|" (untracked / no git history).
func splitCommitDate(v string) string {
	if i := strings.Index(v, "|"); i >= 0 {
		return v[i+1:]
	}
	return ""
}

// cutoffDate returns the "yyyy-mm-dd" cutoff for a `recent --days N` window, anchored at now.
func cutoffDate(days int, now time.Time) string {
	return now.AddDate(0, 0, -days).Format("2006-01-02")
}

// withinRecentWindow reports whether a non-empty date string is on/after cutoff.
func withinRecentWindow(date, cutoff string) bool {
	return date != "" && date >= cutoff
}

func sortFileBriefsByCommitDateDesc(files []fileBrief) {
	sort.Slice(files, func(i, j int) bool {
		di, dj := splitCommitDate(files[i].GitLastCommit), splitCommitDate(files[j].GitLastCommit)
		if di != dj {
			return di > dj
		}
		return files[i].Path < files[j].Path
	})
}

// isOrphan is the pure core of the `orphans` query (spec §5.6): a .md file with empty
// doctype that could be misfiled desk content — i.e. NOT non-entity infrastructure. Files
// whose dir_kind is meta, memory, or infra (dotted infra dirs like .claude/.agents) are
// legitimately outside the entity taxonomy and never count as orphans (spec §5.6); the
// isMetaPath check also guards meta/secrets belt-and-suspenders.
func isOrphan(row fileRow, secretsDir string) bool {
	if !strings.HasSuffix(row.Path, ".md") || row.Doctype != "" {
		return false
	}
	switch row.DirKind {
	case "meta", "memory", "infra":
		return false
	}
	return !isMetaPath(row.Path, secretsDir)
}

// isIndexEntryFile reports whether path's basename (case-insensitive) is one of the by-design
// unreferenced index/entry files: CLAUDE.md, README.md, INDEX.md. These are structurally
// unreferenced on purpose — an entry/index doc is what other docs point AT, not the other way
// round — so they are noise in the default `orphans` view even though the structural isOrphan
// predicate (empty doctype .md outside meta/memory/infra) would otherwise surface them. They are
// filtered from default orphans and opted back in with the query's ShowIndex flag (--show-index).
func isIndexEntryFile(path string) bool {
	switch strings.ToLower(basename(path)) {
	case "claude.md", "readme.md", "index.md":
		return true
	}
	return false
}

func sortOrphanBriefs(files []orphanBrief) {
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
}

func sortFindingBriefs(findings []findingBrief) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Detail < findings[j].Detail
	})
}

// groupFindingsByRule is the pure core of the `findings` query: group open findings by
// rule, each group sorted by (path, detail).
func groupFindingsByRule(findings []findingRow) map[string][]findingBrief {
	byRule := map[string][]findingBrief{}
	for _, f := range findings {
		byRule[f.Rule] = append(byRule[f.Rule], findingBrief{ID: f.ID, Path: f.Path, Detail: f.Detail})
	}
	for rule := range byRule {
		sortFindingBriefs(byRule[rule])
	}
	return byRule
}

// buildSummary is the pure core of the `summary` query aggregate.
func buildSummary(files []fileRow, findings []findingRow) summaryResult {
	byDirKind := map[string]int{}
	for _, row := range files {
		byDirKind[row.DirKind]++
	}
	byRule := map[string]int{}
	bySeverity := map[string]int{}
	for _, f := range findings {
		byRule[f.Rule]++
		bySeverity[f.Severity]++
	}
	return summaryResult{
		Kind:                   "summary",
		FilesTotal:             len(files),
		FilesByDirKind:         byDirKind,
		OpenFindingsTotal:      len(findings),
		OpenFindingsByRule:     byRule,
		OpenFindingsBySeverity: bySeverity,
	}
}

func adoptionDate(raw string) string {
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

// snippetWindow is the context radius (bytes, snapped outward to a rune boundary) shown on each
// side of a `search` match.
const snippetWindow = 120

// snippet returns a short excerpt of content centered on the FIRST case-insensitive occurrence of
// term, with up to `window` bytes of context on each side and a leading/trailing "…" whenever the
// excerpt is clipped from the full body. An empty term — or a term that does not occur — yields the
// leading window of the body (match position 0). Clip boundaries are snapped outward to rune
// boundaries so the excerpt is always valid UTF-8. Pure and allocation-light: it slices the body
// rather than rune-copying the whole thing. Case-insensitive match location assumes ASCII-stable
// lowering (true for desk markdown); the rune-boundary snap keeps the output valid regardless.
func snippet(content, term string, window int) string {
	if window < 0 {
		window = 0
	}
	start, matchLen := 0, 0
	if term != "" {
		if i := strings.Index(strings.ToLower(content), strings.ToLower(term)); i >= 0 {
			start = i
			// Byte length, deliberately: for a multi-byte term this overshoots the rune count,
			// biasing the window a little right of the match end. Harmless — the boundaries are
			// rune-snapped below — and cheaper than a rune count per snippet.
			matchLen = len(term)
		}
	}
	lo := start - window
	if lo < 0 {
		lo = 0
	}
	hi := start + matchLen + window
	if hi > len(content) {
		hi = len(content)
	}
	for lo > 0 && !utf8.RuneStart(content[lo]) {
		lo--
	}
	for hi < len(content) && !utf8.RuneStart(content[hi]) {
		hi++
	}
	var b strings.Builder
	if lo > 0 {
		b.WriteString("…")
	}
	b.WriteString(content[lo:hi])
	if hi < len(content) {
		b.WriteString("…")
	}
	return b.String()
}

// --- DB-touching query implementations ---

func liveFileRecords(app core.App) ([]fileRow, error) {
	recs, err := app.FindRecordsByFilter("files", "deleted = false", "path", 0, 0, dbx.Params{})
	if err != nil {
		return nil, err
	}
	return fileRowsFromRecords(recs), nil
}

func filePathIndex(app core.App) (map[string]string, error) {
	recs, err := app.FindRecordsByFilter("files", "", "", 0, 0, dbx.Params{})
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(recs))
	for _, r := range recs {
		m[r.Id] = r.GetString("path")
	}
	return m, nil
}

func openFindingRows(app core.App, extraFilter string) ([]findingRow, error) {
	filter := "state = 'flagged'"
	if extraFilter != "" {
		filter += " && " + extraFilter
	}
	recs, err := app.FindRecordsByFilter("patrol_findings", filter, "", 0, 0, dbx.Params{})
	if err != nil {
		return nil, err
	}
	paths, err := filePathIndex(app)
	if err != nil {
		return nil, err
	}
	rows := make([]findingRow, len(recs))
	for i, f := range recs {
		rows[i] = findingRow{
			ID:       f.Id,
			Path:     paths[f.GetString("file")],
			Rule:     f.GetString("rule"),
			Detail:   f.GetString("detail"),
			Severity: f.GetString("severity"),
		}
	}
	return rows, nil
}

func queryLiveFiles(app core.App) (json.RawMessage, error) {
	rows, err := liveFileRecords(app)
	if err != nil {
		return nil, err
	}
	files := make([]fileBrief, len(rows))
	for i, row := range rows {
		files[i] = toFileBrief(row)
	}
	return json.Marshal(liveFilesResult{Kind: "live_files", Count: len(files), Files: files})
}

func queryRecent(app core.App, days int) (json.RawMessage, error) {
	rows, err := liveFileRecords(app)
	if err != nil {
		return nil, err
	}
	cutoff := cutoffDate(days, time.Now().UTC())
	var files []fileBrief
	for _, row := range rows {
		if withinRecentWindow(splitCommitDate(row.GitLastCommit), cutoff) {
			files = append(files, toFileBrief(row))
		}
	}
	sortFileBriefsByCommitDateDesc(files)
	return json.Marshal(recentResult{Kind: "recent", Days: days, Count: len(files), Files: files})
}

func queryOrphans(app core.App, cfg *config.Config, showIndex bool) (json.RawMessage, error) {
	rows, err := liveFileRecords(app)
	if err != nil {
		return nil, err
	}
	var files []orphanBrief
	for _, row := range rows {
		if !isOrphan(row, cfg.SecretsDir) {
			continue
		}
		// Index/entry files (CLAUDE.md, READMEs, INDEX.md) are by-design unreferenced, so they are
		// hidden from default orphans as an ADDITIONAL filter layered on top of the structural
		// isOrphan predicate (which is left unchanged so its unit tests keep passing); --show-index
		// (ShowIndex=true) opts them back in.
		if !showIndex && isIndexEntryFile(row.Path) {
			continue
		}
		files = append(files, orphanBrief{Path: row.Path, DirKind: row.DirKind})
	}
	sortOrphanBriefs(files)
	return json.Marshal(orphansResult{Kind: "orphans", Count: len(files), Files: files})
}

func queryUncollapsed(app core.App) (json.RawMessage, error) {
	rows, err := openFindingRows(app, "rule = 'R5' && disposition = 'open'")
	if err != nil {
		return nil, err
	}
	findings := make([]findingBrief, len(rows))
	for i, f := range rows {
		findings[i] = findingBrief{ID: f.ID, Path: f.Path, Detail: f.Detail}
	}
	sortFindingBriefs(findings)
	return json.Marshal(uncollapsedResult{Kind: "uncollapsed", Count: len(findings), Findings: findings})
}

// queryFindings — `findings` kind. By default it is LIVE-ONLY: it returns flagged findings whose
// disposition is still 'open', hiding those a supervisor has acknowledged/triaged/marked wont_fix
// (the disposition lifecycle). includeDisposed=true drops the disposition filter and
// returns every flagged finding regardless of disposition. All three count surfaces built on
// openFindingRows — queryFindings, queryUncollapsed, and querySummary — are disposition-aware by
// default (disposition = 'open'), so a disposed finding does not inflate the summary total or
// still show up as uncollapsed; includeDisposed=true widens queryFindings only, the other two
// surfaces stay live-only. See the disposition-lifecycle migration (0014) for the field
// definition and backfill.
func queryFindings(app core.App, includeDisposed bool) (json.RawMessage, error) {
	extraFilter := "disposition = 'open'"
	if includeDisposed {
		extraFilter = ""
	}
	rows, err := openFindingRows(app, extraFilter)
	if err != nil {
		return nil, err
	}
	byRule := groupFindingsByRule(rows)
	return json.Marshal(findingsResult{Kind: "findings", Count: len(rows), ByRule: byRule})
}

func querySummary(app core.App) (json.RawMessage, error) {
	fileRecs, err := app.FindRecordsByFilter("files", "deleted = false", "", 0, 0, dbx.Params{})
	if err != nil {
		return nil, err
	}
	findingRows, err := openFindingRows(app, "disposition = 'open'")
	if err != nil {
		return nil, err
	}
	return json.Marshal(buildSummary(fileRowsFromRecords(fileRecs), findingRows))
}

// queryFeedback returns the OPEN feedback-log entries newest-first (by the created autodate).
// Each entry carries id/kind/summary/status/created plus the full detail body. Resolved entries
// are omitted — this is the actionable backlog, mirroring how `findings` shows only flagged rows.
func queryFeedback(app core.App) (json.RawMessage, error) {
	recs, err := app.FindRecordsByFilter("feedback", "status = 'open'", "-created", 0, 0, dbx.Params{})
	if err != nil {
		return nil, err
	}
	entries := make([]feedbackBrief, len(recs))
	for i, r := range recs {
		entries[i] = feedbackBrief{
			ID:      r.Id,
			Kind:    r.GetString("kind"),
			Summary: r.GetString("summary"),
			Detail:  r.GetString("detail"),
			Source:  r.GetString("source"),
			Status:  r.GetString("status"),
			Created: r.GetString("created"),
		}
	}
	return json.Marshal(feedbackResult{Kind: "feedback", Count: len(entries), Entries: entries})
}

func queryAdoption(app core.App) (json.RawMessage, error) {
	recs, err := app.FindRecordsByFilter("adoption_log", "", "", 0, 0, dbx.Params{})
	if err != nil {
		return nil, err
	}
	rows := make([]adoptionRow, len(recs))
	for i, r := range recs {
		rows[i] = adoptionRow{
			Date:   adoptionDate(r.GetString("date")),
			Event:  r.GetString("event"),
			Detail: r.GetString("detail"),
		}
	}
	return json.Marshal(adoptionResult{Kind: "adoption", Count: len(rows), Rows: rows})
}

// querySearch — the `search` kind: substring/keyword retrieval over indexed file content (§5.6).
// v1 deliberately uses PocketBase's LIKE-contains operator `~` — `content ~ {:term}` compiles to
// `content LIKE '%term%'` (ASCII-case-insensitive in SQLite) — and NOT SQLite FTS5: plain keyword
// contains is sufficient here, and embeddings / vector search are out of scope. Results are ordered
// by path and capped (default 20, hard max 200); each match carries a short context snippet around
// the first occurrence. The FindRecordsByFilter error is returned so Query routes it through
// translateUninitializedStoreError, exactly like the other kinds (an uninitialized store yields the
// actionable "run migrate up" message, not a spuriously empty result).
func querySearch(app core.App, term string, limit int) (json.RawMessage, error) {
	if term == "" {
		return nil, errors.New("query search: term is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	// The `~` operator compiles to SQLite LIKE, whose `%`/`_` metacharacters are NOT escaped by
	// the param binding — a term containing them widens the match (`_` = any one char). Desk
	// prose rarely searches on bare `%`/`_`, so this is accepted v1 behavior, stated here so a
	// surprised future caller finds the explanation at the query site.
	recs, err := app.FindRecordsByFilter(
		"files",
		"deleted = false && content ~ {:term}",
		"path", limit, 0,
		dbx.Params{"term": term},
	)
	if err != nil {
		return nil, err
	}
	matches := make([]searchMatch, len(recs))
	for i, r := range recs {
		matches[i] = searchMatch{
			Path:    r.GetString("path"),
			DirKind: r.GetString("dir_kind"),
			Snippet: snippet(r.GetString("content"), term, snippetWindow),
		}
	}
	return json.Marshal(searchResult{Kind: "search", Term: term, Count: len(matches), Matches: matches})
}

// queryContent — the `content` kind: return the full stored body of one live desk file by its
// desk-relative path (the retrieval companion to search). found=false when no live row exists at
// that path. The lookup error routes through translateUninitializedStoreError like every other kind.
func queryContent(app core.App, p string) (json.RawMessage, error) {
	if p == "" {
		// Mirror the search kind's empty-term error: a missing --path/path is a usage error, not
		// a "file not found" — a silent found=false here would mask the caller's mistake.
		return nil, errors.New("query content: path is required")
	}
	recs, err := app.FindRecordsByFilter(
		"files",
		"path = {:path} && deleted = false",
		"", 1, 0,
		dbx.Params{"path": p},
	)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return json.Marshal(contentResult{Kind: "content", Path: p, Found: false})
	}
	return json.Marshal(contentResult{Kind: "content", Path: p, Found: true, Content: recs[0].GetString("content")})
}
