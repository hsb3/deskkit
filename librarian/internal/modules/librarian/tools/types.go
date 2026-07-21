// Package tools is the seven-tool SEAM. It defines the exact input/output contracts
// (spec §5.1–§5.6), the registry + the registration-time autonomous-write gate (§5.4),
// and stub function bodies. Tool BODIES, the eino InvokableTool wrappers, and the MCP
// surface are LATER slices that only fill in these functions / ADD files in their own
// packages — the signatures and the registry here are frozen contracts they code against.
package tools

// --- §5.1 sweep ---

type SweepInput struct {
	// No parameters: sweep always covers the whole configured DESK_ROOT.
	_ struct{} `json:"-"`
}

type SweepResult struct {
	Total       int `json:"total"`
	Created     int `json:"created"`
	Updated     int `json:"updated"`
	Unchanged   int `json:"unchanged"`
	SoftDeleted int `json:"soft_deleted"`
}

// --- §5.2 patrol ---

type PatrolInput struct {
	// Optional: restrict to a single file/subtree (used by the file-created hook task).
	Path string `json:"path,omitempty" jsonschema:"description=Relative path to patrol; empty means whole desk"`
}

type PatrolResult struct {
	RunID       string         `json:"run_id"`
	FilesSwept  int            `json:"files_swept"`
	FindingsNew int            `json:"findings_new"`
	ByRule      map[string]int `json:"by_rule"`
}

// --- §5.3 propose_fix ---

type ProposeFixInput struct {
	RunID string   `json:"run_id,omitempty" jsonschema:"description=Optional patrol run to scope fixes; empty means all open fixable findings"`
	Rules []string `json:"rules,omitempty" jsonschema:"description=Optional rule filter; defaults to R1,R2,R3"`
}

type ProposedFix struct {
	FindingID  string `json:"finding_id"`
	RevisionID string `json:"revision_id"` // empty if not recorded (e.g. ignored/stale/noop)
	Path       string `json:"path"`
	Rule       string `json:"rule"`
	Action     string `json:"action"`   // edit | move
	NewPath    string `json:"new_path"` // for moves
	Outcome    string `json:"outcome"`  // recorded | ignored | missing | stale | noop (destination exists) | error
	Error      string `json:"error,omitempty"`
}

type ProposeFixResult struct {
	RunID    string        `json:"run_id"`
	Proposed []ProposedFix `json:"proposed"`
}

// --- §5.4 apply_fix ---

type ApplyFixInput struct {
	RunID       string   `json:"run_id,omitempty" jsonschema:"description=Scope to a run; applies its recorded, un-applied revisions"`
	RevisionIDs []string `json:"revision_ids,omitempty" jsonschema:"description=Optional explicit revision ids to apply"`
}

type ApplyOutcome struct {
	RevisionID string `json:"revision_id"`
	Path       string `json:"path"`
	Rule       string `json:"rule"`
	Outcome    string `json:"outcome"` // applied | ignored | missing | stale | noop (destination exists) | error
	Error      string `json:"error,omitempty"`
}

type ApplyFixResult struct {
	RunID    string         `json:"run_id"`
	Outcomes []ApplyOutcome `json:"outcomes"`
}

// --- §5.5 restore ---

type RestoreInput struct {
	RevisionID string `json:"revision_id,omitempty" jsonschema:"description=The revisions row id to reverse"`
	Path       string `json:"path,omitempty" jsonschema:"description=Alternative to revision_id (CLI --by-path): resolve to the latest applied, not-yet-restored revision whose path or new_path matches"`
}

type RestoreResult struct {
	RevisionID string `json:"revision_id"`
	Path       string `json:"path"`
	Restored   bool   `json:"restored"`
	Reopened   bool   `json:"reopened"` // finding flipped back to flagged
}

// --- §5.6 query ---

type QueryInput struct {
	Kind            string `json:"kind" jsonschema:"description=One of: live_files recent orphans uncollapsed findings summary adoption feedback search content;required"`
	Days            int    `json:"days,omitempty" jsonschema:"description=Window for 'recent'; default 7"`
	IncludeDisposed bool   `json:"include_disposed,omitempty" jsonschema:"description=Include disposed (acknowledged triaged wont_fix) findings; default false shows only open"`
	Term            string `json:"term,omitempty" jsonschema:"description=Substring to search for in indexed file content; required for the search kind"`
	Limit           int    `json:"limit,omitempty" jsonschema:"description=Max results for the search kind; default 20"`
	Path            string `json:"path,omitempty" jsonschema:"description=Desk-relative file path for the content kind"`
	ShowIndex       bool   `json:"show_index,omitempty" jsonschema:"description=For the orphans kind also show by-design-unreferenced index/entry files such as README and INDEX; default false"`
}

// --- record_feedback ---
//
// The librarian's store-native feedback log. Records one row in the feedback collection when a
// tool fails or a desk convention does not fit (kind=problem) or when the user explicitly asks
// for a recording (kind=feedback). A DB-only write — not gated by LIBRARIAN_AUTONOMOUS_WRITES.
// Descriptions are comma-free on purpose: eino's InferTool splits the jsonschema tag on
// unescaped commas, so a comma would truncate the text the model sees.

type RecordFeedbackInput struct {
	Kind    string `json:"kind" jsonschema:"description=Entry type: 'problem' when a tool failed or a desk convention did not fit or 'feedback' when the user explicitly asked to record something;required"`
	Summary string `json:"summary" jsonschema:"description=One-line summary of the problem or feedback;required"`
	Detail  string `json:"detail,omitempty" jsonschema:"description=Optional longer detail such as full error text or reasoning"`
	Context string `json:"context,omitempty" jsonschema:"description=Optional note on what the agent was doing when the entry was recorded such as the trigger tool or turn"`
	Source  string `json:"source,omitempty" jsonschema:"description=Who originated the entry: 'agent' by default or 'user' when the user asked for the recording"`
}

type RecordFeedbackResult struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
