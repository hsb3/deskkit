// Package gates is the gate engine + the editable-YAML gate-config loader (spec §4). Gate
// rules live per desk in desk_config.rules, keyed (item type → transition → required
// documents), plus cross-cutting traits; ParseRules validates a config against the machine's
// legal edges and the schema-v1 vocabulary (§4.3) and fails LOUD on anything unknown, so a
// gate can never reference a non-existent document type. Evaluate answers one transition's
// gate check through the narrow core/schema seam (§2.5) — it never reads librarian
// collections, and with no DocumentValidator registered it fails closed (§2.5).
package gates

import (
	"context"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/example/pocket-librarian/internal/core/schema"
	"github.com/example/pocket-librarian/internal/modules/pm/statemachine"
)

// DocRequirement is one required document in a gate rule (§4.2).
type DocRequirement struct {
	Type    string `yaml:"type"`              // schema-v1 / kit type (R3.4)
	Status  string `yaml:"status,omitempty"`  // required frontmatter status; empty = existence + validity only
	Pointer string `yaml:"pointer,omitempty"` // "item" (default; resolve via item.pointer) | "note:<key>"
}

// TraitMatch is a trait's predicate: an item field (or the pointed doc's frontmatter field)
// equalling a value.
type TraitMatch struct {
	Field  string `yaml:"field"`
	Equals string `yaml:"equals"`
}

// Trait composes a cross-cutting requirement onto any item matching the predicate (§4.2).
type Trait struct {
	Name      string           `yaml:"name"`
	Match     TraitMatch       `yaml:"match"`
	On        string           `yaml:"on"` // transition key, e.g. "review->terminal"
	Documents []DocRequirement `yaml:"documents"`
}

// Config is a desk's parsed + validated gate configuration.
type Config struct {
	SchemaVersion int                                   `yaml:"schema_version"`
	Gates         map[string]map[string]docRequirements `yaml:"gates"` // type -> transition key -> requirements
	Traits        []Trait                               `yaml:"traits"`
}

// docRequirements wraps the `documents:` list under a transition key.
type docRequirements struct {
	Documents []DocRequirement `yaml:"documents"`
}

// ParseRules parses + validates a desk's gate-rules YAML (§4.2/§4.3). Every failure is a
// loud, named error — an invalid config is REJECTED, never silently disabling gates.
func ParseRules(rulesYAML string) (*Config, error) {
	vocab, err := schema.Vocab()
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte(rulesYAML), &cfg); err != nil {
		return nil, fmt.Errorf("gate config: invalid YAML: %w", err)
	}
	if cfg.SchemaVersion != 1 {
		return nil, fmt.Errorf("gate config: schema_version must be 1, got %d", cfg.SchemaVersion)
	}
	for itemType, transitions := range cfg.Gates {
		if !vocab.KnownType(itemType) {
			return nil, fmt.Errorf("gate config: gates key %q is not a known schema-v1/kit type", itemType)
		}
		for key, reqs := range transitions {
			if _, _, err := statemachine.ParseEdgeKey(key); err != nil {
				return nil, fmt.Errorf("gate config: gates.%s: %w", itemType, err)
			}
			if len(reqs.Documents) == 0 {
				return nil, fmt.Errorf("gate config: gates.%s.%q binds no documents", itemType, key)
			}
			for _, d := range reqs.Documents {
				if err := validateDocRequirement(vocab, d); err != nil {
					return nil, fmt.Errorf("gate config: gates.%s.%q: %w", itemType, key, err)
				}
			}
		}
	}
	for _, tr := range cfg.Traits {
		if tr.Name == "" {
			return nil, fmt.Errorf("gate config: every trait needs a name")
		}
		if tr.Match.Field == "" {
			return nil, fmt.Errorf("gate config: trait %q has no match.field", tr.Name)
		}
		if _, _, err := statemachine.ParseEdgeKey(tr.On); err != nil {
			return nil, fmt.Errorf("gate config: trait %q: %w", tr.Name, err)
		}
		if len(tr.Documents) == 0 {
			return nil, fmt.Errorf("gate config: trait %q binds no documents", tr.Name)
		}
		for _, d := range tr.Documents {
			if err := validateDocRequirement(vocab, d); err != nil {
				return nil, fmt.Errorf("gate config: trait %q: %w", tr.Name, err)
			}
		}
	}
	return &cfg, nil
}

func validateDocRequirement(vocab *schema.Vocabulary, d DocRequirement) error {
	if !vocab.KnownType(d.Type) {
		return fmt.Errorf("document type %q is not a known schema-v1/kit type", d.Type)
	}
	if d.Status != "" && !vocab.StatusAllowed(d.Type, d.Status) {
		return fmt.Errorf("status %q is not a legal status for document type %q", d.Status, d.Type)
	}
	switch {
	case d.Pointer == "" || d.Pointer == "item":
	case len(d.Pointer) > 5 && d.Pointer[:5] == "note:":
	default:
		return fmt.Errorf("document pointer %q must be \"item\" or \"note:<key>\"", d.Pointer)
	}
	return nil
}

// FieldLookup resolves a trait predicate field for an item: first-class item fields (and the
// properties overflow) first, then the pointed document's frontmatter through the seam's
// optional FrontmatterReader. Supplied by the engine so gates stays store-free.
type FieldLookup func(field string) (value string, ok bool)

// Effective returns the requirements the config binds to (itemType, edge): the per-type rule
// ∪ every matching trait (§4.2). Only edges with bound rules gate — forward or not, demote/
// reopen carry no gate unless the config names them (§3.2).
func (c *Config) Effective(itemType string, edgeKey string, lookup FieldLookup) []DocRequirement {
	var reqs []DocRequirement
	if perType, ok := c.Gates[itemType]; ok {
		reqs = append(reqs, perType[edgeKey].Documents...)
	}
	for _, tr := range c.Traits {
		if tr.On != edgeKey {
			continue
		}
		if lookup == nil {
			continue
		}
		if v, ok := lookup(tr.Match.Field); ok && v == tr.Match.Equals {
			reqs = append(reqs, tr.Documents...)
		}
	}
	return reqs
}

// Refusal is a gate/machine refusal: the exact, human-readable list of what is missing
// (R3.1). It is an error so the engine's callers can distinguish "refused" from "broken".
type Refusal struct {
	Reasons []string
}

func (r *Refusal) Error() string {
	if len(r.Reasons) == 1 {
		return r.Reasons[0]
	}
	out := "refused:"
	for _, reason := range r.Reasons {
		out += "\n  - " + reason
	}
	return out
}

// Evaluate runs one transition's gate check (§4.1 step 4): resolve each effective
// requirement's document pointer, obtain a verdict through the seam, and return a *Refusal
// naming EXACTLY what is missing if any verdict is unsatisfied. resolvePointer maps a
// DocRequirement's pointer spec ("item" / "note:<key>") to the actual doc path; validator nil
// = fail closed, naming the absence (§2.5).
func Evaluate(
	ctx context.Context,
	validator schema.DocumentValidator,
	reqs []DocRequirement,
	resolvePointer func(spec string) (string, error),
) error {
	if len(reqs) == 0 {
		return nil // an edge with no bound gate passes trivially (§4.1)
	}
	if validator == nil {
		return &Refusal{Reasons: []string{"no document validator available; documented gates fail closed"}}
	}
	var missing []string
	for _, req := range reqs {
		spec := req.Pointer
		if spec == "" {
			spec = "item"
		}
		pointer, err := resolvePointer(spec)
		if err != nil {
			missing = append(missing, fmt.Sprintf(
				"required document (type=%s, status=%s): %v", req.Type, req.Status, err))
			continue
		}
		verdict, err := validator.Verdict(ctx, pointer, schema.ArtifactRequirement{
			Type:           req.Type,
			RequiredStatus: req.Status,
		})
		if err != nil {
			return fmt.Errorf("gate: document verdict for %q: %w", pointer, err)
		}
		if !verdict.Satisfied {
			missing = append(missing, verdict.Missing...)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return &Refusal{Reasons: missing}
	}
	return nil
}
