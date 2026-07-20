// References is the schema-v1 typed cross-reference guard (ADR 0011): it parses the embedded
// copy of the repo's schema/references.yaml into the reference vocabulary and validates a
// { kind, target } reference against it. The primitive is one closed `kind` enum plus a raw
// `target` string; the desk-relative repo qualifier is NOT part of the persisted shape — per
// ADR 0011 it resolves at read time from the profile (repos.shorthand.issue_default) and this
// guard therefore takes no qualifier parameter. The embedded copy MUST stay byte-identical to
// the repo-root schema/references.yaml — TestReferencesEmbeddedCopy_MatchesRepoRoot is the
// drift guard.
//
// Scope (deliberate): this validates the persisted shape only — a known kind and a non-empty
// target. It ships no resolver and no extraction change; today's graduation-marker extraction
// and pointer resolution are untouched, and no field migrates onto this shape here (the field
// migrations ride the schema-v2 track).
package schema

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed references.yaml
var referencesYAML []byte

// ReferenceVocabulary is the parsed schema-v1 reference contract: the closed set of legal
// reference kinds, in file order.
type ReferenceVocabulary struct {
	Kinds []string
}

// rawReferences mirrors the YAML shape of schema/references.yaml.
type rawReferences struct {
	Kind   []string       `yaml:"kind"`
	Target map[string]any `yaml:"target"`
}

var (
	refVocabOnce sync.Once
	refVocab     *ReferenceVocabulary
	refVocabErr  error
)

// ReferenceVocab returns the process-wide parsed reference vocabulary. The embedded YAML is a
// build artifact, so a parse failure is a build defect: it is returned (never swallowed) and
// every caller fails loud on it.
func ReferenceVocab() (*ReferenceVocabulary, error) {
	refVocabOnce.Do(func() { refVocab, refVocabErr = parseReferences(referencesYAML) })
	return refVocab, refVocabErr
}

func parseReferences(b []byte) (*ReferenceVocabulary, error) {
	var raw rawReferences
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("schema: parse embedded references.yaml: %w", err)
	}
	if len(raw.Kind) == 0 {
		return nil, fmt.Errorf("schema: embedded references.yaml missing kind enum")
	}
	kinds := make([]string, 0, len(raw.Kind))
	for _, k := range raw.Kind {
		if k = strings.TrimSpace(k); k != "" {
			kinds = append(kinds, k)
		}
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("schema: embedded references.yaml kind enum is empty after trimming")
	}
	return &ReferenceVocabulary{Kinds: kinds}, nil
}

// KnownKind reports whether kind is one of the canonical schema-v1 reference kinds.
func (v *ReferenceVocabulary) KnownKind(kind string) bool {
	for _, k := range v.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// ValidateReference checks a persisted { kind, target } reference against the schema-v1
// contract: the kind must be known and the target must be non-empty (whitespace-only is
// empty). It takes NO qualifier — the persisted shape never carries one (ADR 0011); the
// desk-relative repo qualifier is a read-time concern resolved from the profile.
func ValidateReference(kind, target string) error {
	v, err := ReferenceVocab()
	if err != nil {
		return err
	}
	if !v.KnownKind(kind) {
		return fmt.Errorf("schema: unknown reference kind %q (known: %v)", kind, v.Kinds)
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("schema: reference target must be non-empty (kind: %q)", kind)
	}
	return nil
}
