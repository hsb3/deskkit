// Package tools is the PM tool family (spec §5.1): twelve thin adapters over the engine's
// core functions — the SAME functions every surface (MCP, CLI, TUI) calls, so §10.10 parity
// holds by construction. Nothing here re-implements transition/gate/cascade logic; every
// operation routes through modules/pm/engine (§4.1 single transition path).
//
// This file freezes the tool INPUT structs. Like the librarian's types.go, each field carries
// a `json` tag (name + optionality: non-omitempty = required) and an eino/invopop-style
// `jsonschema:"description=…"` tag — the single source both the eino loop's InferTool schema
// and the MCP surface's explicit InputSchema are reflected from (core/toolcore).
package tools

// ActorFields identify who acts, for the §3.6 audit trail (R2.5 plain strings). The calling
// surface supplies them: the CLI passes $USER / --actor with kind "human"; the MCP/agent
// surface passes its agent id (+ delegation parent). Unset defaults to actor "agent", kind
// "agent" — the model-facing surfaces are agent-driven by default.
type ActorFields struct {
	Actor            string `json:"actor,omitempty" jsonschema:"description=who acts (a human handle or an agent id); recorded verbatim in the audit trail"`
	ActorKind        string `json:"actor_kind,omitempty" jsonschema:"description=human or agent; defaults to agent"`
	DelegationParent string `json:"delegation_parent,omitempty" jsonschema:"description=the parent agent/session id when acting under delegation"`
}

// GetContextInput — §5.2 single-call cold-start briefing.
type GetContextInput struct {
	StalledDays int `json:"stalled_days,omitempty" jsonschema:"description=override the stalled threshold in days (default 14 / PM_STALLED_DAYS)"`
}

// ListItemsInput — §5.1 filtered graph query.
type ListItemsInput struct {
	Phase   string `json:"phase,omitempty" jsonschema:"description=filter by phase: queue, work, review, or terminal"`
	Court   string `json:"court,omitempty" jsonschema:"description=filter by court: owner, desk, crew, vendor, or external-session"`
	Type    string `json:"type,omitempty" jsonschema:"description=filter by schema-v1/kit item type"`
	Blocked string `json:"blocked,omitempty" jsonschema:"description=filter by the blocked flag: true or false; omit for all"`
	Parent  string `json:"parent,omitempty" jsonschema:"description=filter to the direct children of this item id"`
}

// GetItemInput — §5.1 one item + notes, deps, recent transitions, ancestor chain.
type GetItemInput struct {
	ItemID string `json:"item_id" jsonschema:"description=the item id"`
}

// CreateItemInput — §5.1 add a work item to the graph (phase starts at queue).
type CreateItemInput struct {
	Title    string `json:"title" jsonschema:"description=the item title"`
	Type     string `json:"type,omitempty" jsonschema:"description=schema-v1/kit item type (e.g. decision, task)"`
	Parent   string `json:"parent,omitempty" jsonschema:"description=parent item id; omit for a root item"`
	Court    string `json:"court,omitempty" jsonschema:"description=owner, desk, crew, vendor, or external-session"`
	Pointer  string `json:"pointer,omitempty" jsonschema:"description=doc path / issue URL / other locus"`
	Body     string `json:"body,omitempty" jsonschema:"description=long-form body: narrative, acceptance criteria, or spec, stored inline on the item"`
	Severity string `json:"severity,omitempty" jsonschema:"description=low, medium, or high"`
	Priority int    `json:"priority,omitempty" jsonschema:"description=ordinal within a court/queue"`
	ActorFields
}

// UpdateItemInput — §5.1 edit first-class fields (version-checked, R2.6). An omitted/empty
// field is left unchanged (priority 0 = unchanged); phase and blocked are NOT editable here —
// use transition_item / block_item. A status_label naming a different phase is a transition
// request routed through the machine + gates (§3.3).
type UpdateItemInput struct {
	ItemID      string `json:"item_id" jsonschema:"description=the item id"`
	Version     int    `json:"version" jsonschema:"description=the version you read; refused on mismatch"`
	Title       string `json:"title,omitempty" jsonschema:"description=new title; empty = unchanged"`
	Type        string `json:"type,omitempty" jsonschema:"description=new schema-v1/kit type; empty = unchanged"`
	Court       string `json:"court,omitempty" jsonschema:"description=new court; empty = unchanged"`
	Pointer     string `json:"pointer,omitempty" jsonschema:"description=new document pointer; empty = unchanged"`
	Body        string `json:"body,omitempty" jsonschema:"description=new body; empty = unchanged (clearing a set body is engine-only for now)"`
	Severity    string `json:"severity,omitempty" jsonschema:"description=new severity; empty = unchanged"`
	Priority    int    `json:"priority,omitempty" jsonschema:"description=new priority; 0 = unchanged"`
	Properties  string `json:"properties,omitempty" jsonschema:"description=new properties JSON object; empty = unchanged"`
	StatusLabel string `json:"status_label,omitempty" jsonschema:"description=new status label; a label of a different phase is a gated transition request"`
	ActorFields
}

// TransitionItemInput — §4.1 the one generic transition tool: advance, demote, and reopen are
// all requests through (item, target_phase); the machine derives the edge kind.
type TransitionItemInput struct {
	ItemID      string `json:"item_id" jsonschema:"description=the item id"`
	TargetPhase string `json:"target_phase" jsonschema:"description=queue, work, review, or terminal"`
	Version     int    `json:"version" jsonschema:"description=the version you read; refused on mismatch"`
	ActorFields
}

// BlockItemInput / UnblockItemInput — §3.2 the blocked side-state.
type BlockItemInput struct {
	ItemID  string `json:"item_id" jsonschema:"description=the item id"`
	Version int    `json:"version" jsonschema:"description=the version you read; refused on mismatch"`
	Reason  string `json:"reason,omitempty" jsonschema:"description=why the item is blocked (audit detail)"`
	ActorFields
}

// UnblockItemInput clears the blocked side-state.
type UnblockItemInput struct {
	ItemID  string `json:"item_id" jsonschema:"description=the item id"`
	Version int    `json:"version" jsonschema:"description=the version you read; refused on mismatch"`
	Reason  string `json:"reason,omitempty" jsonschema:"description=why the block clears (audit detail)"`
	ActorFields
}

// AddNoteInput — §3.7 phase-scoped keyed note.
type AddNoteInput struct {
	ItemID string `json:"item_id" jsonschema:"description=the item id"`
	Key    string `json:"key" jsonschema:"description=the note key (e.g. rationale, handoff)"`
	Body   string `json:"body" jsonschema:"description=the note body"`
	ActorFields
}

// LinkItemsInput — §3.4 one typed dependency edge. is-blocked-by is stored as the inverse
// blocks edge (one graph representation).
type LinkItemsInput struct {
	From      string `json:"from" jsonschema:"description=the from item id (for blocks: the blocker)"`
	To        string `json:"to" jsonschema:"description=the to item id (for blocks: the blocked item)"`
	Kind      string `json:"kind" jsonschema:"description=blocks, is-blocked-by, or relates-to"`
	UnblockAt string `json:"unblock_at,omitempty" jsonschema:"description=for blocks edges: the blocker phase releasing the block (work, review, terminal)"`
	Cascade   string `json:"cascade,omitempty" jsonschema:"description=for blocks edges: auto, manual, auto-reopen, or permanent"`
	ActorFields
}

// ClaimItemInput / ReleaseItemInput — §3.6 claim TTL for multi-agent safety (R2.6).
type ClaimItemInput struct {
	ItemID  string `json:"item_id" jsonschema:"description=the item id"`
	Version int    `json:"version" jsonschema:"description=the version you read; refused on mismatch"`
	ActorFields
}

// ReleaseItemInput clears a claim (only the holder may release a live claim).
type ReleaseItemInput struct {
	ItemID  string `json:"item_id" jsonschema:"description=the item id"`
	Version int    `json:"version" jsonschema:"description=the version you read; refused on mismatch"`
	ActorFields
}
