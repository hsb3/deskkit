// Doctypes is the schema-v1 vocabulary engine (spec §4.3, §4.4): it parses the embedded copy
// of the repo's schema/doctypes.yaml (the D1 kit port's contract-as-data) into the type/status
// vocabulary the gate engine validates rules against and the DocumentValidator validates
// frontmatter against. The embedded copy MUST stay byte-identical to the repo-root
// schema/doctypes.yaml — TestDoctypesEmbeddedCopy_MatchesRepoRoot is the drift guard.
//
// Scope (deliberate, flagged in the D3 PR): v1 validation covers the universal keys, known
// type, the type's status family, and the type's required fields. The `formats:`/`enums:`
// value-format checks are NOT enforced by this engine yet — the librarian's own patrol rules
// don't enforce them either, and a stricter verdict than the existing validation engine would
// invent a second, divergent notion of validity (§4.4 forbids exactly that).
package schema

import (
	_ "embed"
	"fmt"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed doctypes.yaml
var doctypesYAML []byte

// TypeSpec is one `types:` entry from doctypes.yaml.
type TypeSpec struct {
	StatusFamily string   // "" for lightweight types (no status family)
	Lightweight  bool     // universal fields only; status optional
	Required     []string // type-specific required frontmatter fields
	Optional     []string
}

// Vocabulary is the parsed schema-v1 doc-type vocabulary.
type Vocabulary struct {
	Universal      []string            // required on every doc (status optional on lightweight)
	StatusFamilies map[string][]string // family name -> allowed status values
	Types          map[string]TypeSpec
}

// rawDoctypes mirrors the YAML shape of schema/doctypes.yaml.
type rawDoctypes struct {
	Universal []string                  `yaml:"universal"`
	Status    map[string][]string       `yaml:"status"`
	Types     map[string]map[string]any `yaml:"types"`
}

var (
	vocabOnce sync.Once
	vocab     *Vocabulary
	vocabErr  error
)

// Vocab returns the process-wide parsed vocabulary. The embedded YAML is a build artifact, so
// a parse failure is a build defect: it is returned (never swallowed) and every caller fails
// loud on it.
func Vocab() (*Vocabulary, error) {
	vocabOnce.Do(func() { vocab, vocabErr = parseDoctypes(doctypesYAML) })
	return vocab, vocabErr
}

func parseDoctypes(b []byte) (*Vocabulary, error) {
	var raw rawDoctypes
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("schema: parse embedded doctypes.yaml: %w", err)
	}
	if len(raw.Universal) == 0 || len(raw.Types) == 0 {
		return nil, fmt.Errorf("schema: embedded doctypes.yaml missing universal/types sections")
	}
	v := &Vocabulary{
		Universal:      raw.Universal,
		StatusFamilies: raw.Status,
		Types:          make(map[string]TypeSpec, len(raw.Types)),
	}
	for name, spec := range raw.Types {
		ts := TypeSpec{}
		if lw, _ := spec["lightweight"].(bool); lw {
			ts.Lightweight = true
		}
		if fam, _ := spec["status"].(string); fam != "" {
			if _, ok := raw.Status[fam]; !ok {
				return nil, fmt.Errorf("schema: doctypes type %q names unknown status family %q", name, fam)
			}
			ts.StatusFamily = fam
		}
		ts.Required = stringList(spec["required"])
		ts.Optional = stringList(spec["optional"])
		v.Types[name] = ts
	}
	return v, nil
}

func stringList(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// KnownType reports whether typ is a canonical schema-v1 doc type.
func (v *Vocabulary) KnownType(typ string) bool {
	_, ok := v.Types[typ]
	return ok
}

// StatusAllowed reports whether status is a legal value for typ. A type without a status
// family (lightweight) accepts any status string — its statuses are unclassified by design
// (the spec's own §4.2 example gates a lightweight `task` doc on status `active`).
func (v *Vocabulary) StatusAllowed(typ, status string) bool {
	ts, ok := v.Types[typ]
	if !ok {
		return false
	}
	if ts.StatusFamily == "" {
		return true
	}
	for _, s := range v.StatusFamilies[ts.StatusFamily] {
		if s == status {
			return true
		}
	}
	return false
}

// TypeNames returns the sorted canonical type names (for error messages).
func (v *Vocabulary) TypeNames() []string {
	names := make([]string, 0, len(v.Types))
	for n := range v.Types {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ValidateFrontmatter checks a parsed frontmatter map against the schema-v1 contract for typ
// and returns human-readable reasons (empty = valid). Values are the string/[]string shapes
// desklib.ParseFrontmatter produces.
func (v *Vocabulary) ValidateFrontmatter(fm map[string]any, typ string) []string {
	var reasons []string
	ts, ok := v.Types[typ]
	if !ok {
		return []string{fmt.Sprintf("unknown document type %q", typ)}
	}
	for _, key := range v.Universal {
		if key == "status" && ts.Lightweight {
			continue // status is optional on lightweight types
		}
		if _, present := fm[key]; !present {
			reasons = append(reasons, fmt.Sprintf("missing required frontmatter key %q", key))
		}
	}
	if ts.StatusFamily != "" {
		if status, _ := fm["status"].(string); status != "" && !v.StatusAllowed(typ, status) {
			reasons = append(reasons, fmt.Sprintf(
				"status %q is not in the %q family %v", status, ts.StatusFamily, v.StatusFamilies[ts.StatusFamily]))
		}
	}
	for _, req := range ts.Required {
		if _, present := fm[req]; !present {
			reasons = append(reasons, fmt.Sprintf("missing required %q field %q", typ, req))
		}
	}
	return reasons
}
