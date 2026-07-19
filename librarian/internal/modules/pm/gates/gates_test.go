package gates

import (
	"context"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/core/schema"
)

func TestDefaultRulesYAML_Parses(t *testing.T) {
	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("the shipped default gate rules must parse: %v", err)
	}
	if len(cfg.Gates["decision"]["review->terminal"].Documents) != 1 {
		t.Error("default rules should gate decision review->terminal on one document")
	}
	if len(cfg.Gates["task"]["work->review"].Documents) != 1 {
		t.Error("default rules should gate task work->review on one document")
	}
	if len(cfg.Traits) != 0 {
		t.Error("the shipped default deliberately seeds no traits")
	}
}

// TestParseRules_SpecExample proves the spec §4.2 example config (per-type rules + a trait)
// parses as written.
func TestParseRules_SpecExample(t *testing.T) {
	example := `
schema_version: 1
gates:
  decision:
    "review->terminal":
      documents:
        - type: decision
          status: accepted
          pointer: item
  task:
    "work->review":
      documents:
        - type: task
          status: active
traits:
  - name: governs-desk-operations
    match: { field: governs, equals: desk-operations }
    on: "review->terminal"
    documents:
      - type: decision
        status: accepted
`
	if _, err := ParseRules(example); err != nil {
		t.Fatalf("the spec's own §4.2 example must parse: %v", err)
	}
}

// TestParseRules_RefusesUnknownVocabulary is §4.3: a gate can never reference a non-existent
// document type or an illegal status, and malformed configs are rejected loud (R3.3/R7).
func TestParseRules_Refuses(t *testing.T) {
	cases := map[string]string{
		"unknown item type": `
schema_version: 1
gates:
  not-a-type:
    "review->terminal":
      documents: [{ type: decision, status: accepted }]`,
		"unknown document type": `
schema_version: 1
gates:
  decision:
    "review->terminal":
      documents: [{ type: not-a-type }]`,
		"illegal status for family type": `
schema_version: 1
gates:
  decision:
    "review->terminal":
      documents: [{ type: decision, status: shipped }]`,
		"illegal edge": `
schema_version: 1
gates:
  decision:
    "queue->terminal":
      documents: [{ type: decision, status: accepted }]`,
		"bad pointer spec": `
schema_version: 1
gates:
  decision:
    "review->terminal":
      documents: [{ type: decision, status: accepted, pointer: elsewhere }]`,
		"wrong schema_version": `
schema_version: 2
gates: {}`,
		"empty documents": `
schema_version: 1
gates:
  decision:
    "review->terminal":
      documents: []`,
		"trait with illegal edge": `
schema_version: 1
traits:
  - name: t
    match: { field: governs, equals: x }
    on: "queue->terminal"
    documents: [{ type: decision }]`,
		"not yaml": `: [`,
	}
	for name, yamlStr := range cases {
		if _, err := ParseRules(yamlStr); err == nil {
			t.Errorf("%s: ParseRules should refuse", name)
		}
	}
}

func TestEffective_TraitComposition(t *testing.T) {
	cfg, err := ParseRules(`
schema_version: 1
gates:
  decision:
    "review->terminal":
      documents: [{ type: decision, status: accepted }]
traits:
  - name: governs-desk-operations
    match: { field: governs, equals: desk-operations }
    on: "review->terminal"
    documents: [{ type: analysis, status: approved }]
`)
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}

	matching := func(field string) (string, bool) {
		if field == "governs" {
			return "desk-operations", true
		}
		return "", false
	}
	reqs := cfg.Effective("decision", "review->terminal", matching)
	if len(reqs) != 2 {
		t.Errorf("per-type rule ∪ matching trait should yield 2 requirements, got %d", len(reqs))
	}

	nonMatching := func(string) (string, bool) { return "", false }
	reqs = cfg.Effective("decision", "review->terminal", nonMatching)
	if len(reqs) != 1 {
		t.Errorf("non-matching trait must not add requirements, got %d", len(reqs))
	}

	// A type with no rule and no matching trait carries no gate at all.
	if reqs := cfg.Effective("analysis", "work->review", matching); len(reqs) != 0 {
		t.Errorf("ungated (type, edge) should be requirement-free, got %d", len(reqs))
	}
}

// stubValidator drives Evaluate without any librarian internals (test lane §10.5).
type stubValidator struct {
	verdicts map[string]schema.Verdict
}

func (s *stubValidator) Verdict(_ context.Context, pointer string, _ schema.ArtifactRequirement) (schema.Verdict, error) {
	return s.verdicts[pointer], nil
}

func TestEvaluate_StubValidator(t *testing.T) {
	reqs := []DocRequirement{{Type: "decision", Status: "accepted", Pointer: "item"}}
	resolve := func(string) (string, error) { return "_structure/decisions/0021-x.md", nil }

	// Unsatisfied: the refusal names exactly what is missing (R3.1).
	sv := &stubValidator{verdicts: map[string]schema.Verdict{
		"_structure/decisions/0021-x.md": {
			Exists: true, FrontmatterValid: true, Status: "proposed", Satisfied: false,
			Missing: []string{`required document (type=decision, status=accepted) at "_structure/decisions/0021-x.md" is at status "proposed", needs "accepted"`},
		},
	}}
	err := Evaluate(context.Background(), sv, reqs, resolve)
	var refusal *Refusal
	if r, ok := err.(*Refusal); ok {
		refusal = r
	} else {
		t.Fatalf("expected *Refusal, got %v", err)
	}
	if len(refusal.Reasons) != 1 || !strings.Contains(refusal.Reasons[0], `needs "accepted"`) {
		t.Errorf("refusal must carry the verdict's missing reasons verbatim: %v", refusal.Reasons)
	}

	// Satisfied: the same gate passes.
	sv.verdicts["_structure/decisions/0021-x.md"] = schema.Verdict{
		Exists: true, FrontmatterValid: true, Status: "accepted", Satisfied: true,
	}
	if err := Evaluate(context.Background(), sv, reqs, resolve); err != nil {
		t.Errorf("satisfied gate should pass: %v", err)
	}

	// No validator: fail closed (§2.5).
	err = Evaluate(context.Background(), nil, reqs, resolve)
	if r, ok := err.(*Refusal); !ok || !strings.Contains(r.Error(), "no document validator") {
		t.Errorf("nil validator must fail closed naming the absence, got %v", err)
	}

	// No requirements: trivially passes even with no validator.
	if err := Evaluate(context.Background(), nil, nil, resolve); err != nil {
		t.Errorf("ungated edge should pass trivially: %v", err)
	}
}
