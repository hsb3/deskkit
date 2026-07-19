package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrettyQuery_Summary(t *testing.T) {
	raw := json.RawMessage(`{"kind":"summary","files_total":5,
		"files_by_dir_kind":{"tasks":1,"infra":1},"open_findings_total":3,
		"open_findings_by_rule":{"R1":2,"R4":1},"open_findings_by_severity":{"mechanical":2,"judgment":1}}`)
	out, ok := prettyQuery("summary", raw)
	if !ok {
		t.Fatalf("summary should render")
	}
	for _, want := range []string{"files: 5 total", "by dir_kind", "infra", "by severity", "judgment", "R4"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary output missing %q:\n%s", want, out)
		}
	}
}

func TestPrettyQuery_ListOrphans(t *testing.T) {
	raw := json.RawMessage(`{"kind":"orphans","count":2,"files":[
		{"path":"scratch/loose.md","dir_kind":"other"},
		{"path":"tasks/t.md","dir_kind":"tasks"}]}`)
	out, ok := prettyQuery("orphans", raw)
	if !ok {
		t.Fatalf("orphans should render")
	}
	if !strings.Contains(out, "orphans: 2") || !strings.Contains(out, "scratch/loose.md") || !strings.Contains(out, "dir_kind") {
		t.Errorf("orphans output unexpected:\n%s", out)
	}
	// An all-empty optional column must be dropped, not printed blank.
	if strings.Contains(out, "graduated_to") {
		t.Errorf("empty column graduated_to should not appear:\n%s", out)
	}
}

func TestPrettyQuery_Findings(t *testing.T) {
	raw := json.RawMessage(`{"kind":"findings","count":2,"by_rule":{
		"R4":[{"path":"d/0001.md","detail":"decision status bogus"}],
		"R1":[{"path":"t/x.md","detail":"missing universal frontmatter"}]}}`)
	out, ok := prettyQuery("findings", raw)
	if !ok {
		t.Fatalf("findings should render")
	}
	// Rules are printed sorted: R1 before R4.
	if i, j := strings.Index(out, "R1 ("), strings.Index(out, "R4 ("); i < 0 || j < 0 || i > j {
		t.Errorf("findings groups should be sorted R1 before R4:\n%s", out)
	}
}

// TestPrettyQuery_ListAdoption pins the one list kind whose array key ("rows") differs from the
// "files" the others use — a wrong key would silently render a zero-row table, not fall back.
func TestPrettyQuery_ListAdoption(t *testing.T) {
	raw := json.RawMessage(`{"kind":"adoption","count":1,"rows":[
		{"date":"2026-07-17","event":"init","detail":"first run"}]}`)
	out, ok := prettyQuery("adoption", raw)
	if !ok {
		t.Fatalf("adoption should render")
	}
	for _, want := range []string{"adoption: 1", "2026-07-17", "init", "first run"} {
		if !strings.Contains(out, want) {
			t.Errorf("adoption output missing %q:\n%s", want, out)
		}
	}
}

func TestPrettyQuery_ListLiveFiles(t *testing.T) {
	raw := json.RawMessage(`{"kind":"live_files","count":1,"files":[
		{"path":"tasks/t.md","dir_kind":"tasks","entity_type":"task","status":"","graduated_to":"","git_last_commit":"abc|2026-07-15"}]}`)
	out, ok := prettyQuery("live_files", raw)
	if !ok || !strings.Contains(out, "tasks/t.md") || !strings.Contains(out, "git_last_commit") {
		t.Errorf("live_files output unexpected (ok=%v):\n%s", ok, out)
	}
}

func TestPrettyQuery_EmptyListStillRenders(t *testing.T) {
	raw := json.RawMessage(`{"kind":"orphans","count":0,"files":[]}`)
	out, ok := prettyQuery("orphans", raw)
	if !ok || !strings.Contains(out, "orphans: 0") {
		t.Errorf("empty orphans should render a count line, got ok=%v out=%q", ok, out)
	}
}

func TestPrettyQuery_UnknownKindFallsBack(t *testing.T) {
	// A kind the renderer does not format returns ok=false so the caller prints raw JSON.
	if _, ok := prettyQuery("live_files_v2", json.RawMessage(`{"kind":"x"}`)); ok {
		t.Errorf("unknown kind should return ok=false (fall back to raw JSON)")
	}
	// Malformed JSON also falls back rather than panicking.
	if _, ok := prettyQuery("summary", json.RawMessage(`{not json`)); ok {
		t.Errorf("malformed JSON should return ok=false")
	}
}
