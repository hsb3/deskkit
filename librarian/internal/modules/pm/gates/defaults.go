package gates

// DefaultRulesYAML is the shipped default gate ruleset a desk starts from when its
// desk_config carries no rules row yet (spec §4.2, §3.8).
//
// KNOWN UNAUTHORED DESIGN GAP — flagged for the owner (spec build-order + D3 brief): the
// spec establishes the SCHEMA and two examples but no ruled default. This seed ships exactly
// the two per-type rules the spec's §4.2 examples establish, and nothing beyond them (no
// traits — the example trait predicates on a desk-specific `governs` frontmatter field, which
// is not a defensible universal default). The owner may re-rule this wholesale; a desk edits
// it via desk_config.rules either way. Identity-neutral by construction (R5.3).
const DefaultRulesYAML = `schema_version: 1
gates:
  # A decision item cannot complete until its decision document is accepted (§4.2 example 1).
  decision:
    "review->terminal":
      documents:
        - type: decision
          status: accepted
          pointer: item
  # A task item cannot enter review until its task document exists, validates, and is active
  # (§4.2 example 2).
  task:
    "work->review":
      documents:
        - type: task
          status: active
          pointer: item
`

// DefaultConfig returns the parsed shipped default. The default is compile-time constant and
// covered by TestDefaultRulesYAML_Parses, so a parse failure here is a build defect.
func DefaultConfig() (*Config, error) { return ParseRules(DefaultRulesYAML) }
