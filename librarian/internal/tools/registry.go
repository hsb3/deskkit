package tools

import "github.com/example/pocket-librarian/internal/config"

// ToolSpec is the harness-agnostic descriptor for one tool. The eino-loop and MCP slices
// map Name -> a constructed InvokableTool / MCP tool; the CLI maps Name -> the core
// function below. Description is the one-line text the model reads (spec §5 registration).
type ToolSpec struct {
	Name        string
	Description string
	// WritesFiles is true for the two tools that mutate the desk tree (apply_fix, restore).
	WritesFiles bool
	// AgentDefault: included in the autonomous serve agent's tool set unconditionally.
	AgentDefault bool
	// AgentGated: included in the autonomous serve agent's set ONLY when
	// LIBRARIAN_AUTONOMOUS_WRITES=true (the §5.4 registration-time gate). Applies to
	// apply_fix, the only write tool an autonomous loop may ever be granted.
	AgentGated bool
}

// Registry is the ordered list of the librarian's tools (spec §5, plus record_feedback). Order
// = the CLI/registration order.
//
// Gate encoding (spec §5.4):
//   - the autonomous serve agent gets {query, sweep, patrol, propose_fix} always, plus
//     apply_fix ONLY when AutonomousWrites is set;
//   - restore is CLI/supervised-only (recovery is a human action) — it is in neither the
//     AgentDefault nor the AgentGated set, so an autonomous loop never receives it;
//   - the CLI (AllTools) always builds the full six.
var Registry = []ToolSpec{
	{Name: "sweep", Description: "Reindex the desk tree into the files collection. Idempotent.", AgentDefault: true},
	{Name: "patrol", Description: "Flag rule violations (R1–R6) as findings; never writes files.", AgentDefault: true},
	{Name: "propose_fix", Description: "Compute a mechanical fix and record the file's original content before any write.", AgentDefault: true},
	{Name: "apply_fix", Description: "Commit a previously proposed fix to disk, byte-exact.", WritesFiles: true, AgentGated: true},
	{Name: "restore", Description: "Reverse a change to the exact recorded original.", WritesFiles: true},
	{Name: "query", Description: "Read-only questions over the file index and findings.", AgentDefault: true},
	// record_feedback writes only to the feedback collection (no desk file), so it is an
	// AgentDefault — available to every model-facing surface without the write gate that
	// governs apply_fix. WritesFiles stays false.
	{Name: "record_feedback", Description: "Record a problem or feedback entry to the store's feedback log.", AgentDefault: true},
}

// Spec returns the ToolSpec for name.
func Spec(name string) (ToolSpec, bool) {
	for _, t := range Registry {
		if t.Name == name {
			return t, true
		}
	}
	return ToolSpec{}, false
}

// AllTools returns the full six-tool set — the CLI/supervised surface (spec §5.4:
// "The CLI builds its own set including apply_fix regardless of the flag").
func AllTools() []ToolSpec {
	out := make([]ToolSpec, len(Registry))
	copy(out, Registry)
	return out
}

// AgentTools returns the tool set the AUTONOMOUS serve agent may be registered with,
// applying the §5.4 registration-time gate: apply_fix is excluded unless
// cfg.AutonomousWrites (LIBRARIAN_AUTONOMOUS_WRITES) is true; restore is always excluded.
// The eino-loop slice builds InvokableTools only for the names this returns — the gate is
// enforced by EXCLUSION FROM THE SLICE, not a runtime if inside the tool.
func AgentTools(cfg *config.Config) []ToolSpec {
	autonomousWrites := cfg != nil && cfg.AutonomousWrites
	var out []ToolSpec
	for _, t := range Registry {
		if t.AgentDefault || (t.AgentGated && autonomousWrites) {
			out = append(out, t)
		}
	}
	return out
}

// ToolNames maps a []ToolSpec to its names (convenience for the loop/MCP slices).
func ToolNames(specs []ToolSpec) []string {
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	return names
}
