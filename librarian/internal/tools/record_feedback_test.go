package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// TestFeedbackCollection_AppliesOnFreshStore proves the forward migration lands on a fresh store
// (newTestEnv runs RunAllMigrations) and that every content-bearing text field carries an
// EXPLICIT finite Max — never the bare Max==0 that silently caps at 5,000 chars.
func TestFeedbackCollection_AppliesOnFreshStore(t *testing.T) {
	app, _ := newTestEnv(t)
	c, err := app.FindCollectionByNameOrId("feedback")
	if err != nil {
		t.Fatalf("feedback collection missing after migrations: %v", err)
	}
	wantMax := map[string]int{"summary": 2000, "detail": 50000, "context": 2000}
	for name, want := range wantMax {
		tf, ok := c.Fields.GetByName(name).(*core.TextField)
		if !ok {
			t.Fatalf("field %q is not a TextField", name)
		}
		if tf.Max != want {
			t.Errorf("field %q Max = %d, want %d (explicit-Max convention; 0 is the 5,000-char trap)", name, tf.Max, want)
		}
	}
	for _, name := range []string{"summary"} {
		if tf, _ := c.Fields.GetByName(name).(*core.TextField); tf == nil || !tf.Required {
			t.Errorf("field %q should be required", name)
		}
	}
}

// TestRecordFeedback_CreatesRowWithDefaults exercises the DB write end-to-end on a fresh store
// (all migrations applied via newTestEnv): the feedback row round-trips with the right fields,
// and source/status default correctly when the caller omits source.
func TestRecordFeedback_CreatesRowWithDefaults(t *testing.T) {
	app, cfg := newTestEnv(t)

	res, err := RecordFeedback(context.Background(), app, cfg, &RecordFeedbackInput{
		Kind:    "problem",
		Summary: "sweep failed on a symlinked path",
		Detail:  "the tool returned an unexpected error walking .agents/",
		Context: "trigger=hook tool=sweep",
		// Source omitted → should default to "agent".
	})
	if err != nil {
		t.Fatalf("RecordFeedback: %v", err)
	}
	if res.ID == "" {
		t.Fatalf("expected a non-empty created id")
	}
	if res.Kind != "problem" || res.Status != "open" {
		t.Fatalf("result kind/status = %q/%q, want problem/open", res.Kind, res.Status)
	}
	if !strings.Contains(res.Message, res.ID) {
		t.Fatalf("confirmation message %q should mention the created id %q", res.Message, res.ID)
	}

	rec, err := app.FindRecordById("feedback", res.ID)
	if err != nil {
		t.Fatalf("reload feedback row: %v", err)
	}
	if got := rec.GetString("kind"); got != "problem" {
		t.Errorf("kind = %q, want problem", got)
	}
	if got := rec.GetString("summary"); got != "sweep failed on a symlinked path" {
		t.Errorf("summary = %q", got)
	}
	if got := rec.GetString("detail"); got != "the tool returned an unexpected error walking .agents/" {
		t.Errorf("detail = %q", got)
	}
	if got := rec.GetString("context"); got != "trigger=hook tool=sweep" {
		t.Errorf("context = %q", got)
	}
	if got := rec.GetString("source"); got != "agent" {
		t.Errorf("source = %q, want defaulted agent", got)
	}
	if got := rec.GetString("status"); got != "open" {
		t.Errorf("status = %q, want open", got)
	}
	if rec.GetString("created") == "" {
		t.Errorf("created autodate should be populated")
	}
}

// TestRecordFeedback_UserSource confirms an explicit source="user" (the value the model sets
// when the user asked for the recording) is persisted verbatim rather than defaulted.
func TestRecordFeedback_UserSource(t *testing.T) {
	app, cfg := newTestEnv(t)
	res, err := RecordFeedback(context.Background(), app, cfg, &RecordFeedbackInput{
		Kind:    "feedback",
		Summary: "please track that the handoff template is confusing",
		Source:  "user",
	})
	if err != nil {
		t.Fatalf("RecordFeedback: %v", err)
	}
	rec, err := app.FindRecordById("feedback", res.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := rec.GetString("source"); got != "user" {
		t.Errorf("source = %q, want user", got)
	}
	if got := rec.GetString("kind"); got != "feedback" {
		t.Errorf("kind = %q, want feedback", got)
	}
}

// TestRecordFeedback_LargeDetailSurvives proves a >5 KB detail body is stored intact — the bare-
// TextField Max==0 trap (implicit 5,000-char cap) would silently truncate it; the migration's
// explicit Max=50000 on `detail` prevents that.
func TestRecordFeedback_LargeDetailSurvives(t *testing.T) {
	app, cfg := newTestEnv(t)
	big := strings.Repeat("x", 8*1024) // 8 KB, well over the 5,000-char implicit cap
	res, err := RecordFeedback(context.Background(), app, cfg, &RecordFeedbackInput{
		Kind:    "problem",
		Summary: "large-detail round-trip",
		Detail:  big,
	})
	if err != nil {
		t.Fatalf("RecordFeedback: %v", err)
	}
	rec, err := app.FindRecordById("feedback", res.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := rec.GetString("detail"); len(got) != len(big) {
		t.Fatalf("detail length = %d, want %d (truncated? explicit Max not applied)", len(got), len(big))
	}
}

// TestRecordFeedback_Validation rejects an invalid kind, a blank summary, and an invalid source
// before any row is written.
func TestRecordFeedback_Validation(t *testing.T) {
	app, cfg := newTestEnv(t)
	cases := []struct {
		name string
		in   RecordFeedbackInput
	}{
		{"bad kind", RecordFeedbackInput{Kind: "bug", Summary: "x"}},
		{"blank summary", RecordFeedbackInput{Kind: "problem", Summary: "   "}},
		{"bad source", RecordFeedbackInput{Kind: "problem", Summary: "x", Source: "system"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := RecordFeedback(context.Background(), app, cfg, &c.in); err == nil {
				t.Fatalf("expected an error for %s", c.name)
			}
		})
	}
	// No rows should have been written by the rejected calls.
	recs, err := app.FindRecordsByFilter("feedback", "", "", 0, 0, dbx.Params{})
	if err != nil {
		t.Fatalf("list feedback: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 feedback rows after rejected calls, got %d", len(recs))
	}
}

// TestQueryFeedback_OpenNewestFirst pins the read-back: kind=feedback returns only OPEN entries,
// newest-first by created, each carrying id/kind/summary/status/created and the full detail body.
func TestQueryFeedback_OpenNewestFirst(t *testing.T) {
	app, cfg := newTestEnv(t)

	// Two open entries, recorded in order; a third resolved entry that must be excluded.
	first, err := RecordFeedback(context.Background(), app, cfg, &RecordFeedbackInput{
		Kind: "problem", Summary: "first", Detail: "first-detail",
	})
	if err != nil {
		t.Fatalf("record first: %v", err)
	}
	// created is a millisecond-precision autodate; space the inserts so "-created" ordering is
	// unambiguous rather than relying on tie-break behavior.
	time.Sleep(10 * time.Millisecond)
	second, err := RecordFeedback(context.Background(), app, cfg, &RecordFeedbackInput{
		Kind: "feedback", Summary: "second", Detail: "second-detail",
	})
	if err != nil {
		t.Fatalf("record second: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	resolved, err := RecordFeedback(context.Background(), app, cfg, &RecordFeedbackInput{
		Kind: "problem", Summary: "resolved-one", Detail: "resolved-detail",
	})
	if err != nil {
		t.Fatalf("record resolved: %v", err)
	}
	rec, err := app.FindRecordById("feedback", resolved.ID)
	if err != nil {
		t.Fatalf("reload resolved: %v", err)
	}
	rec.Set("status", "resolved")
	if err := app.Save(rec); err != nil {
		t.Fatalf("resolve row: %v", err)
	}

	raw, err := Query(context.Background(), app, cfg, &QueryInput{Kind: "feedback"})
	if err != nil {
		t.Fatalf("Query feedback: %v", err)
	}
	var out feedbackResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal feedback result: %v", err)
	}
	if out.Kind != "feedback" {
		t.Errorf("result kind = %q, want feedback", out.Kind)
	}
	if out.Count != 2 || len(out.Entries) != 2 {
		t.Fatalf("count/entries = %d/%d, want 2/2 (resolved entry must be excluded)", out.Count, len(out.Entries))
	}
	// Newest-first: the second-recorded entry sorts ahead of the first.
	if out.Entries[0].ID != second.ID || out.Entries[1].ID != first.ID {
		t.Fatalf("order = [%s %s], want newest-first [%s %s]",
			out.Entries[0].ID, out.Entries[1].ID, second.ID, first.ID)
	}
	if out.Entries[0].Summary != "second" || out.Entries[0].Detail != "second-detail" {
		t.Errorf("entry[0] = %+v, want summary/detail second/second-detail", out.Entries[0])
	}
	if out.Entries[0].Status != "open" || out.Entries[0].Created == "" {
		t.Errorf("entry[0] status/created = %q/%q, want open/<non-empty>", out.Entries[0].Status, out.Entries[0].Created)
	}
}
