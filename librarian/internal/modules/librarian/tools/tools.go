package tools

import (
	"errors"
	"fmt"
)

// ErrNotImplemented is the sentinel every spine stub returns. The tool-body fan-out slices
// replace their file's body (one owner per tool file: sweep.go, patrol.go, …) with the real
// §5 logic; the signatures in types.go and the registry in registry.go are frozen contracts.
// Callers can errors.Is(err, ErrNotImplemented) to distinguish "unbuilt" from a real failure.
//
// The seven core functions live one-per-file (spec §2.8): sweep.go, patrol.go, propose_fix.go,
// apply_fix.go, restore.go, query.go, record_feedback.go. Each is the SINGLE implementation both the CLI
// subcommands and the (later) eino InvokableTool wrappers call — "sweep behaves identically
// whether the agent calls it or an operator runs pocket-librarian sweep" (spec §2.6).
var ErrNotImplemented = errors.New("pocket-librarian: tool not implemented (spine stub)")

func notImplemented(name string) error {
	return fmt.Errorf("%s: %w", name, ErrNotImplemented)
}
