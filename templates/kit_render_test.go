package templates

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRenderPortedKitTemplate proves a ported SOP kit template (kits/, the SOP-kit port) renders
// end-to-end through the existing template surface — the same Render() the librarian's fixer uses.
// It reads the kit off disk (go:embed can't reach across `..`), so it also guards that the ported
// template stays render-compatible: leading annotation stripped (none here), {{key}} substituted.
func TestRenderPortedKitTemplate(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file path")
	}
	// templates/ -> repo root -> kits/analysis/template.md
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..")
	kitTemplate := filepath.Join(repoRoot, "kits", "analysis", "template.md")

	raw, err := os.ReadFile(kitTemplate)
	if err != nil {
		t.Fatalf("read ported kit template: %v", err)
	}

	rendered := Render(string(raw), map[string]string{"date": "2026-07-18"})

	if strings.Contains(rendered, "{{date}}") {
		t.Errorf("Render did not substitute {{date}}:\n%s", rendered)
	}
	if !strings.Contains(rendered, "2026-07-18") {
		t.Errorf("expected substituted date in output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "type: analysis") {
		t.Errorf("expected frontmatter preserved (type: analysis) in output:\n%s", rendered)
	}
}
