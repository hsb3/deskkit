// Scroll anchoring for streaming output. New streamed content must never yank a reader who has
// scrolled up back down to the bottom: the viewport only auto-follows when it was already at the
// bottom before the content update. The decision is a pure function of the pre-update scroll
// position so it can be unit-tested without a live viewport.
package tui

// shouldAutoFollow reports whether newly appended transcript content should scroll the viewport to
// the bottom. It follows only when the viewport was already pinned to the bottom (scroll percent at
// or past 1.0 before the update); a reader scrolled up keeps their position. The viewport reports a
// scroll percent of 1.0 whenever the content fits, so a fresh/short transcript still auto-follows.
func shouldAutoFollow(scrollPercentBefore float64) bool {
	return scrollPercentBefore >= 1.0
}
