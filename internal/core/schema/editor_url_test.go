package schema

import "testing"

// minimalProfile is the smallest profile the schema accepts, so the assertions below isolate the
// `preferences` block instead of tripping on an unrelated required key.
func minimalProfile(prefs map[string]any) map[string]any {
	p := map[string]any{
		"schema_version": 1,
		"desk": map[string]any{
			"name": "example-desk",
			"root": ".",
		},
	}
	if prefs != nil {
		p["preferences"] = prefs
	}
	return p
}

// TestProfileSchema_EditorURLAccepted: `preferences` is additionalProperties:false, so the
// editor hand-off key is unusable until the contract declares it — an undeclared key would make
// every desk that sets one fail validation, which is a worse failure than not having the feature.
func TestProfileSchema_EditorURLAccepted(t *testing.T) {
	res, err := ValidateProfile(minimalProfile(map[string]any{
		"editor_url": "x-editor://open?path={abs}",
	}))
	if err != nil {
		t.Fatalf("ValidateProfile: %v", err)
	}
	if !res.Valid {
		t.Fatalf("a profile declaring preferences.editor_url is rejected: %v", res.Errors)
	}
}

// TestProfileSchema_EditorURLIsOptional: a desk that declares no editor_url must still validate
// — the browser simply renders no hand-off button. Optional means optional.
func TestProfileSchema_EditorURLIsOptional(t *testing.T) {
	for name, prefs := range map[string]map[string]any{
		"no preferences block": nil,
		"preferences without editor_url": {
			"commit_style": "conventional",
		},
	} {
		res, err := ValidateProfile(minimalProfile(prefs))
		if err != nil {
			t.Fatalf("%s: ValidateProfile: %v", name, err)
		}
		if !res.Valid {
			t.Errorf("%s: should validate, got %v", name, res.Errors)
		}
	}
}

// TestProfileSchema_EditorURLIsAString: the value is a URL template the browser puts in an
// anchor's href. A non-string (a map, a list) would reach the surface as a rendered Go map
// literal, so the contract types it rather than accepting anything.
func TestProfileSchema_EditorURLIsAString(t *testing.T) {
	res, err := ValidateProfile(minimalProfile(map[string]any{
		"editor_url": map[string]any{"command": "open"},
	}))
	if err != nil {
		t.Fatalf("ValidateProfile: %v", err)
	}
	if res.Valid {
		t.Fatal("preferences.editor_url accepted a non-string value")
	}
}
