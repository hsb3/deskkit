// Markdown rendering for assistant answers. A glamour renderer is rebuilt on every
// WindowSizeMsg so its word-wrap always matches the current viewport inner width (a renderer
// built for the old width would wrap wrongly after a resize). Any renderer or render failure
// degrades to the raw text rather than crashing the UI.
//
// Streaming policy (a deliberate deviation from the plan's "re-render accumulated text per
// token", recorded in the build report): while a turn is streaming we render the in-flight
// answer as PLAIN text and only run glamour once, on the terminal event. Re-running a full
// markdown parse+render on every token is O(answer length) per token — quadratic over a long
// answer — and visibly stutters the stream; rendering plain while streaming and glamour on
// finalize keeps the stream smooth and still delivers formatted output the instant the answer
// completes. The plan explicitly sanctions this fallback.
package tui

import (
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

// minWrap floors the glamour word-wrap so a very narrow (or not-yet-sized) terminal still
// produces sane output instead of wrapping to one character per line.
const minWrap = 20

// glamourStyle resolves the markdown style ONCE, from a fixed default, never from a runtime
// terminal query. glamour's WithAutoStyle asks the terminal for its background color (an OSC 11
// query, plus a DSR probe) at renderer-build time — which, because the renderer is (re)built on
// WindowSizeMsg AFTER the Bubble Tea program has started its own stdin reader, races that reader
// and leaks the terminal's escape-sequence RESPONSE into the textarea as typed input. Pinning a
// static style removes the query entirely. GLAMOUR_STYLE still overrides for operators who want
// a light or custom theme; "auto" is rejected because it reintroduces the query.
func glamourStyle() string {
	if s := os.Getenv("GLAMOUR_STYLE"); s != "" && s != styles.AutoStyle {
		return s
	}
	return styles.DarkStyle
}

// newRenderer builds a glamour renderer whose word-wrap matches the given inner width, using a
// statically resolved style (never a terminal query — see glamourStyle). On any construction
// error it returns nil; renderMarkdown treats a nil renderer as "plain text".
func newRenderer(width int) *glamour.TermRenderer {
	if width < minWrap {
		width = minWrap
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	return r
}

// renderMarkdown formats text through the renderer, falling back to the raw text on a nil
// renderer or a render error — the UI never blocks on or crashes over markdown. The trailing
// newline glamour adds is trimmed by the caller when composing the transcript.
func renderMarkdown(r *glamour.TermRenderer, text string) string {
	if r == nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return out
}
