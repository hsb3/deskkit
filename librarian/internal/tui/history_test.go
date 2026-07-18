package tui

import "testing"

// TestHistory_EmptyIsInert: with no prior prompts, prev/next never move and leave the live draft
// untouched, so the arrow keys fall through to normal textarea cursor behavior.
func TestHistory_EmptyIsInert(t *testing.T) {
	h := newHistory(nil)
	if got, moved := h.prev("draft"); moved || got != "draft" {
		t.Errorf("prev on empty = (%q, %v), want (draft, false)", got, moved)
	}
	if got, moved := h.next("draft"); moved || got != "draft" {
		t.Errorf("next on empty = (%q, %v), want (draft, false)", got, moved)
	}
}

// TestHistory_WalkUpDown walks up through prompts (newest-first) then back down, and verifies the
// stashed draft is restored when stepping down past the newest entry.
func TestHistory_WalkUpDown(t *testing.T) {
	// newest-first: "third" is most recent.
	h := newHistory([]string{"third", "second", "first"})

	// Up from the live draft stashes it and recalls the newest.
	if got, moved := h.prev("my draft"); !moved || got != "third" {
		t.Fatalf("first prev = (%q, %v), want (third, true)", got, moved)
	}
	if got, moved := h.prev("third"); !moved || got != "second" {
		t.Fatalf("second prev = (%q, %v), want (second, true)", got, moved)
	}
	if got, moved := h.prev("second"); !moved || got != "first" {
		t.Fatalf("third prev = (%q, %v), want (first, true)", got, moved)
	}
	// At the oldest entry, further up does not move.
	if got, moved := h.prev("first"); moved || got != "first" {
		t.Fatalf("prev past oldest = (%q, %v), want (first, false)", got, moved)
	}

	// Walk back down.
	if got, moved := h.next("first"); !moved || got != "second" {
		t.Fatalf("first next = (%q, %v), want (second, true)", got, moved)
	}
	if got, moved := h.next("second"); !moved || got != "third" {
		t.Fatalf("second next = (%q, %v), want (third, true)", got, moved)
	}
	// Down past the newest restores the stashed draft and exits navigation.
	if got, moved := h.next("third"); !moved || got != "my draft" {
		t.Fatalf("next past newest = (%q, %v), want (my draft, true)", got, moved)
	}
	// Now out of navigation: another down is inert.
	if got, moved := h.next("my draft"); moved || got != "my draft" {
		t.Fatalf("next when not navigating = (%q, %v), want (my draft, false)", got, moved)
	}
}

// TestHistory_NextWithoutNavigatingIsInert: Down while editing the live draft (never stepped into
// history) does nothing, so it falls through to normal textarea behavior.
func TestHistory_NextWithoutNavigatingIsInert(t *testing.T) {
	h := newHistory([]string{"a"})
	if got, moved := h.next("draft"); moved || got != "draft" {
		t.Errorf("next without navigating = (%q, %v), want (draft, false)", got, moved)
	}
}

// TestHistory_AddRecordsNewestAndResets: a freshly-sent prompt becomes the newest recallable entry
// and resets any in-progress navigation back to the live draft.
func TestHistory_AddRecordsNewestAndResets(t *testing.T) {
	h := newHistory([]string{"old"})
	// Step into history, then send a new prompt: navigation must reset.
	_, _ = h.prev("draft")
	h.add("brand new")

	if got, moved := h.prev(""); !moved || got != "brand new" {
		t.Fatalf("prev after add = (%q, %v), want (brand new, true)", got, moved)
	}
	if got, moved := h.prev("brand new"); !moved || got != "old" {
		t.Fatalf("second prev after add = (%q, %v), want (old, true)", got, moved)
	}
}

// TestHistory_AddDedupesConsecutive: sending the same prompt twice in a row keeps a single entry
// (shell-history feel), and an empty prompt is never recorded.
func TestHistory_AddDedupesConsecutive(t *testing.T) {
	h := newHistory(nil)
	h.add("same")
	h.add("same")
	h.add("")
	if len(h.items) != 1 || h.items[0] != "same" {
		t.Fatalf("items = %v, want [same]", h.items)
	}
}

// TestHistory_DraftStashRestoredMidWalk: stepping up then immediately back down restores the exact
// stashed draft even after only one step.
func TestHistory_DraftStashRestoredMidWalk(t *testing.T) {
	h := newHistory([]string{"prev-prompt"})
	if got, _ := h.prev("half-typed"); got != "prev-prompt" {
		t.Fatalf("prev = %q, want prev-prompt", got)
	}
	if got, moved := h.next("prev-prompt"); !moved || got != "half-typed" {
		t.Fatalf("next = (%q, %v), want (half-typed, true)", got, moved)
	}
}
