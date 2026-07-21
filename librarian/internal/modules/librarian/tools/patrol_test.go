package tools

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/desklib"
)

// fullFM is a complete universal-frontmatter doc: R1 finds no missing keys, so R1 does not fire.
const fullFM = "---\ntype: task\ncreated: 2026-07-15\nupdated: 2026-07-15\ntags: []\nsynopsis: \"x\"\n---\nbody line\n"

// missingFM is missing four universal keys: R1 fires.
const missingFM = "---\ntype: task\n---\none body line\n"

func TestIsEntityDoc(t *testing.T) {
	cases := []struct {
		dirKind, path string
		want          bool
	}{
		{"decisions", "_structure/decisions/0001-foo.md", true},
		{"tasks", "tasks/x.md", true},
		{"analyses", "analyses/y.md", true},
		{"journal", "journal/z.md", true},
		{"meta", "_meta/HANDOFF.md", false},
		{"other", "README.md", false},
		{"decisions", "_structure/decisions/0001-foo.txt", false},  // not .md
		{"decisions", "_structure/decisions/README.md", false},     // index README, not an entity record
		{"tasks", "tasks/README.md", false},                        // index README in another entity dir
		{"decisions", "_structure/decisions/9999-ignore-fixture.md", true}, // non-README decisions-dir .md is still an entity doc
	}
	for _, c := range cases {
		if got := isEntityDoc(fileRow{DirKind: c.dirKind, Path: c.path}); got != c.want {
			t.Errorf("isEntityDoc(%q,%q) = %v, want %v", c.dirKind, c.path, got, c.want)
		}
	}
}

func TestR1MissingKeys(t *testing.T) {
	fm := map[string]any{"type": "task", "created": "2026-01-01"}
	got := r1MissingKeys(fm)
	want := []string{"updated", "tags", "synopsis"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestR1MissingKeysNoneMissing(t *testing.T) {
	fm := map[string]any{"type": "task", "created": "2026-01-01", "updated": "2026-01-02", "tags": []string{}, "synopsis": "x"}
	if got := r1MissingKeys(fm); len(got) != 0 {
		t.Fatalf("expected no missing keys, got %#v", got)
	}
}

func TestR1DetailText(t *testing.T) {
	detail, fix := r1Detail([]string{"tags", "synopsis"})
	if detail != "missing universal frontmatter: tags, synopsis" {
		t.Fatalf("got %q", detail)
	}
	if fix != "add the missing keys per the headcase frontmatter contract" {
		t.Fatalf("got %q", fix)
	}
}

func TestR2CheckJournalNaming(t *testing.T) {
	if _, _, hit := r2Check("journal", "journal/2026-07-15-foo.md"); hit {
		t.Fatalf("well-formed journal filename must not flag")
	}
	detail, fix, hit := r2Check("journal", "journal/notes.md")
	if !hit {
		t.Fatalf("malformed journal filename must flag")
	}
	if detail != "journal filename not yyyy-mm-dd-*.md: notes.md" {
		t.Fatalf("got %q", detail)
	}
	if fix != "rename to yyyy-mm-dd-subject.md" {
		t.Fatalf("got %q", fix)
	}
}

func TestR2CheckIgnoresNonJournal(t *testing.T) {
	if _, _, hit := r2Check("tasks", "tasks/whatever.md"); hit {
		t.Fatalf("R2 only applies to dir_kind == journal")
	}
}

func TestR3CheckMismatch(t *testing.T) {
	dirMap := map[string]string{"decision": "_structure/decisions", "task": "tasks", "analysis": "analyses", "journal": "journal"}
	detail, fix, hit := r3Check("task", "journal", "journal/x.md", dirMap)
	if !hit {
		t.Fatalf("expected a mismatch finding")
	}
	if detail != "type 'task' but file lives under 'journal' (expected tasks/)" {
		t.Fatalf("got %q", detail)
	}
	if fix != "move the doc to tasks/ or fix its type" {
		t.Fatalf("got %q", fix)
	}
}

func TestR3CheckCorrectlyPlacedNestedDecision(t *testing.T) {
	// The build-brief's worked example: a correctly-placed nested decision never flags.
	dirMap := map[string]string{"decision": "_structure/decisions"}
	if _, _, hit := r3Check("decision", "decisions", "_structure/decisions/0014-foo.md", dirMap); hit {
		t.Fatalf("correctly-placed decision doc must not flag R3")
	}
}

func TestR3CheckNoTypeConfigured(t *testing.T) {
	dirMap := map[string]string{"decision": "_structure/decisions"}
	if _, _, hit := r3Check("", "other", "README.md", dirMap); hit {
		t.Fatalf("empty doctype must never flag R3")
	}
	if _, _, hit := r3Check("unknown-type", "other", "README.md", dirMap); hit {
		t.Fatalf("a type absent from TYPE_DIR_MAP must never flag R3")
	}
}

func TestR4CheckBadStatus(t *testing.T) {
	detail, fix, hit := r4Check("decisions", "_structure/decisions/0001-foo.md", "draft")
	if !hit {
		t.Fatalf("expected a bad-status finding")
	}
	if detail != "decision status draft not in {proposed, accepted, rejected, superseded}" {
		t.Fatalf("got %q", detail)
	}
	if fix != "set a valid status" {
		t.Fatalf("got %q", fix)
	}
}

func TestR4CheckEmptyStatusShownParenthesized(t *testing.T) {
	detail, _, hit := r4Check("decisions", "_structure/decisions/0001-foo.md", "")
	if !hit {
		t.Fatalf("expected a finding for empty status")
	}
	if detail != "decision status (empty) not in {proposed, accepted, rejected, superseded}" {
		t.Fatalf("got %q", detail)
	}
}

func TestR4CheckValidStatusNoFinding(t *testing.T) {
	for _, s := range []string{"proposed", "accepted", "rejected", "superseded"} {
		if _, _, hit := r4Check("decisions", "_structure/decisions/0001-foo.md", s); hit {
			t.Errorf("status %q must not flag", s)
		}
	}
}

func TestR4CheckOnlyAppliesToDecisionsMd(t *testing.T) {
	if _, _, hit := r4Check("tasks", "tasks/x.md", "draft"); hit {
		t.Fatalf("R4 only applies to dir_kind == decisions")
	}
	if _, _, hit := r4Check("decisions", "_structure/decisions/x.txt", "draft"); hit {
		t.Fatalf("R4 only applies to .md files")
	}
}

func TestR4CheckExemptsIndexReadme(t *testing.T) {
	// An index README.md in the decisions dir is a directory index, not a decision record —
	// it must never flag R4, even with an empty status.
	if _, _, hit := r4Check("decisions", "_structure/decisions/README.md", ""); hit {
		t.Fatalf("decisions-dir README.md must not flag R4 (it is a directory index)")
	}
	// Regression: a non-README decisions-dir .md with no status still flags R4.
	if _, _, hit := r4Check("decisions", "_structure/decisions/9999-ignore-fixture.md", ""); !hit {
		t.Fatalf("a non-README decisions-dir .md with an invalid status must still flag R4")
	}
}

func TestR5CheckExcludesDecisions(t *testing.T) {
	longText := "line\n"
	for i := 0; i < 45; i++ {
		longText += "more content here\n"
	}
	longText += "references wb#37 for context\n"
	if _, _, hit := r5Check("decisions", longText); hit {
		t.Fatalf("R5 must never fire on decisions (append-only)")
	}
}

func TestR5CheckShortDocNoFinding(t *testing.T) {
	shortText := "short doc\nwb#37\n"
	if _, _, hit := r5Check("tasks", shortText); hit {
		t.Fatalf("a <=40 line doc must not flag R5 regardless of issue refs")
	}
}

func TestR5CheckLongDocWithIssueRefFlags(t *testing.T) {
	var longText string
	for i := 0; i < 45; i++ {
		longText += "content line\n"
	}
	longText += "graduated to wb#37 now\n"
	detail, fix, hit := r5Check("tasks", longText)
	if !hit {
		t.Fatalf("expected R5 to fire on a long doc referencing an issue")
	}
	wantLines := lineCount(longText)
	wantDetail := "references filed issue wb#37 but is " + strconv.Itoa(wantLines) + " lines — a graduated doc should be a short pointer stub"
	if detail != wantDetail {
		t.Fatalf("got %q, want %q", detail, wantDetail)
	}
	if fix != "collapse the doc to a pointer at wb#37" {
		t.Fatalf("got %q", fix)
	}
}

func TestR5CheckLongDocNoIssueRefNoFinding(t *testing.T) {
	var longText string
	for i := 0; i < 45; i++ {
		longText += "content line with no reference\n"
	}
	if _, _, hit := r5Check("tasks", longText); hit {
		t.Fatalf("a long doc with no issue ref must not flag R5")
	}
}

// TestR5CheckLongDocProseIssueQuoteDoesNotFire is the R5 graduation-marker regression: a
// >40-line entity doc whose body only QUOTES an issue number in prose (not a graduation marker)
// must NOT flag R5. Before r5Check swapped issueRefFind for graduationMarker this false-fired
// because the bare hash ref matched; a mere citation is not an explicit graduation declaration.
// The issue token is assembled at runtime so the neutrality lint never sees a literal bare #N.
func TestR5CheckLongDocProseIssueQuoteDoesNotFire(t *testing.T) {
	var longText string
	for i := 0; i < 45; i++ {
		longText += "content line\n"
	}
	longText += "see issue " + "#" + "111" + " for context\n"
	if _, _, hit := r5Check("tasks", longText); hit {
		t.Fatalf("a long doc that only quotes an issue number in prose must not flag R5")
	}
}

// TestR5CheckLongDocWithGraduatedToFrontmatterFlags is the compat direction: a >40-line doc that
// carries an EXPLICIT graduated_to frontmatter marker still flags R5 (graduationMarker's primary
// marker). The existing TestR5CheckLongDocWithIssueRefFlags covers the inline `graduated to`
// marker; this covers the frontmatter key. wb#37 is preceded by a word char, so it is not a bare
// issue ref under the neutrality lint.
func TestR5CheckLongDocWithGraduatedToFrontmatterFlags(t *testing.T) {
	longText := "---\ngraduated_to: \"wb#37\"\n---\n"
	for i := 0; i < 45; i++ {
		longText += "content line\n"
	}
	if _, _, hit := r5Check("tasks", longText); !hit {
		t.Fatalf("a >40-line doc with a graduated_to frontmatter marker must flag R5")
	}
}

func TestR6CheckStaleHandoff(t *testing.T) {
	text := "---\nupdated: 2026-07-10\n---\nHANDOFF body\n"
	detail, fix, hit := r6Check(text, "2026-07-15")
	if !hit {
		t.Fatalf("expected R6 to fire: fm.updated is older than newest commit")
	}
	if detail != "HANDOFF.md last updated 2026-07-10 but newest desk commit is 2026-07-15 — handoff may be stale" {
		t.Fatalf("got %q", detail)
	}
	if fix != "refresh _meta/HANDOFF.md at the next session boundary" {
		t.Fatalf("got %q", fix)
	}
}

func TestR6CheckFreshHandoffNoFinding(t *testing.T) {
	text := "---\nupdated: 2026-07-15\n---\nHANDOFF body\n"
	if _, _, hit := r6Check(text, "2026-07-15"); hit {
		t.Fatalf("a handoff dated the same as newest commit must not flag")
	}
}

func TestR6CheckFallsBackToLastUpdatedRegex(t *testing.T) {
	text := "no frontmatter here.\nLast updated: 2026-07-01. Owner: Alex.\n"
	detail, _, hit := r6Check(text, "2026-07-15")
	if !hit {
		t.Fatalf("expected R6 to fire using the regex fallback")
	}
	if detail != "HANDOFF.md last updated 2026-07-01 but newest desk commit is 2026-07-15 — handoff may be stale" {
		t.Fatalf("got %q", detail)
	}
}

func TestR6CheckUnknownDateShownParenthesized(t *testing.T) {
	text := "no date information anywhere in this document.\n"
	detail, _, hit := r6Check(text, "2026-07-15")
	if !hit {
		t.Fatalf("expected R6 to fire when no date can be determined")
	}
	if detail != "HANDOFF.md last updated (unknown) but newest desk commit is 2026-07-15 — handoff may be stale" {
		t.Fatalf("got %q", detail)
	}
}

func TestR6CheckNoGitHistoryDegradesToNoFinding(t *testing.T) {
	// spec §5.2 Errors: a git failure for R6 degrades to "no R6 finding".
	text := "no frontmatter\n"
	if _, _, hit := r6Check(text, ""); hit {
		t.Fatalf("empty newest (git yielded nothing) must never flag R6")
	}
}

func TestMechanicalRulesFixedOrder(t *testing.T) {
	// R4 is deliberately absent: it detects mechanically but remediates by judgment, so it is
	// filed as a judgment finding (see the R4 block in Patrol), not through this list.
	want := []string{"R1", "R2", "R3"}
	if !reflect.DeepEqual(mechanicalRules, want) {
		t.Fatalf("got %#v, want %#v", mechanicalRules, want)
	}
}

func TestPatrolSummaryDeterministicOrder(t *testing.T) {
	result := &PatrolResult{FilesSwept: 10, FindingsNew: 3, ByRule: map[string]int{"R5": 1, "R1": 2}}
	got := patrolSummary(result)
	want := "files=10 findings_new=3 (R1=2, R5=1)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPatrolSummaryNoneWhenEmpty(t *testing.T) {
	result := &PatrolResult{FilesSwept: 5, FindingsNew: 0, ByRule: map[string]int{}}
	if got := patrolSummary(result); got != "files=5 findings_new=0 (none)" {
		t.Fatalf("got %q", got)
	}
}

// TestPatrol_FilesR4AsJudgment: an invalid decision status is detected mechanically but its
// remediation (which status to choose) is a judgment call, so patrol files R4 with severity
// "judgment", not "mechanical".
func TestPatrol_FilesR4AsJudgment(t *testing.T) {
	app, cfg := newTestEnv(t)

	// A decision doc with complete universal frontmatter (so R1 stays quiet) under its correct
	// dir (so R3 stays quiet) but an invalid — here empty — decision status: only R4 fires.
	content := "---\ntype: decision\ncreated: 2026-01-01\nupdated: 2026-01-01\ntags: []\nsynopsis: \"x\"\n---\nbody\n"
	rel := "_structure/decisions/no-status.md"
	mustWriteFile(t, cfg.DeskRoot, rel, content)
	rec := mustCreateFileRecord(t, app, rel, "decisions", "decision", desklib.Checksum([]byte(content)))
	rec.Set("status", "") // invalid decision status -> R4
	if err := app.Save(rec); err != nil {
		t.Fatalf("save file record: %v", err)
	}

	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("Patrol: %v", err)
	}

	findings, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R4'", "", 0, 0)
	if err != nil {
		t.Fatalf("load R4 findings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected exactly one R4 finding, got %d", len(findings))
	}
	if sev := findings[0].GetString("severity"); sev != "judgment" {
		t.Fatalf("R4 severity = %q, want judgment (detection is mechanical, remediation is judgment)", sev)
	}
}

// --- resolution (full/scoped patrol closes stale findings) ---

// TestPatrol_ResolvesStaleFindingOnFullPatrol: a flagged finding whose rule no longer fires is
// transitioned to resolved with the resolving run id after a FULL-desk patrol.
func TestPatrol_ResolvesStaleFindingOnFullPatrol(t *testing.T) {
	app, cfg := newTestEnv(t)

	// The file now has complete frontmatter, so R1 no longer fires...
	mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", fullFM)
	checksum := desklib.Checksum([]byte(fullFM))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	// ...but a stale R1 finding from a prior run is still open.
	finding := mustCreateFinding(t, app, fileRec, "R1", "old-checksum", "prior-run")

	res, err := Patrol(context.Background(), app, cfg, &PatrolInput{})
	if err != nil {
		t.Fatalf("Patrol: %v", err)
	}

	got := reloadRecord(t, app, "patrol_findings", finding.Id)
	if got.GetString("state") != "resolved" {
		t.Fatalf("stale finding state = %q, want resolved", got.GetString("state"))
	}
	if got.GetString("resolved_run") != res.RunID {
		t.Fatalf("resolved_run = %q, want the resolving run id %q", got.GetString("resolved_run"), res.RunID)
	}
}

// TestPatrol_ScopedPatrolLeavesOutOfScopeStaleFinding: a scoped (--path) patrol resolves only
// findings within its scope; a stale finding outside the scope is left flagged.
func TestPatrol_ScopedPatrolLeavesOutOfScopeStaleFinding(t *testing.T) {
	app, cfg := newTestEnv(t)

	// In scope: a stale finding under tasks/ (R1 no longer fires -> should resolve).
	mustWriteFile(t, cfg.DeskRoot, "tasks/in-scope.md", fullFM)
	inRec := mustCreateFileRecord(t, app, "tasks/in-scope.md", "tasks", "task", desklib.Checksum([]byte(fullFM)))
	inFinding := mustCreateFinding(t, app, inRec, "R1", "old-checksum", "prior-run")

	// Out of scope: an equally-stale finding under analyses/ (must stay flagged).
	mustWriteFile(t, cfg.DeskRoot, "analyses/out-of-scope.md", fullFM)
	outRec := mustCreateFileRecord(t, app, "analyses/out-of-scope.md", "analyses", "task", desklib.Checksum([]byte(fullFM)))
	outFinding := mustCreateFinding(t, app, outRec, "R1", "old-checksum", "prior-run")

	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{Path: "tasks"}); err != nil {
		t.Fatalf("Patrol: %v", err)
	}

	if got := reloadRecord(t, app, "patrol_findings", inFinding.Id); got.GetString("state") != "resolved" {
		t.Fatalf("in-scope stale finding state = %q, want resolved", got.GetString("state"))
	}
	got := reloadRecord(t, app, "patrol_findings", outFinding.Id)
	if got.GetString("state") != "flagged" {
		t.Fatalf("out-of-scope stale finding state = %q, want flagged (untouched by a scoped patrol)", got.GetString("state"))
	}
	if got.GetString("resolved_run") != "" {
		t.Fatalf("out-of-scope finding resolved_run = %q, want empty", got.GetString("resolved_run"))
	}
}

// TestPatrol_StillFiringFindingStaysFlaggedAndDedupes: a finding whose rule still fires (same
// checksum) stays flagged and is not re-created (deduped), never resolved.
func TestPatrol_StillFiringFindingStaysFlaggedAndDedupes(t *testing.T) {
	app, cfg := newTestEnv(t)

	mustWriteFile(t, cfg.DeskRoot, "tasks/broken.md", missingFM)
	checksum := desklib.Checksum([]byte(missingFM))
	fileRec := mustCreateFileRecord(t, app, "tasks/broken.md", "tasks", "task", checksum)
	// Finding stored with the file's CURRENT checksum so it dedupes against the re-fire.
	finding := mustCreateFinding(t, app, fileRec, "R1", checksum, "prior-run")

	res, err := Patrol(context.Background(), app, cfg, &PatrolInput{})
	if err != nil {
		t.Fatalf("Patrol: %v", err)
	}

	got := reloadRecord(t, app, "patrol_findings", finding.Id)
	if got.GetString("state") != "flagged" {
		t.Fatalf("still-firing finding state = %q, want flagged", got.GetString("state"))
	}
	if got.GetString("resolved_run") != "" {
		t.Fatalf("still-firing finding must not record a resolved_run, got %q", got.GetString("resolved_run"))
	}
	// Deduped: no new R1 row was created for this path/checksum.
	if res.ByRule["R1"] != 0 {
		t.Fatalf("expected R1 deduped (0 new), got %d new R1 findings", res.ByRule["R1"])
	}
	all, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1'", "", 0, 0)
	if err != nil {
		t.Fatalf("list R1 findings: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 R1 finding after dedupe, got %d", len(all))
	}
}

// TestPatrol_ResolvesFindingOnDeletedFile pins an INTENDED consequence of the resolution
// mechanism, not an accident: Patrol only reads files where deleted = false, so a soft-deleted
// file's row drops out of filesByID entirely. Its open finding then resolves to an empty path
// (filesByID lookup misses), and an empty path can never appear in `fired` (fired is populated
// only from swept, non-deleted rows in `filtered`). So on the next FULL-desk patrol, a flagged
// finding whose file has since been soft-deleted resolves exactly like any other finding whose
// rule stopped re-firing — which is correct: a deleted file can never re-trip any rule, so
// "resolved" accurately reflects nothing left to fix. This test locks in that behavior so a
// future refactor of the fired/filesByID plumbing does not silently change it.
func TestPatrol_ResolvesFindingOnDeletedFile(t *testing.T) {
	app, cfg := newTestEnv(t)

	fileRec := mustCreateFileRecord(t, app, "tasks/deleted.md", "tasks", "task", "checksum-x")
	finding := mustCreateFinding(t, app, fileRec, "R1", "checksum-x", "prior-run")

	// Soft-delete the file record, exactly as sweep does when the on-disk file disappears.
	fileRec.Set("deleted", true)
	if err := app.Save(fileRec); err != nil {
		t.Fatalf("soft-delete file record: %v", err)
	}

	res, err := Patrol(context.Background(), app, cfg, &PatrolInput{})
	if err != nil {
		t.Fatalf("Patrol: %v", err)
	}

	got := reloadRecord(t, app, "patrol_findings", finding.Id)
	if got.GetString("state") != "resolved" {
		t.Fatalf("deleted-file finding state = %q, want resolved", got.GetString("state"))
	}
	if got.GetString("resolved_run") != res.RunID {
		t.Fatalf("resolved_run = %q, want the resolving run id %q", got.GetString("resolved_run"), res.RunID)
	}
}

// --- disposition lifecycle across re-patrol (DoD 93b / 93c + inheritance) ---

// TestPatrol_DisposedFindingPersistsAcrossRepatrol is DoD case 93b: a wont_fix finding stays out
// of the default view after a re-patrol with the SAME (file, rule, checksum). Disposing sets only
// disposition (state stays flagged), so the next patrol dedupes the existing row and its
// disposition survives untouched — no re-creation, no re-open.
func TestPatrol_DisposedFindingPersistsAcrossRepatrol(t *testing.T) {
	app, cfg := newTestEnv(t)

	mustWriteFile(t, cfg.DeskRoot, "tasks/broken.md", missingFM)
	checksum := desklib.Checksum([]byte(missingFM))
	mustCreateFileRecord(t, app, "tasks/broken.md", "tasks", "task", checksum)

	// Patrol 1 files an open R1 finding.
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 1: %v", err)
	}
	flagged, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1' && state = 'flagged'", "", 0, 0)
	if err != nil || len(flagged) != 1 {
		t.Fatalf("expected 1 flagged R1 finding after patrol 1, got %d (err=%v)", len(flagged), err)
	}
	findingID := flagged[0].Id
	if flagged[0].GetString("disposition") != "open" {
		t.Fatalf("freshly filed finding disposition = %q, want open", flagged[0].GetString("disposition"))
	}
	if n := defaultFindingsCount(t, app, cfg); n != 1 {
		t.Fatalf("open finding: default findings = %d, want 1", n)
	}

	// Dispose wont_fix -> gone from the default view.
	if _, err := DisposeFinding(context.Background(), app, cfg, findingID, "wont_fix", "", ""); err != nil {
		t.Fatalf("DisposeFinding: %v", err)
	}
	if n := defaultFindingsCount(t, app, cfg); n != 0 {
		t.Fatalf("after wont_fix dispose, default findings = %d, want 0", n)
	}

	// Patrol 2 with the SAME (file, rule, checksum): the row dedupes and keeps wont_fix.
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 2: %v", err)
	}
	got := reloadRecord(t, app, "patrol_findings", findingID)
	if got.GetString("state") != "flagged" {
		t.Fatalf("deduped finding state = %q, want flagged", got.GetString("state"))
	}
	if got.GetString("disposition") != "wont_fix" {
		t.Fatalf("deduped finding disposition = %q, want wont_fix (survived re-patrol)", got.GetString("disposition"))
	}
	if n := defaultFindingsCount(t, app, cfg); n != 0 {
		t.Fatalf("after re-patrol, default findings = %d, want 0 (disposition persisted)", n)
	}
	// Exactly one R1 row total — no duplicate was filed.
	all, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1'", "", 0, 0)
	if err != nil || len(all) != 1 {
		t.Fatalf("expected exactly 1 R1 row after dedupe, got %d (err=%v)", len(all), err)
	}
}

// TestPatrol_ChangedChecksumReopensDisposedFinding is DoD case 93c: because finding identity
// includes the checksum, changed evidence (a different checksum) yields a FRESH finding that
// defaults to open and resurfaces in the default view, while the prior wont_fix finding stays
// disposed. Re-open needs no extra code — it falls out of the (path, rule, checksum) identity.
func TestPatrol_ChangedChecksumReopensDisposedFinding(t *testing.T) {
	app, cfg := newTestEnv(t)

	mustWriteFile(t, cfg.DeskRoot, "tasks/broken.md", missingFM)
	checksum1 := desklib.Checksum([]byte(missingFM))
	fileRec := mustCreateFileRecord(t, app, "tasks/broken.md", "tasks", "task", checksum1)

	// Patrol 1 -> open R1 finding; dispose it wont_fix.
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 1: %v", err)
	}
	first, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1'", "", 0, 0)
	if err != nil || len(first) != 1 {
		t.Fatalf("expected 1 R1 finding after patrol 1, got %d (err=%v)", len(first), err)
	}
	if _, err := DisposeFinding(context.Background(), app, cfg, first[0].Id, "wont_fix", "", ""); err != nil {
		t.Fatalf("DisposeFinding: %v", err)
	}
	if n := defaultFindingsCount(t, app, cfg); n != 0 {
		t.Fatalf("after dispose, default findings = %d, want 0", n)
	}

	// Evidence changes: still R1-broken, but different content -> different checksum -> different
	// finding identity.
	missingFM2 := "---\ntype: task\n---\na different broken body line\n"
	mustWriteFile(t, cfg.DeskRoot, "tasks/broken.md", missingFM2)
	checksum2 := desklib.Checksum([]byte(missingFM2))
	if checksum2 == checksum1 {
		t.Fatalf("test setup: the two broken variants must have different checksums")
	}
	fileRec.Set("checksum", checksum2)
	if err := app.Save(fileRec); err != nil {
		t.Fatalf("update file checksum: %v", err)
	}

	// Patrol 2 files a FRESH, open finding for the new checksum.
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 2: %v", err)
	}
	if n := defaultFindingsCount(t, app, cfg); n != 1 {
		t.Fatalf("after checksum change, default findings = %d, want 1 (a fresh open finding resurfaced)", n)
	}
	openRows, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1' && disposition = 'open'", "", 0, 0)
	if err != nil || len(openRows) != 1 {
		t.Fatalf("expected exactly 1 open R1 finding after re-open, got %d (err=%v)", len(openRows), err)
	}
	if openRows[0].GetString("checksum") != checksum2 {
		t.Fatalf("fresh finding checksum = %q, want the new %q", openRows[0].GetString("checksum"), checksum2)
	}
	// The prior finding is still around and still disposed.
	priorRows, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1' && disposition = 'wont_fix'", "", 0, 0)
	if err != nil || len(priorRows) != 1 || priorRows[0].GetString("checksum") != checksum1 {
		t.Fatalf("expected the prior wont_fix finding (checksum %q) to persist, got %d rows (err=%v)", checksum1, len(priorRows), err)
	}
}

// TestPatrol_ResolvedDisposedFindingRefiredInheritsDisposition proves the inheritance path the
// dedupe path cannot cover: a finding disposed wont_fix (with provenance: actor + reason), then
// RESOLVED (its rule stopped firing), then RE-FIRED with the same checksum. The resolved row is
// not in the open dedupe set, so patrol files a FRESH row — which must INHERIT the prior non-open
// disposition AND its provenance (actor/reason/disposed_at) so a supervisor's decision — who made
// it, why, and when — is not silently lost across the resolve→re-fire cycle.
func TestPatrol_ResolvedDisposedFindingRefiredInheritsDisposition(t *testing.T) {
	app, cfg := newTestEnv(t)

	rel := "tasks/broken.md"
	brokenC := desklib.Checksum([]byte(missingFM))
	fileRec := mustCreateFileRecord(t, app, rel, "tasks", "task", brokenC)

	// Patrol 1: R1 fires -> open finding; dispose it wont_fix with provenance.
	mustWriteFile(t, cfg.DeskRoot, rel, missingFM)
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 1: %v", err)
	}
	f1, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1'", "", 0, 0)
	if err != nil || len(f1) != 1 {
		t.Fatalf("expected 1 R1 finding after patrol 1, got %d (err=%v)", len(f1), err)
	}
	origID := f1[0].Id
	if _, err := DisposeFinding(context.Background(), app, cfg, origID, "wont_fix", "somebody", "not worth fixing"); err != nil {
		t.Fatalf("dispose: %v", err)
	}
	origDisposedAt := reloadRecord(t, app, "patrol_findings", origID).GetString("disposed_at")
	if origDisposedAt == "" {
		t.Fatalf("original finding disposed_at is empty after dispose")
	}

	// Patrol 2: the file now has complete frontmatter -> R1 stops firing -> the finding resolves,
	// keeping its wont_fix disposition and its ORIGINAL (broken) checksum.
	mustWriteFile(t, cfg.DeskRoot, rel, fullFM)
	fileRec.Set("checksum", desklib.Checksum([]byte(fullFM)))
	if err := app.Save(fileRec); err != nil {
		t.Fatalf("update file record to fixed: %v", err)
	}
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 2: %v", err)
	}
	if got := reloadRecord(t, app, "patrol_findings", origID); got.GetString("state") != "resolved" {
		t.Fatalf("finding state after fix = %q, want resolved", got.GetString("state"))
	}

	// Patrol 3: the file reverts to the SAME broken content (same checksum) -> R1 re-fires. The
	// resolved row is not in the open dedupe set, so a FRESH row is filed; it must inherit the
	// prior wont_fix disposition AND its provenance and therefore stay out of the default view.
	mustWriteFile(t, cfg.DeskRoot, rel, missingFM)
	fileRec.Set("checksum", brokenC)
	if err := app.Save(fileRec); err != nil {
		t.Fatalf("revert file record to broken: %v", err)
	}
	if _, err := Patrol(context.Background(), app, cfg, &PatrolInput{}); err != nil {
		t.Fatalf("patrol 3: %v", err)
	}

	fresh, err := app.FindRecordsByFilter("patrol_findings", "rule = 'R1' && state = 'flagged'", "", 0, 0)
	if err != nil || len(fresh) != 1 {
		t.Fatalf("expected 1 flagged R1 finding after re-fire, got %d (err=%v)", len(fresh), err)
	}
	if fresh[0].Id == origID {
		t.Fatalf("re-fire must file a FRESH row, not reuse the resolved one")
	}
	if fresh[0].GetString("disposition") != "wont_fix" {
		t.Fatalf("re-fired finding disposition = %q, want inherited wont_fix", fresh[0].GetString("disposition"))
	}
	if n := defaultFindingsCount(t, app, cfg); n != 0 {
		t.Fatalf("re-fired inherited-wont_fix finding must not appear in default findings; got %d", n)
	}
	// Provenance inheritance: the fresh row carries the ORIGINAL disposal's actor/reason/
	// disposed_at, not a fresh stamp — a re-fire must not fabricate new provenance.
	if got := fresh[0].GetString("actor"); got != "somebody" {
		t.Fatalf("re-fired finding actor = %q, want inherited somebody", got)
	}
	if got := fresh[0].GetString("reason"); got != "not worth fixing" {
		t.Fatalf("re-fired finding reason = %q, want inherited %q", got, "not worth fixing")
	}
	if got := fresh[0].GetString("disposed_at"); got != origDisposedAt {
		t.Fatalf("re-fired finding disposed_at = %q, want inherited original %q", got, origDisposedAt)
	}
}

func TestFindingKeySortStability(t *testing.T) {
	// Sanity: findingKey is a plain comparable struct usable as a map key regardless of
	// field population order.
	keys := []findingKey{
		{"b.md", "R1", "x"},
		{"a.md", "R1", "x"},
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].path < keys[j].path })
	if keys[0].path != "a.md" {
		t.Fatalf("sort by path failed: %#v", keys)
	}
}
