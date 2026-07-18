// Input history: walking the current session's prior user prompts from the textarea, with the
// in-progress draft stashed on the first step into history and restored when the walk steps back
// past the newest entry. bubbles' textarea ships no history of its own, so this is app-land — but
// it is a pure type (no textarea, no Bubble Tea) so the walk logic is unit-testable in isolation
// and the model just feeds it the current input text and applies the returned value.
package tui

// history is the prompt-recall walker. items holds prior user prompts newest-first (items[0] is
// the most recent). pos is -1 while the user is editing their live draft (not navigating), and
// 0..len-1 while walking (0 = newest recalled prompt). stash preserves the live draft captured on
// the first step into history so it can be restored on the step back out.
type history struct {
	items []string
	pos   int
	stash string
}

// newHistory builds a walker seeded with prior prompts (newest-first), positioned on the live
// draft (not navigating). A resumed conversation seeds this from its loaded user messages.
func newHistory(items []string) history {
	return history{items: items, pos: -1}
}

// add records a freshly-sent prompt as the newest item and resets navigation back to the live
// draft. A prompt identical to the current newest is not duplicated (repeated identical sends stay
// one entry), matching the usual shell-history feel.
func (h *history) add(prompt string) {
	if prompt != "" && (len(h.items) == 0 || h.items[0] != prompt) {
		h.items = append([]string{prompt}, h.items...)
	}
	h.pos = -1
	h.stash = ""
}

// prev walks to an older prompt (the Up key at the textarea's first line). current is the live
// input, stashed on the first step so it can be restored later. It returns the text to place in
// the textarea and whether the walk actually moved (false when there is no history, or the walk is
// already at the oldest entry — the caller then leaves the input untouched).
func (h *history) prev(current string) (string, bool) {
	if len(h.items) == 0 {
		return current, false
	}
	if h.pos == -1 {
		h.stash = current
		h.pos = 0
		return h.items[0], true
	}
	if h.pos < len(h.items)-1 {
		h.pos++
		return h.items[h.pos], true
	}
	return h.items[h.pos], false // already at the oldest entry
}

// next walks toward newer prompts (the Down key at the textarea's last line). Stepping past the
// newest entry restores the stashed draft and exits navigation. It returns the text to place in
// the textarea and whether the walk moved (false when not currently navigating — the caller then
// lets the key do its normal thing).
func (h *history) next(current string) (string, bool) {
	if h.pos == -1 {
		return current, false
	}
	if h.pos == 0 {
		h.pos = -1
		draft := h.stash
		h.stash = ""
		return draft, true
	}
	h.pos--
	return h.items[h.pos], true
}
