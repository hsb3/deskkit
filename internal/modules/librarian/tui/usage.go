// Token-accounting helpers for the chat surface: the per-model context-window table (drives the
// header's ctx% gauge) and the compact K/M token formatter (header segment + per-turn footer).
//
// The window table is keyed by PUBLIC model-id substrings — these are the vendors' own published
// model names, not a deployment's identity, so they are neutrality-safe (they carry no person,
// org, or repo). Sizes are deliberately conservative neutral defaults; a desk that runs a model
// with a different window sets profile models.context_window (Config.LLMContextWindow), which
// overrides the table outright.
package tui

import (
	"fmt"
	"strconv"
	"strings"
)

// Neutral per-family context-window defaults, in tokens. Conservative on purpose: the gauge is a
// rough "how full is the context" cue, and a desk with an exact figure overrides via the profile.
const (
	windowAnthropic = 200_000   // claude family (opus / sonnet / haiku)
	windowOpenAI    = 400_000   // gpt / o-series / codex family
	windowGemini    = 1_000_000 // gemini / gemma family
	windowDefault   = 200_000   // unknown model + unknown provider: a safe floor
)

// contextWindow resolves the model's context-window size in tokens. An explicit override (from the
// profile / LLM_CONTEXT_WINDOW, >0) always wins. Otherwise a small table keyed by public model-id
// substrings is consulted, then the provider family as a fallback when the model id is unrecognized,
// then a conservative default. Pure, so it is resolved once in newModel and unit-tested directly.
func contextWindow(provider, model string, override int) int {
	if override > 0 {
		return override
	}
	m := strings.ToLower(model)
	switch {
	case containsAny(m, "gemini", "gemma"):
		return windowGemini
	case containsAny(m, "gpt", "codex", "o1", "o3", "o4"):
		return windowOpenAI
	case containsAny(m, "claude", "opus", "sonnet", "haiku"):
		return windowAnthropic
	}
	// Unrecognized model id: fall back to the provider family so a custom model name still gets a
	// sensible budget from the configured provider.
	switch strings.ToLower(provider) {
	case "gemini", "google":
		return windowGemini
	case "openai":
		return windowOpenAI
	case "anthropic":
		return windowAnthropic
	}
	return windowDefault
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// fmtTokens renders a token count compactly: >=1M as "N.NM", >=1000 as "N.NK", and a bare integer
// below 1000 (e.g. 12300 -> "12.3K", 1_200_000 -> "1.2M", 42 -> "42"). Kept in step with
// fmtDuration's terse footer register.
func fmtTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	default:
		return strconv.Itoa(n)
	}
}
