// Package tools is the six-tool SEAM. It defines the exact input/output contracts
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
	Outcome    string `json:"outcome"`  // recorded | ignored | missing | stale | noop (destination exists)
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
	Kind string `json:"kind" jsonschema:"description=One of: live_files recent orphans uncollapsed findings summary adoption;required"`
	Days int    `json:"days,omitempty" jsonschema:"description=Window for 'recent'; default 7"`
}
