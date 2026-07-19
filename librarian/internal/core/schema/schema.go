// Package schema defines the narrow document-verdict seam (spec §2.5) the PM module (D3)
// consumes to ask "does this document pointer satisfy a gate's artifact requirement?" WITHOUT
// reading the librarian's collections directly. The librarian module implements
// DocumentValidator (internal/modules/librarian/module.go); core captures the implementation at
// module registration and injects it into consumers. D2 defines the seam only — no gate
// consumes it yet.
package schema

import "context"

// ArtifactRequirement is what a gate demands of a document (§2.5).
type ArtifactRequirement struct {
	Type           string // schema-v1 / kit type, e.g. "decision"
	RequiredStatus string // e.g. "accepted"; empty = existence + frontmatter validity only
}

// Verdict is the seam's answer about one document pointer.
type Verdict struct {
	Exists           bool
	FrontmatterValid bool
	Status           string   // the doc's actual status (for the refusal message)
	Satisfied        bool     // Exists && FrontmatterValid && (RequiredStatus=="" || Status==RequiredStatus)
	Missing          []string // human-readable reasons, verbatim into the gate refusal
}

// DocumentValidator is the narrow interface the PM module (D3) will consume to obtain document
// verdicts WITHOUT reading the librarian's collections. The librarian module implements it. Core
// injects the impl into consumers at registration; if none is registered the gate engine (D3)
// fails closed.
type DocumentValidator interface {
	Verdict(ctx context.Context, pointer string, req ArtifactRequirement) (Verdict, error)
}

// FrontmatterReader is the optional companion seam for trait predicates (spec §4.2): a trait's
// `match` may reference a frontmatter field of the item's POINTED document, resolved "through
// the validation seam, never by reading librarian collections". Verdict alone cannot answer an
// arbitrary-field lookup, so the librarian module also implements this; the gate engine
// type-asserts for it and treats its absence as "field not present" (the trait simply does not
// match — fail-safe, not fail-closed, because a trait is an ADDITIVE requirement).
type FrontmatterReader interface {
	Frontmatter(ctx context.Context, pointer string) (map[string]any, error)
}
