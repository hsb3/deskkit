package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/config"
)

// hermeticXDG points XDG_DATA_HOME at a throwaway scratch so no test can touch a real
// ~/.local/share store, and returns that scratch dir (mirrors verify.sh's hermetic pattern).
func hermeticXDG(t *testing.T) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)
	return xdg
}

func TestInitProfileFreshDir(t *testing.T) {
	xdg := hermeticXDG(t)
	dir := t.TempDir()

	res, err := InitProfile(dir, InitOptions{}, nil)
	if err != nil {
		t.Fatalf("InitProfile: %v", err)
	}

	wantProfile := filepath.Join(dir, "_knowledge", "profile.yaml")
	if res.ProfilePath != wantProfile {
		t.Errorf("ProfilePath = %q, want %q", res.ProfilePath, wantProfile)
	}
	if res.DeskName != filepath.Base(dir) {
		t.Errorf("DeskName = %q, want %q", res.DeskName, filepath.Base(dir))
	}
	if res.EnvPath != "" {
		t.Errorf("EnvPath = %q, want empty (no --with-env)", res.EnvPath)
	}
	if _, err := os.Stat(wantProfile); err != nil {
		t.Fatalf("profile not written: %v", err)
	}

	// init must NOT create the store dir — assert the resolved XDG path is absent.
	if res.StorePath == "" {
		t.Fatalf("StorePath empty; want a resolved path")
	}
	if !strings.HasPrefix(res.StorePath, xdg) {
		t.Errorf("StorePath = %q, want under XDG %q", res.StorePath, xdg)
	}
	if _, err := os.Stat(res.StorePath); !os.IsNotExist(err) {
		t.Errorf("init created the store dir %q (stat err = %v); it must not", res.StorePath, err)
	}
}

// TestInitProfileRoundTrip proves the written profile parses back through the config loader:
// with no DESK_* env, config.Load() from the init'd dir resolves DeskName + DeskRoot.
func TestInitProfileRoundTrip(t *testing.T) {
	hermeticXDG(t)
	dir := t.TempDir()

	if _, err := InitProfile(dir, InitOptions{}, nil); err != nil {
		t.Fatalf("InitProfile: %v", err)
	}

	// Discovery + parse must both succeed against the written profile.
	pp, ok := config.DiscoverProfile(dir)
	if !ok {
		t.Fatalf("DiscoverProfile did not find the written profile")
	}
	if _, err := config.LoadProfile(pp); err != nil {
		t.Fatalf("LoadProfile on the written profile: %v", err)
	}

	// Full resolution with NO env identity: Load must derive the desk from the profile alone.
	os.Unsetenv("DESK_ROOT")
	os.Unsetenv("DESK_NAME")
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prev) }()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load from init'd dir: %v", err)
	}
	if cfg.DeskName != filepath.Base(dir) {
		t.Errorf("DeskName = %q, want %q", cfg.DeskName, filepath.Base(dir))
	}
	// DeskRoot must point at the dir that owns _knowledge/profile.yaml.
	if _, err := os.Stat(filepath.Join(cfg.DeskRoot, "_knowledge", "profile.yaml")); err != nil {
		t.Errorf("DeskRoot %q does not own the written profile: %v", cfg.DeskRoot, err)
	}
}

func TestInitProfileRefusesExisting(t *testing.T) {
	hermeticXDG(t)
	dir := t.TempDir()

	if _, err := InitProfile(dir, InitOptions{}, nil); err != nil {
		t.Fatalf("first InitProfile: %v", err)
	}
	// Second run without --force must refuse.
	if _, err := InitProfile(dir, InitOptions{}, nil); err == nil {
		t.Fatal("second InitProfile without --force must refuse an existing profile")
	}
}

func TestInitProfileForceOverwrites(t *testing.T) {
	hermeticXDG(t)
	dir := t.TempDir()

	if _, err := InitProfile(dir, InitOptions{}, nil); err != nil {
		t.Fatalf("first InitProfile: %v", err)
	}
	if _, err := InitProfile(dir, InitOptions{Force: true}, nil); err != nil {
		t.Fatalf("InitProfile --force must overwrite: %v", err)
	}
}

func TestInitProfileWithEnv(t *testing.T) {
	hermeticXDG(t)
	dir := t.TempDir()

	res, err := InitProfile(dir, InitOptions{WithEnv: true}, nil)
	if err != nil {
		t.Fatalf("InitProfile --with-env: %v", err)
	}
	if res.EnvPath == "" {
		t.Fatal("EnvPath empty; want a written .env")
	}
	b, err := os.ReadFile(res.EnvPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		"LLM_API_KEY_ENV=ANTHROPIC_API_KEY",
		"# ANTHROPIC_API_KEY=",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
	} {
		if !strings.Contains(got, want) {
			t.Errorf(".env stub missing %q; got:\n%s", want, got)
		}
	}
}

func TestInitProfileEnvClobberRefused(t *testing.T) {
	hermeticXDG(t)
	dir := t.TempDir()

	// Pre-create a .env; --with-env without --force must refuse and write NOTHING.
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("PREEXISTING=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitProfile(dir, InitOptions{WithEnv: true}, nil); err == nil {
		t.Fatal("--with-env must refuse to clobber an existing .env without --force")
	}
	// The pre-existing .env is untouched, and the refusal happened before any profile write.
	b, _ := os.ReadFile(envPath)
	if string(b) != "PREEXISTING=1\n" {
		t.Errorf(".env was modified despite refusal: %q", string(b))
	}
	if _, err := os.Stat(filepath.Join(dir, "_knowledge", "profile.yaml")); !os.IsNotExist(err) {
		t.Error("profile was written despite the pre-write .env-clobber refusal")
	}
}

func TestInitProfileAncestorDeskDetection(t *testing.T) {
	hermeticXDG(t)
	parent := t.TempDir()

	// Make `parent` a desk, then try to init a child inside it.
	if _, err := InitProfile(parent, InitOptions{}, nil); err != nil {
		t.Fatalf("init parent desk: %v", err)
	}
	child := filepath.Join(parent, "sub", "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	// nil confirm (a decline) must refuse the nested desk.
	if _, err := InitProfile(child, InitOptions{}, nil); err == nil {
		t.Fatal("nested desk must be refused when confirmNested declines")
	}
	if _, err := os.Stat(filepath.Join(child, "_knowledge", "profile.yaml")); !os.IsNotExist(err) {
		t.Error("nested profile was written despite refusal")
	}

	// confirmNested that accepts is honored — the ancestor is reported.
	var gotName, gotPath string
	confirm := func(name, path string) (bool, error) {
		gotName, gotPath = name, path
		return true, nil
	}
	res, err := InitProfile(child, InitOptions{}, confirm)
	if err != nil {
		t.Fatalf("nested desk with confirm accept: %v", err)
	}
	if gotName != filepath.Base(parent) {
		t.Errorf("confirmNested desk name = %q, want %q", gotName, filepath.Base(parent))
	}
	if gotPath == "" || res.AncestorProfile == "" {
		t.Errorf("ancestor path not reported: confirm=%q result=%q", gotPath, res.AncestorProfile)
	}

	// --force also creates the nested desk without any confirmation.
	other := filepath.Join(parent, "sub", "child2")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InitProfile(other, InitOptions{Force: true}, nil); err != nil {
		t.Fatalf("nested desk with --force: %v", err)
	}
}

func TestFirstRunDecision(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		isTTY   bool
		noInput bool
		want    bool
		prompt  bool // whether the prompt should have been emitted
	}{
		{"accept-y", "y\n", true, false, true, true},
		{"accept-upper", "Y\n", true, false, true, true},
		{"accept-yes", "yes\n", true, false, true, true},
		{"default-yes-empty", "\n", true, false, true, true},
		{"decline-n", "n\n", true, false, false, true},
		{"decline-upper", "N\n", true, false, false, true},
		{"decline-no", "no\n", true, false, false, true},
		{"decline-other", "maybe\n", true, false, false, true},
		{"no-input", "y\n", true, true, false, false},
		{"non-tty", "y\n", false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w bytes.Buffer
			got, err := FirstRunDecision(strings.NewReader(tc.input), &w, tc.isTTY, tc.noInput)
			if err != nil {
				t.Fatalf("FirstRunDecision: %v", err)
			}
			if got != tc.want {
				t.Errorf("decision = %v, want %v", got, tc.want)
			}
			if emitted := w.Len() > 0; emitted != tc.prompt {
				t.Errorf("prompt emitted = %v (%q), want %v", emitted, w.String(), tc.prompt)
			}
		})
	}
}
