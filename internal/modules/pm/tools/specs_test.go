package tools

import (
	"testing"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/core/toolcore"
)

func noValidator() schema.DocumentValidator { return nil }

// TestSpecs_FrozenNames pins the twelve tool ids (spec §5.1) in order — docs and the D5
// plugin build on these names; a rename is a breaking change, not a refactor.
func TestSpecs_FrozenNames(t *testing.T) {
	want := []string{
		"get_context", "list_items", "get_item", "create_item", "update_item",
		"transition_item", "block_item", "unblock_item", "add_note", "link_items",
		"claim_item", "release_item",
	}
	specs := Specs(noValidator, true)
	if len(specs) != len(want) {
		t.Fatalf("got %d specs, want %d", len(specs), len(want))
	}
	for i, s := range specs {
		if s.Name != want[i] {
			t.Errorf("spec[%d] = %q, want %q (frozen)", i, s.Name, want[i])
		}
		if s.Module != "pm" {
			t.Errorf("spec %q Module = %q, want pm", s.Name, s.Module)
		}
	}
	names := ToolNames()
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("ToolNames()[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

// TestSpecs_NoDeskFileWrites: PM tools write only the STORE (spec §5.1) — WritesFiles must
// be false on all twelve (the librarian's §5.4 desk-file gate does not apply to them).
func TestSpecs_NoDeskFileWrites(t *testing.T) {
	for _, s := range Specs(noValidator, true) {
		if s.WritesFiles {
			t.Errorf("%q has WritesFiles=true — PM tools never write desk files (§5.1)", s.Name)
		}
	}
}

// TestSpecs_WriteGateComposition pins the PM_AUTONOMOUS_WRITES gate (spec §13 item 9):
// writes ON (the default) exposes all twelve to the model-facing surfaces; writes OFF
// exposes only the three read tools, making agents read-only over the graph.
func TestSpecs_WriteGateComposition(t *testing.T) {
	reads := map[string]bool{"get_context": true, "list_items": true, "get_item": true}
	for _, writes := range []bool{true, false} {
		toolcore.Reset()
		toolcore.Register(Specs(noValidator, writes)...)
		exposed := map[string]bool{}
		for _, n := range toolcore.ExposedTools(&config.Config{}) {
			exposed[n] = true
		}
		if writes {
			if len(exposed) != 12 {
				t.Errorf("writes on: exposed %d tools, want 12 (%v)", len(exposed), exposed)
			}
		} else {
			if len(exposed) != 3 {
				t.Errorf("writes off: exposed %d tools, want the 3 reads (%v)", len(exposed), exposed)
			}
			for n := range exposed {
				if !reads[n] {
					t.Errorf("writes off: %q exposed but is not a read tool", n)
				}
			}
		}
	}
	toolcore.Reset()
}

// TestSpecs_SchemasBuild proves every frozen input struct survives the shared schema builder
// (the MCP registration path): each InputType yields a valid object schema.
func TestSpecs_SchemasBuild(t *testing.T) {
	for _, s := range Specs(noValidator, true) {
		schemaMap := toolcore.SchemaForType(s.InputType)
		if schemaMap["type"] != "object" {
			t.Errorf("%q schema is not an object: %v", s.Name, schemaMap)
		}
		if _, err := toolcore.BuildInputSchema(s.InputType); err != nil {
			t.Errorf("%q BuildInputSchema: %v", s.Name, err)
		}
	}
}
