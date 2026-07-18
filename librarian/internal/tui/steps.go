// Rendering of tool steps in the transcript. Collapsed (the default) a step is one faint
// line — `▸ tool(args-summary) ✓` on success, `✗` on failure — so a tool-using turn stays
// scannable. ctrl+t expands every step to its pretty-printed arguments and a truncated
// result or error body. Any pre-tool commentary the model streamed before calling the tool is
// carried on the step (retagged from the answer bubble in model.go) and shown dim.
package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// step is one tool invocation within a turn. commentary holds any tokens the model streamed in
// the same model step BEFORE the tool_start (retagged out of the answer bubble). failed is set
// when the engine reported the call via tool_end.Err (the sole failed-call signal, per the
// engine contract); result carries a successful call's response. done marks that the matching
// tool_end has landed (a still-running call renders with an ellipsis rather than a badge).
type step struct {
	tool       string
	callID     string
	args       string
	result     string
	errText    string
	commentary string
	failed     bool
	done       bool
}

// argsSummary collapses a step's arguments to a single short line for the collapsed view. JSON
// is compacted; anything else is whitespace-flattened. Long summaries are elided.
func argsSummary(args string) string {
	s := strings.TrimSpace(args)
	if s == "" {
		return ""
	}
	var buf bytes.Buffer
	if json.Valid([]byte(s)) {
		if err := json.Compact(&buf, []byte(s)); err == nil {
			s = buf.String()
		}
	}
	s = strings.Join(strings.Fields(s), " ")
	return truncate(s, 48)
}

// prettyArgs pretty-prints a step's arguments for the expanded view, falling back to the raw
// string when it is not valid JSON.
func prettyArgs(args string) string {
	s := strings.TrimSpace(args)
	if s == "" {
		return "(none)"
	}
	var out bytes.Buffer
	if json.Indent(&out, []byte(s), "", "  ") == nil {
		return out.String()
	}
	return s
}

// truncate shortens s to at most n runes, appending an ellipsis when it cuts.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// renderSteps renders a turn's steps. Collapsed: one faint line each. Expanded (showSteps):
// the collapsed line plus pretty args, a truncated result/error, and any dim commentary.
func renderSteps(sts []step, st styleSet, showSteps bool) string {
	if len(sts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range sts {
		badge := "…"
		badgeStyle := st.step
		switch {
		case !s.done:
			badge, badgeStyle = "…", st.step
		case s.failed:
			badge, badgeStyle = "✗", st.stepErr
		default:
			badge, badgeStyle = "✓", st.stepOK
		}
		head := st.step.Render(fmt.Sprintf("▸ %s(%s) ", s.tool, argsSummary(s.args)))
		b.WriteString(head + badgeStyle.Render(badge) + "\n")

		if showSteps {
			if s.commentary != "" {
				b.WriteString(st.step.Render("  "+strings.TrimSpace(s.commentary)) + "\n")
			}
			b.WriteString(st.step.Render("  args: "+indentBlock(prettyArgs(s.args))) + "\n")
			if s.failed {
				b.WriteString(st.stepErr.Render("  error: "+truncate(oneLine(s.errText), 200)) + "\n")
			} else if s.result != "" {
				b.WriteString(st.step.Render("  result: "+truncate(oneLine(s.result), 200)) + "\n")
			}
		}
	}
	return b.String()
}

// oneLine flattens whitespace so a result/error body stays on the expanded line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// indentBlock keeps a multi-line pretty-printed block aligned under the "args:" label.
func indentBlock(s string) string {
	return strings.ReplaceAll(s, "\n", "\n  ")
}
