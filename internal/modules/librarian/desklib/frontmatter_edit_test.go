package desklib

import (
	"strings"
	"testing"
)

const fmDoc = `---
type: guide
status: draft
synopsis: "a: colon-tolerant value"
tags: [a, b]
---

# Body

body text stays byte-identical, even --- this line.
`

func TestSetFrontmatterField_ReplacesScalarOnly(t *testing.T) {
	out, err := SetFrontmatterField([]byte(fmDoc), "status", "active")
	if err != nil {
		t.Fatalf("SetFrontmatterField: %v", err)
	}
	want := strings.Replace(fmDoc, "status: draft", "status: active", 1)
	if string(out) != want {
		t.Errorf("byte-exactness violated:\n got: %q\nwant: %q", out, want)
	}
}

func TestSetFrontmatterField_InsertsMissingKeyBeforeFence(t *testing.T) {
	out, err := SetFrontmatterField([]byte(fmDoc), "updated", "2026-08-20")
	if err != nil {
		t.Fatalf("SetFrontmatterField: %v", err)
	}
	if !strings.Contains(string(out), "tags: [a, b]\nupdated: 2026-08-20\n---") {
		t.Errorf("missing key not inserted before the closing fence:\n%s", out)
	}
	fm := ParseFrontmatter(string(out))
	if fm["updated"] != "2026-08-20" {
		t.Errorf("inserted key does not round-trip through ParseFrontmatter: %v", fm["updated"])
	}
}

func TestSetFrontmatterField_Refusals(t *testing.T) {
	blockDoc := "---\nrefs:\n- a\n- b\n---\nbody\n"
	cases := []struct {
		name    string
		doc     string
		key     string
		value   string
		wantErr string
	}{
		{"no frontmatter", "# plain doc\n", "status", "x", "no frontmatter"},
		{"unterminated fence", "---\nstatus: draft\n", "status", "x", "never closes"},
		{"block array key", blockDoc, "refs", "x", "block array"},
		{"block scalar pipe", "---\nnotes: |\n  line one\n  line two\n---\nbody\n", "notes", "x", "block scalar"},
		{"block scalar folded chomped", "---\nnotes: >-\n  folded\n---\nbody\n", "notes", "x", "block scalar"},
		{"multiline value", fmDoc, "status", "a\nb", "single line"},
		{"empty key", fmDoc, " ", "x", "empty key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := SetFrontmatterField([]byte(tc.doc), tc.key, tc.value); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestSetFrontmatterField_PreservesCRLF(t *testing.T) {
	doc := "---\r\nstatus: draft\r\n---\r\nbody\r\n"
	out, err := SetFrontmatterField([]byte(doc), "status", "active")
	if err != nil {
		t.Fatalf("SetFrontmatterField: %v", err)
	}
	if string(out) != "---\r\nstatus: active\r\n---\r\nbody\r\n" {
		t.Errorf("CRLF not preserved: %q", out)
	}
}
