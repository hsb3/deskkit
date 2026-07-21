package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestDevModeDefaultOff_BinaryUnderTempDir is a regression test for the bug where
// pocketbase.Config.DefaultDev was wired to osutils.IsProbablyGoRun(), which returns true
// whenever os.Args[0] is prefixed by os.TempDir() (or the Go build-cache dir) — a heuristic
// meant to detect `go run`'s temp-binary pattern, but one that also fires for ANY binary
// invoked from a path under the OS temp dir. Before the fix, building deskkit via
// `go build -o "$(mktemp -d)/deskkit" ./cmd/deskkit` (a perfectly normal packaging step, and
// exactly what a scratch/CI build can do) silently turned on PocketBase's dev mode, which
// prints raw SQL debug lines (e.g. "[0.12ms] CREATE TABLE ...") to STDOUT via
// color.HiBlack/fmt.Print — corrupting any client that treats stdout as machine-readable (the
// motivating case: an MCP stdio JSON-RPC stream). After the fix DefaultDev is unconditionally
// false, so running the SAME binary — built to the SAME kind of temp-dir path — against a
// brand-new store (a `migrate up`, which necessarily executes a pile of CREATE TABLE
// statements) must NOT leak any SQL debug lines to stdout.
//
// This builds a real binary and runs it as a subprocess rather than calling main()'s internals
// directly: the bug is specifically about os.Args[0]'s path at process-invocation time, which
// only a real subprocess invocation can reproduce faithfully.
func TestDevModeDefaultOff_BinaryUnderTempDir(t *testing.T) {
	// Build to a path explicitly under os.TempDir() (not just t.TempDir(), whose OS-level
	// backing directory can vary by platform/runner) — the exact shape of the reported bug,
	// and of the `mktemp -d`-based build command that triggered it.
	buildDir, err := os.MkdirTemp(os.TempDir(), "deskkit-devmode-build-")
	if err != nil {
		t.Fatalf("MkdirTemp under os.TempDir(): %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(buildDir) })

	binName := "deskkit-under-tmp"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(buildDir, binName)

	if !strings.HasPrefix(binPath, os.TempDir()) {
		t.Fatalf("test setup bug: built binary path %q is not under os.TempDir() %q", binPath, os.TempDir())
	}

	// cwd is this package's directory (cmd/deskkit) under `go test`, so "." builds the current
	// (fixed) package — the same package main.go lives in.
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, berr := build.CombinedOutput(); berr != nil {
		t.Fatalf("go build deskkit into %s: %v\n%s", binPath, berr, out)
	}

	storeDir := filepath.Join(t.TempDir(), "store")

	// --dir is explicit, so `migrate` never resolves DESK_NAME/config (ADR 0002 §2; see the
	// "migrate is schema-only" note in main.go) — a hermetic invocation needs nothing else. A
	// brand-new store dir guarantees the migration runner actually executes CREATE TABLE
	// statements, so the pre-fix binary would reliably leak debug output here.
	run := exec.Command(binPath, "--dir", storeDir, "migrate", "up")
	run.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	if rerr := run.Run(); rerr != nil {
		t.Fatalf("%s --dir %s migrate up failed: %v\nstdout:\n%s\nstderr:\n%s",
			binPath, storeDir, rerr, stdout.String(), stderr.String())
	}

	out := stdout.String()
	if strings.Contains(out, "CREATE TABLE") {
		t.Errorf("stdout leaked PocketBase dev-mode SQL debug output (dev mode must default to false when invoked from a path under os.TempDir()):\n%s", out)
	}
	if m := regexp.MustCompile(`(?m)^\[\d+\.\d+ms\]`).FindString(out); m != "" {
		t.Errorf("stdout leaked a PocketBase dev-mode timing-prefixed debug line (%q); dev mode must default to false:\n%s", m, out)
	}
}
