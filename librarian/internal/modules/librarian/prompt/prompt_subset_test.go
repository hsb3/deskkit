package prompt

import (
	"regexp"
	"sort"
	"testing"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/librarian/tools"
)

// claimedToolRE matches a tool-list line in the embedded system prompt, e.g.
//
//   - query           read-only questions over the file index and findings (...)
//
// but deliberately NOT a boundary bullet, e.g.
//
//   - Never propose or apply a FIX to any path on the ignore list.
//
// The distinguishing signal is a lowercase_snake_case token immediately after "- ", followed
// by two-or-more spaces of column alignment. Boundary bullets are ordinary capitalized prose
// sentences, so the first word after the dash never matches [a-z_]+ there.
var claimedToolRE = regexp.MustCompile(`(?m)^\s+-\s+([a-z_]+)\s{2,}`)

// TestPromptToolList_SubsetOfExposedTools pins the invariant that the librarian's shipped
// system prompt (templates/librarian-system-prompt.txt, exposed here as Embedded()) never
// claims a tool the librarian does not actually hand to the agent loop on a DEFAULT desk
// (autonomous writes off, i.e. the zero-value config.Config).
//
// This is a red-before/green-after regression test: before the prompt was trimmed, it named
// an unconditional apply_fix and a restore tool. apply_fix is registration-time gated OFF
// unless autonomous writes are explicitly enabled, and restore is never exposed to the agent
// loop at all (it is a CLI/supervised-only human recovery action) — so on a default desk this
// test used to fail with both names reported as phantom tools. After the trim, the prompt's
// claimed tool set is exactly the exposed set and this test is green.
func TestPromptToolList_SubsetOfExposedTools(t *testing.T) {
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	t.Cleanup(toolcore.Reset)

	// Default desk: a zero-value config has autonomous writes off.
	exposedList := toolcore.ExposedTools(&config.Config{})
	exposed := make(map[string]bool, len(exposedList))
	for _, name := range exposedList {
		exposed[name] = true
	}

	matches := claimedToolRE.FindAllStringSubmatch(Embedded(), -1)
	if len(matches) == 0 {
		t.Fatalf("claimedToolRE matched no tool-list lines in the embedded prompt; the regex or the prompt format has drifted")
	}

	sortedExposed := append([]string(nil), exposedList...)
	sort.Strings(sortedExposed)

	for _, m := range matches {
		name := m[1]
		if !exposed[name] {
			t.Errorf("prompt claims tool %q, but it is not in the exposed librarian tool set on a default desk (phantom tool); exposed set = %v", name, sortedExposed)
		}
	}
}
