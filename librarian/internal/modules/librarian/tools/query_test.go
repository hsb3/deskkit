package tools

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestSplitCommitDate(t *testing.T) {
	cases := map[string]string{
		"abc123|2026-07-15": "2026-07-15",
		"":                  "",
		"nodateformat":      "",
	}
	for v, want := range cases {
		if got := splitCommitDate(v); got != want {
			t.Errorf("splitCommitDate(%q) = %q, want %q", v, got, want)
		}
	}
}

func TestCutoffDate(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	if got := cutoffDate(7, now); got != "2026-07-09" {
		t.Fatalf("got %q, want 2026-07-09", got)
	}
	if got := cutoffDate(0, now); got != "2026-07-16" {
		t.Fatalf("got %q, want 2026-07-16", got)
	}
}

func TestWithinRecentWindow(t *testing.T) {
	cases := []struct {
		date, cutoff string
		want         bool
	}{
		{"2026-07-15", "2026-07-09", true},
		{"2026-07-09", "2026-07-09", true}, // boundary is inclusive
		{"2026-07-01", "2026-07-09", false},
		{"", "2026-07-09", false}, // empty date (no git history) never counts
	}
	for _, c := range cases {
		if got := withinRecentWindow(c.date, c.cutoff); got != c.want {
			t.Errorf("withinRecentWindow(%q,%q) = %v, want %v", c.date, c.cutoff, got, c.want)
		}
	}
}

func TestIsOrphan(t *testing.T) {
	cases := []struct {
		row        fileRow
		secretsDir string
		want       bool
	}{
		{fileRow{Path: "docs/spec.md", EntityType: ""}, "_meta/secrets", true},
		{fileRow{Path: "docs/spec.md", EntityType: "analysis"}, "_meta/secrets", false}, // has a type
		{fileRow{Path: "README.txt", EntityType: ""}, "_meta/secrets", false},           // not .md
		{fileRow{Path: "_meta/HANDOFF.md", EntityType: ""}, "_meta/secrets", false},     // under _meta/
		{fileRow{Path: "_meta/secrets/key.md", EntityType: ""}, "_meta/secrets", false}, // under SECRETS_DIR
		{fileRow{Path: "custom-secrets/k.md", EntityType: ""}, "custom-secrets", false}, // configurable SECRETS_DIR
		// Non-entity infrastructure is not an orphan, keyed off dir_kind (spec §5.6): dotted
		// infra dirs and the memory store are outside the desk taxonomy, not misfiled content.
		{fileRow{Path: ".claude/agents/reviewer.md", EntityType: "", DirKind: "infra"}, "_meta/secrets", false},
		{fileRow{Path: ".agents/skills/x.md", EntityType: "", DirKind: "infra"}, "_meta/secrets", false},
		{fileRow{Path: ".claude/memory/note.md", EntityType: "", DirKind: "memory"}, "_meta/secrets", false},
		// A genuinely misfiled .md in a non-infra dir is still an orphan.
		{fileRow{Path: "scratch/loose.md", EntityType: "", DirKind: "other"}, "_meta/secrets", true},
	}
	for _, c := range cases {
		if got := isOrphan(c.row, c.secretsDir); got != c.want {
			t.Errorf("isOrphan(%+v, %q) = %v, want %v", c.row, c.secretsDir, got, c.want)
		}
	}
}

func TestGroupFindingsByRule(t *testing.T) {
	findings := []findingRow{
		{Path: "tasks/b.md", Rule: "R1", Detail: "missing universal frontmatter: tags"},
		{Path: "tasks/a.md", Rule: "R1", Detail: "missing universal frontmatter: synopsis"},
		{Path: "journal/y.md", Rule: "R3", Detail: "type 'task' but file lives under 'journal' (expected tasks/)"},
	}
	got := groupFindingsByRule(findings)
	want := map[string][]findingBrief{
		"R1": {
			{Path: "tasks/a.md", Detail: "missing universal frontmatter: synopsis"},
			{Path: "tasks/b.md", Detail: "missing universal frontmatter: tags"},
		},
		"R3": {
			{Path: "journal/y.md", Detail: "type 'task' but file lives under 'journal' (expected tasks/)"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestBuildSummary(t *testing.T) {
	files := []fileRow{
		{DirKind: "root"}, {DirKind: "meta"}, {DirKind: "meta"}, {DirKind: "decisions"},
	}
	findings := []findingRow{
		{Rule: "R1", Severity: "mechanical"},
		{Rule: "R1", Severity: "mechanical"},
		{Rule: "R5", Severity: "judgment"},
	}
	got := buildSummary(files, findings)
	if got.Kind != "summary" {
		t.Fatalf("kind = %q", got.Kind)
	}
	if got.FilesTotal != 4 {
		t.Fatalf("FilesTotal = %d", got.FilesTotal)
	}
	wantDirKind := map[string]int{"root": 1, "meta": 2, "decisions": 1}
	if !reflect.DeepEqual(got.FilesByDirKind, wantDirKind) {
		t.Fatalf("FilesByDirKind = %#v, want %#v", got.FilesByDirKind, wantDirKind)
	}
	if got.OpenFindingsTotal != 3 {
		t.Fatalf("OpenFindingsTotal = %d", got.OpenFindingsTotal)
	}
	wantByRule := map[string]int{"R1": 2, "R5": 1}
	if !reflect.DeepEqual(got.OpenFindingsByRule, wantByRule) {
		t.Fatalf("OpenFindingsByRule = %#v, want %#v", got.OpenFindingsByRule, wantByRule)
	}
	wantBySeverity := map[string]int{"mechanical": 2, "judgment": 1}
	if !reflect.DeepEqual(got.OpenFindingsBySeverity, wantBySeverity) {
		t.Fatalf("OpenFindingsBySeverity = %#v, want %#v", got.OpenFindingsBySeverity, wantBySeverity)
	}
}

func TestAdoptionDate(t *testing.T) {
	if got := adoptionDate("2026-07-15 10:20:30.000Z"); got != "2026-07-15" {
		t.Fatalf("got %q", got)
	}
	if got := adoptionDate("short"); got != "short" {
		t.Fatalf("got %q", got)
	}
	if got := adoptionDate(""); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestToFileBrief(t *testing.T) {
	row := fileRow{
		Path: "tasks/x.md", DirKind: "tasks", EntityType: "task", Status: "",
		GraduatedTo: "", GitLastCommit: "abc|2026-07-15",
	}
	got := toFileBrief(row)
	want := fileBrief{
		Path: "tasks/x.md", DirKind: "tasks", EntityType: "task", Status: "",
		GraduatedTo: "", GitLastCommit: "abc|2026-07-15",
	}
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSortFileBriefsByCommitDateDesc(t *testing.T) {
	files := []fileBrief{
		{Path: "a.md", GitLastCommit: "h1|2026-07-01"},
		{Path: "b.md", GitLastCommit: "h2|2026-07-15"},
		{Path: "c.md", GitLastCommit: "h3|2026-07-10"},
	}
	sortFileBriefsByCommitDateDesc(files)
	want := []string{"b.md", "c.md", "a.md"}
	for i, w := range want {
		if files[i].Path != w {
			t.Fatalf("index %d: got %q, want %q (full: %#v)", i, files[i].Path, w, files)
		}
	}
}

func TestSortOrphanBriefs(t *testing.T) {
	files := []orphanBrief{{Path: "z.md"}, {Path: "a.md"}}
	sortOrphanBriefs(files)
	if files[0].Path != "a.md" || files[1].Path != "z.md" {
		t.Fatalf("got %#v", files)
	}
}

func TestSortFindingBriefs(t *testing.T) {
	findings := []findingBrief{
		{Path: "b.md", Detail: "z"},
		{Path: "a.md", Detail: "z"},
		{Path: "a.md", Detail: "a"},
	}
	sortFindingBriefs(findings)
	want := []findingBrief{{Path: "a.md", Detail: "a"}, {Path: "a.md", Detail: "z"}, {Path: "b.md", Detail: "z"}}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("got %#v, want %#v", findings, want)
	}
}

// A query against a store whose collections were never created (no prior migrate/serve)
// resolves each collection lookup to the bare database/sql sentinel sql.ErrNoRows —
// indistinguishable, at that call site, from a genuinely empty result. Query must turn
// that into an actionable message instead of leaking the driver-level string.
func TestTranslateUninitializedStoreError(t *testing.T) {
	if err := translateUninitializedStoreError(sql.ErrNoRows); !errors.Is(err, errStoreNotInitialized) {
		t.Fatalf("translateUninitializedStoreError(sql.ErrNoRows) = %v, want errStoreNotInitialized", err)
	}
	// Still recognized when wrapped (e.g. fmt.Errorf("...: %w", sql.ErrNoRows)), since
	// pocketbase's own lookups may wrap it on the way up.
	wrapped := fmt.Errorf("query %q: %w", "files", sql.ErrNoRows)
	if err := translateUninitializedStoreError(wrapped); !errors.Is(err, errStoreNotInitialized) {
		t.Fatalf("translateUninitializedStoreError(wrapped sql.ErrNoRows) = %v, want errStoreNotInitialized", err)
	}
	// Any other error (a real driver failure, an invalid filter) must pass through
	// unchanged — this translation is scoped to the uninitialized-store condition only.
	other := errors.New("disk I/O error")
	if err := translateUninitializedStoreError(other); !errors.Is(err, other) {
		t.Fatalf("translateUninitializedStoreError(other) = %v, want unchanged %v", err, other)
	}
}
