// Profile-schema validation (schema v1): parses the embedded copy of the repo's
// schema/profile.schema.yaml — a JSON Schema authored in YAML — and validates a
// `_knowledge/profile.{yaml,yml,json,md}` profile map against it. The embedded copy MUST stay
// byte-identical to the repo-root schema/profile.schema.yaml;
// TestProfileSchemaEmbeddedCopy_MatchesRepoRoot is the drift guard, the same arrangement the
// reference vocabulary uses.
//
// Contract-version gate (ADR 0009): the schema file carries an `x-contract-version` marker
// naming the version of the SHARED CONTRACT FILE itself. This build understands the versions in
// KnownProfileContractVersions; anything else is refused LOUD rather than silently misread.
// The marker is a schema-meta key, not a JSON Schema keyword, so it is stripped before the
// schema is compiled — otherwise the root object's `additionalProperties: false` would see it.
// It is deliberately distinct from a profile INSTANCE's `schema_version` (which const-pins an
// instance to v1) and from the store-side module_schema_versions migration mechanism.
package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed profile.schema.yaml
var profileSchemaYAML []byte

// KnownProfileContractVersions is the set of profile.schema.yaml `x-contract-version` values
// this build understands. A schema file declaring anything else is refused in
// compileProfileSchema.
var KnownProfileContractVersions = []int{1}

// profileSchemaBaseURI is the base the schema's relative `$id` resolves against. The contract
// file declares a repo-relative `$id`, which the resolver rejects on its own (a bare relative
// URI is not absolute), so a base is required to compile it at all. The schema carries no
// external `$ref`, so nothing is ever loaded from this base — it exists purely to make the
// declared `$id` well-formed.
const profileSchemaBaseURI = "file:///"

// ProfileValidation is a flat validation result: the verdict plus human-readable violation
// strings, each naming WHERE in the profile the violation is and WHAT is wrong.
type ProfileValidation struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

var (
	profileSchemaOnce sync.Once
	profileSchema     *jsonschema.Resolved
	profileSchemaErr  error
)

// ProfileSchema returns the process-wide compiled schema-v1 profile validator. The embedded
// YAML is a build artifact, so a parse/compile failure is a build defect: it is returned
// (never swallowed) and every caller fails loud on it. Compilation happens at most once.
func ProfileSchema() (*jsonschema.Resolved, error) {
	profileSchemaOnce.Do(func() {
		raw, err := parseProfileSchemaObject(profileSchemaYAML)
		if err != nil {
			profileSchemaErr = err
			return
		}
		profileSchema, profileSchemaErr = compileProfileSchema(raw)
	})
	return profileSchema, profileSchemaErr
}

// ValidateProfile validates a parsed profile against schema v1. A schema-level failure (a build
// defect) is returned as an error; a PROFILE-level failure is a non-error result with Valid
// false and the violations listed.
func ValidateProfile(profile map[string]any) (ProfileValidation, error) {
	rs, err := ProfileSchema()
	if err != nil {
		return ProfileValidation{}, err
	}
	if profile == nil {
		profile = map[string]any{}
	}
	if verr := rs.Validate(profile); verr != nil {
		return ProfileValidation{Valid: false, Errors: []string{formatProfileViolation(verr)}}, nil
	}
	return ProfileValidation{Valid: true, Errors: []string{}}, nil
}

// parseProfileSchemaObject parses the YAML-authored JSON Schema into a plain object map.
func parseProfileSchemaObject(b []byte) (map[string]any, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("schema: parse embedded profile.schema.yaml: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("schema: embedded profile.schema.yaml is not a JSON-Schema object")
	}
	return raw, nil
}

// compileProfileSchema applies the contract-version gate and compiles the remaining schema
// object. It never mutates its caller's map: the `x-contract-version` key is dropped from a
// shallow copy, which is what gets compiled.
func compileProfileSchema(raw map[string]any) (*jsonschema.Resolved, error) {
	version, present := raw["x-contract-version"]
	n, numeric := asContractNumber(version)
	if !present || !numeric || !knownContractVersion(n) {
		shown := "(absent)"
		if present {
			shown = fmt.Sprintf("%v", version)
		}
		return nil, fmt.Errorf(
			"schema: profile contract version %s is not recognized (known versions: %v)",
			shown, KnownProfileContractVersions)
	}

	forCompile := make(map[string]any, len(raw))
	for k, v := range raw {
		if k == "x-contract-version" {
			continue
		}
		forCompile[k] = v
	}
	b, err := json.Marshal(forCompile)
	if err != nil {
		return nil, fmt.Errorf("schema: encode profile schema: %w", err)
	}
	s := new(jsonschema.Schema)
	if err := json.Unmarshal(b, s); err != nil {
		return nil, fmt.Errorf("schema: decode profile schema: %w", err)
	}
	rs, err := s.Resolve(&jsonschema.ResolveOptions{BaseURI: profileSchemaBaseURI})
	if err != nil {
		return nil, fmt.Errorf("schema: resolve profile schema: %w", err)
	}
	return rs, nil
}

// asContractNumber accepts the numeric shapes a YAML/JSON decoder can produce for the marker.
// A string (or anything else) is NOT numeric — the gate rejects it.
func asContractNumber(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		if t != float64(int(t)) {
			return 0, false
		}
		return int(t), true
	default:
		return 0, false
	}
}

func knownContractVersion(n int) bool {
	for _, k := range KnownProfileContractVersions {
		if k == n {
			return true
		}
	}
	return false
}

// validatingPrefixRe matches one `validating <schema-location>: ` wrapper the validator nests
// around each level it descended into.
var validatingPrefixRe = regexp.MustCompile(`^validating ([^:]*): `)

// formatProfileViolation turns the validator's nested error into one human-readable violation
// naming WHERE and WHAT. The validator reports SCHEMA locations (`/properties/repos/properties/
// default`), so the deepest one is translated into the profile-instance path a profile author
// actually edits (`repos.default`); a violation at the document root is marked "(root)".
//
// Note the deliberate delta from the TypeScript predecessor, which used a validator with an
// all-errors mode: this validator stops at the FIRST violation, so a profile with several
// problems surfaces them one round at a time. Both the offending key and its location are still
// named, which is what makes the message actionable; closing the delta would mean a second
// JSON-Schema dependency, which is not worth it.
func formatProfileViolation(err error) string {
	msg := err.Error()
	loc := ""
	for {
		m := validatingPrefixRe.FindStringSubmatch(msg)
		if m == nil {
			break
		}
		if strings.HasPrefix(m[1], "/") {
			loc = m[1]
		}
		msg = msg[len(m[0]):]
	}
	where := instancePathFromSchemaPath(loc)
	if where == "" {
		where = "(root)"
	}
	return where + ": " + msg
}

// instancePathFromSchemaPath maps a JSON-Schema location pointer onto the profile path it
// constrains: `/properties/repos/properties/default` -> `repos.default`, and an array's
// `/properties/machines/items/properties/role` -> `machines[].role`.
func instancePathFromSchemaPath(schemaPath string) string {
	trimmed := strings.Trim(schemaPath, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	var out []string
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "properties":
			if i+1 < len(parts) {
				out = append(out, parts[i+1])
				i++
			}
		case "items":
			out = append(out, "[]")
		}
	}
	return strings.ReplaceAll(strings.Join(out, "."), ".[]", "[]")
}
