package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runCmdOut executes a standalone command with the given args against a buffer and returns its
// output. Mirrors how main() runs an intercepted command (runIntercepted), minus the os.Exit.
func runCmdOut(t *testing.T, cmd *cobra.Command, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%s %v: %v (output: %s)", cmd.Name(), args, err, buf.String())
	}
	return buf.String()
}

// seedDesks redirects both XDG homes to throwaway dirs (mandatory: no test may read or write the
// operator's real ones) and creates a store dir per name. It returns the store home.
func seedDesks(t *testing.T, names ...string) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	base := filepath.Join(data, "deskkit")
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(base, n), 0o700); err != nil {
			t.Fatalf("seed desk %s: %v", n, err)
		}
	}
	return base
}

// TestDesksCmd_ListsEachDeskWithPathAndMarksResolved is the definition-of-done case: two seeded
// stores, both listed with their paths, and the desk this invocation resolves to marked.
func TestDesksCmd_ListsEachDeskWithPathAndMarksResolved(t *testing.T) {
	base := seedDesks(t, "alpha", "beta")
	t.Setenv("DESK_ROOT", t.TempDir())
	t.Setenv("DESK_NAME", "beta")

	out := runCmdOut(t, newDesksCmd())

	for _, n := range []string{"alpha", "beta"} {
		if !strings.Contains(out, filepath.Join(base, n)) {
			t.Errorf("desks output omits %s's store path:\n%s", n, out)
		}
	}
	var alphaLine, betaLine string
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.Contains(line, filepath.Join(base, "alpha")):
			alphaLine = line
		case strings.Contains(line, filepath.Join(base, "beta")):
			betaLine = line
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(betaLine), "*") {
		t.Errorf("the resolved desk (beta) is not marked: %q", betaLine)
	}
	if strings.Contains(alphaLine, "*") {
		t.Errorf("a desk that is NOT resolved here is marked: %q", alphaLine)
	}
	if !strings.Contains(out, "deskkit gui") {
		t.Errorf("desks output must point at the visual browser:\n%s", out)
	}
}

// TestDesksCmd_EmptyStateIsHelpful: nothing seeded must print how to make a desk, never nothing.
func TestDesksCmd_EmptyStateIsHelpful(t *testing.T) {
	seedDesks(t)

	out := runCmdOut(t, newDesksCmd())

	if strings.TrimSpace(out) == "" {
		t.Fatal("desks printed nothing on an empty store home")
	}
	if !strings.Contains(out, "deskkit init") {
		t.Errorf("the empty state must say how to make a desk (`deskkit init`):\n%s", out)
	}
	if !strings.Contains(out, "deskkit gui") {
		t.Errorf("desks output must point at the visual browser:\n%s", out)
	}
}

// TestDesksCmd_UnresolvableConfigIsNotFatal: `desks` is how an operator finds out what exists,
// so a desk-less working directory still lists the stores — with nothing marked.
func TestDesksCmd_UnresolvableConfigIsNotFatal(t *testing.T) {
	base := seedDesks(t, "alpha")
	t.Setenv("DESK_ROOT", "")
	t.Setenv("DESK_NAME", "")

	out := runCmdOut(t, newDesksCmd())

	if !strings.Contains(out, filepath.Join(base, "alpha")) {
		t.Errorf("desks must list stores even when no desk resolves here:\n%s", out)
	}
	if strings.Contains(out, "* alpha") {
		t.Errorf("nothing should be marked when no desk resolves here:\n%s", out)
	}
}

// The "desks creates no store" guarantee is NOT tested here. It comes from main()'s
// interception, not from newDesksCmd(), so an in-process test of it cannot fail — see
// TestReadOnlyCommandsCreateNoStore_Subprocess in configcmd_test.go, which runs the real binary
// and covers `desks` alongside `config`.
