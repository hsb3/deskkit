package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
)

// RecordFeedback writes one row to the feedback collection and returns the created id plus a
// one-line confirmation. It is the librarian's store-native feedback log: the agent records a
// `problem` entry when a tool fails or a desk convention does not fit, and a `feedback` entry
// when the user explicitly asks it to record something.
//
// This is a DB-only write (no desk file is touched), so — like patrol filing findings — it is
// NOT gated behind LIBRARIAN_AUTONOMOUS_WRITES; that registration-time gate governs only the
// desk-file writers (apply_fix). Validation mirrors the collection's select enums so a bad
// kind/source surfaces a clear tool error rather than a raw PocketBase validation failure.
func RecordFeedback(ctx context.Context, app core.App, cfg *config.Config, in *RecordFeedbackInput) (*RecordFeedbackResult, error) {
	kind := strings.TrimSpace(in.Kind)
	switch kind {
	case "problem", "feedback":
	default:
		return nil, fmt.Errorf("record_feedback: kind must be \"problem\" or \"feedback\", got %q", in.Kind)
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, fmt.Errorf("record_feedback: summary is required")
	}

	// source defaults to "agent"; the model sets "user" when the user asked for the recording.
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "agent"
	}
	switch source {
	case "agent", "user":
	default:
		return nil, fmt.Errorf("record_feedback: source must be \"agent\" or \"user\", got %q", in.Source)
	}

	coll, err := app.FindCollectionByNameOrId("feedback")
	if err != nil {
		// A store that has never had its migrations applied resolves this lookup to the bare
		// sql.ErrNoRows sentinel; translate it into the same actionable message Query uses.
		return nil, translateUninitializedStoreError(err)
	}

	rec := core.NewRecord(coll)
	rec.Set("kind", kind)
	rec.Set("summary", summary)
	rec.Set("detail", in.Detail)
	rec.Set("source", source)
	rec.Set("context", strings.TrimSpace(in.Context))
	rec.Set("status", "open") // every new entry starts open (default/seed per the schema)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("record_feedback: save row: %w", err)
	}

	return &RecordFeedbackResult{
		ID:      rec.Id,
		Kind:    kind,
		Status:  "open",
		Message: fmt.Sprintf("Recorded %s entry %s (source=%s).", kind, rec.Id, source),
	}, nil
}
