package agent

import (
	"context"
	"testing"

	"github.com/hsb3/deskkit/internal/core/config"
	coreschema "github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/core/toolcore"
	"github.com/hsb3/deskkit/internal/modules/librarian/tools"
	pmtools "github.com/hsb3/deskkit/internal/modules/pm/tools"
)

// TestBuildTools_LibrarianOnly proves the in-binary eino agent's tool slice is librarian-only
// (ADR 0014(c)) even when the merged registry ALSO carries the PM module's twelve tools (the
// PM_ENABLED case). This is the RED/GREEN pivot for this slice: before buildTools filtered on
// module, it iterated toolcore.AgentTools(cfg) directly, so this test would fail with the PM
// tools' names present in the built slice; after the toolcore.SelectByModules(..., "librarian")
// filter was added, the eino loop never receives a tool its system prompt does not cover.
func TestBuildTools_LibrarianOnly(t *testing.T) {
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	// A no-op validator getter: buildTools only CONSTRUCTS eino tools (reflects the input
	// struct into a schema), it never INVOKES a tool handler, so the validator closure is
	// never called here. writesEnabled=true so all twelve PM specs get AgentDefault=true and
	// register — this exercises the worst case (every PM tool present in the merged registry).
	toolcore.Register(pmtools.Specs(func() coreschema.DocumentValidator { return nil }, true)...)
	// Restore the registry so a future test in this package doesn't inherit the PM specs
	// (same cleanup discipline as the mcp package's module-gate tests).
	t.Cleanup(func() {
		toolcore.Reset()
		toolcore.Register(tools.Specs()...)
	})

	cfg := &config.Config{AutonomousWrites: true}

	// buildTools only builds tool.BaseTool wrappers (schema reflection); it never invokes a
	// tool handler, so it is safe to call with a nil core.App here.
	built, err := buildTools(nil, cfg)
	if err != nil {
		t.Fatalf("buildTools: %v", err)
	}

	gotNames := map[string]bool{}
	for _, bt := range built {
		info, err := bt.Info(context.Background())
		if err != nil {
			t.Fatalf("tool Info: %v", err)
		}
		gotNames[info.Name] = true
	}

	// Every librarian tool the §5.4 gate admits must be present.
	wantLibrarian := toolcore.ToolNames(toolcore.SelectByModules(toolcore.AgentTools(cfg), "librarian"))
	if len(wantLibrarian) == 0 {
		t.Fatalf("expected at least one librarian tool in the gated set (test setup broken)")
	}
	for _, name := range wantLibrarian {
		if !gotNames[name] {
			t.Errorf("expected librarian tool %q in the eino slice, missing", name)
		}
	}
	if len(gotNames) != len(wantLibrarian) {
		t.Errorf("eino slice has %d tools, want exactly the %d librarian tools: got %v", len(gotNames), len(wantLibrarian), gotNames)
	}

	// Zero PM tools may leak into the eino slice, no matter how they are named.
	for _, pmName := range pmtools.ToolNames() {
		if gotNames[pmName] {
			t.Errorf("PM tool %q leaked into the eino agent's tool slice (must be librarian-only, ADR 0014(c))", pmName)
		}
	}
}
