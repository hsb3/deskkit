package profile

import (
	"strings"
	"testing"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/module"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/modules/profile/tools"
)

// TestModule_OwnsNoPersistedState pins the pure-read contract: no collections, no migrations,
// and a SchemaVersion consistent with having none. A future migration here must move all three
// together, which is what this guard forces.
func TestModule_OwnsNoPersistedState(t *testing.T) {
	m := New()
	if got := m.Name(); got != "profile" {
		t.Errorf("Name() = %q, want %q", got, "profile")
	}
	if !m.Enabled(nil) || !m.Enabled(&config.Config{}) {
		t.Error("the profile module must be enabled unconditionally, including on a nil config")
	}
	if len(m.OwnedCollections()) != 0 {
		t.Errorf("OwnedCollections() = %v, want none", m.OwnedCollections())
	}
	if len(m.Migrations()) != 0 {
		t.Errorf("Migrations() = %v, want none", m.Migrations())
	}
	if m.SchemaVersion() != 0 {
		t.Errorf("SchemaVersion() = %d, want 0 for a module with no migrations", m.SchemaVersion())
	}
	if m.TUIViews(nil, nil) != nil {
		t.Error("TUIViews() should be nil")
	}
	if err := m.RegisterHooks(nil, nil); err != nil {
		t.Errorf("RegisterHooks() = %v, want nil", err)
	}
}

// TestModuleGate_ProfileOnlyMount is the acceptance proof for a shared MCP mount narrowed with
// MCP_MODULES=profile: the SAME SelectByModules-over-ExposedSpecs filter the MCP server applies
// must resolve to EXACTLY this module's four tools and nothing else.
func TestModuleGate_ProfileOnlyMount(t *testing.T) {
	toolcore.Reset()
	t.Cleanup(toolcore.Reset)
	if _, err := module.Register(&config.Config{}, New()); err != nil {
		t.Fatalf("module.Register: %v", err)
	}

	// Both write-gate settings, since none of these tools is write-gated: the count must not move.
	for _, autonomous := range []bool{false, true} {
		cfg := &config.Config{AutonomousWrites: autonomous}
		got := toolcore.ToolNames(toolcore.SelectByModules(toolcore.ExposedSpecs(cfg), "profile"))
		want := tools.ToolNames()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("MCP_MODULES=profile (AutonomousWrites=%v) exposes %v, want exactly %v",
				autonomous, got, want)
		}
	}

	// Every registered spec is read-only and owned by this module.
	for _, s := range toolcore.AllTools() {
		if s.Module != "profile" {
			t.Errorf("tool %q carries module %q, want %q", s.Name, s.Module, "profile")
		}
		if s.WritesFiles || s.AgentGated || !s.AgentDefault {
			t.Errorf("tool %q gate flags = writes:%v gated:%v default:%v; want a plain read-only tool",
				s.Name, s.WritesFiles, s.AgentGated, s.AgentDefault)
		}
	}
}
