package main

import "testing"

func TestHasDirFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"absent", []string{"sweep"}, false},
		{"value form", []string{"sweep", "--dir", "/tmp/store"}, true},
		{"equals form", []string{"sweep", "--dir=/tmp/store"}, true},
		{"valueless trailing", []string{"sweep", "--dir"}, true},
		{"before the subcommand", []string{"--dir", "/tmp/store", "sweep"}, true},
		{"equals form before the subcommand", []string{"--dir=/tmp/store", "sweep"}, true},
		{"unrelated flag only", []string{"sweep", "--dev"}, false},
		{"unrelated flag with similar prefix does not match", []string{"sweep", "--directory=/x"}, false},
		{"empty args", []string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasDirFlag(c.args); got != c.want {
				t.Errorf("hasDirFlag(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestFirstSubcommand(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"bare subcommand", []string{"sweep"}, "sweep"},
		{"subcommand with trailing flags", []string{"sweep", "--dir", "/tmp/store"}, "sweep"},
		{"flags before the subcommand, value form", []string{"--dir", "/tmp/store", "serve"}, "serve"},
		{"flags before the subcommand, equals form", []string{"--dir=/tmp/store", "serve"}, "serve"},
		{"boolean global flag before the subcommand", []string{"--dev", "serve"}, "serve"},
		{"multiple value flags before the subcommand", []string{"--dir", "/x", "--queryTimeout", "5", "query"}, "query"},
		{"no subcommand (bare invocation)", []string{}, ""},
		{"only flags, no subcommand", []string{"--dev"}, ""},
		// Regression coverage for the --hooksDir bypass: an unregistered-in-this-build but
		// recognized value flag must still have its value skipped, not mistaken for the
		// subcommand — this is exactly how "--hooksDir /some/path serve" previously resolved to
		// "/some/path" (not store-touching) and let PocketBase Bootstrap create a stray data dir.
		{"hooksDir value form before the subcommand", []string{"--hooksDir", "/some/path", "serve"}, "serve"},
		{"hooksDir equals form before the subcommand", []string{"--hooksDir=/some/path", "serve"}, "serve"},
		{"hooksWatch value form before the subcommand", []string{"--hooksWatch", "true", "serve"}, "serve"},
		{"hooksPool value form before the subcommand", []string{"--hooksPool", "25", "serve"}, "serve"},
		{"multiple jsvm-style value flags before the subcommand", []string{"--hooksDir", "/x", "--hooksPool", "25", "sweep"}, "sweep"},
		// The narrated residual gap: a value flag NOT on globalValueFlags still shadows the
		// subcommand token — its value is treated as the subcommand. Pinned as current behavior
		// (documented in the globalValueFlags comment as "a genuinely novel unrecognized
		// value-flag ... could still shadow the subcommand token the same way"), not endorsed.
		{"unrecognized value flag shadows the subcommand", []string{"--novel", "/x", "serve"}, "/x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firstSubcommand(c.args); got != c.want {
				t.Errorf("firstSubcommand(%v) = %q, want %q", c.args, got, c.want)
			}
		})
	}
}

func TestSubcommandIndex(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"bare subcommand", []string{"sweep"}, 0},
		{"value flag then subcommand", []string{"--dir", "/x", "serve"}, 2},
		{"equals form does not skip a value token", []string{"--dir=/x", "serve"}, 1},
		{"boolean global flag before the subcommand", []string{"--dev", "serve"}, 1},
		{"two value flags before the subcommand", []string{"--dir", "/x", "--queryTimeout", "5", "query"}, 4},
		{"no subcommand (bare invocation)", []string{}, -1},
		{"only flags, no subcommand", []string{"--dev"}, -1},
		{"trailing value flag with no value", []string{"--dir"}, -1},
		{"unrecognized value flag shadows the subcommand", []string{"--novel", "/x", "serve"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := subcommandIndex(c.args); got != c.want {
				t.Errorf("subcommandIndex(%v) = %d, want %d", c.args, got, c.want)
			}
		})
	}
}

func TestIsInitInvocation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare init", []string{"init"}, true},
		{"value flag before init", []string{"--dir", "/x", "init"}, true},
		{"init with force flag", []string{"init", "--force"}, true},
		{"init with help long flag", []string{"init", "--help"}, false},
		{"init with help short flag anywhere", []string{"init", "-h"}, false},
		{"help short flag before init", []string{"-h", "init"}, false},
		{"a different subcommand", []string{"serve"}, false},
		{"bare invocation", []string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isInitInvocation(c.args); got != c.want {
				t.Errorf("isInitInvocation(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestHasNoInputFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare no-input", []string{"init", "--no-input"}, true},
		{"equals true", []string{"init", "--no-input=true"}, true},
		{"equals false still present", []string{"init", "--no-input=false"}, true},
		{"absent", []string{"init", "--force"}, false},
		{"similar prefix does not match", []string{"--no-inputs"}, false},
		{"empty args", []string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasNoInputFlag(c.args); got != c.want {
				t.Errorf("hasNoInputFlag(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestStripNoInput(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"strips bare token", []string{"--no-input", "/dir"}, []string{"/dir"}},
		{"strips equals form", []string{"--no-input=true", "/dir"}, []string{"/dir"}},
		{"leaves other flags", []string{"--force", "/dir"}, []string{"--force", "/dir"}},
		{"strips only the no-input token", []string{"--no-input", "--force", "/dir"}, []string{"--force", "/dir"}},
		{"nothing to strip", []string{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripNoInput(c.args)
			if len(got) != len(c.want) {
				t.Fatalf("stripNoInput(%v) = %v, want %v", c.args, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("stripNoInput(%v) = %v, want %v", c.args, got, c.want)
				}
			}
		})
	}
}

func TestIsStoreTouchingInvocation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"store-touching subcommand", []string{"sweep"}, true},
		{"serve", []string{"serve"}, true},
		{"migrate", []string{"migrate", "up"}, true},
		{"help long flag excluded even with a store-touching subcommand present", []string{"sweep", "--help"}, false},
		{"help short flag excluded", []string{"-h"}, false},
		{"version short flag excluded", []string{"-v"}, false},
		{"version long flag excluded", []string{"--version"}, false},
		{"help before the subcommand excluded", []string{"--help", "serve"}, false},
		{"unknown command is not store-touching", []string{"frobnicate"}, false},
		{"bare invocation is not store-touching", []string{}, false},
		{"flags before the subcommand still resolve", []string{"--dir", "/x", "sweep"}, true},
		{"completion-style unknown subcommand not store-touching", []string{"completion", "bash"}, false},
		// The --hooksDir bypass, end to end: before globalValueFlags recognized these, this case
		// resolved firstSubcommand to "/some/path" and returned false here, silently skipping the
		// unresolved-location guard for a real `serve` invocation.
		{"hooksDir before serve does not bypass the guard", []string{"--hooksDir", "/some/path", "serve"}, true},
		{"hooksWatch before serve does not bypass the guard", []string{"--hooksWatch", "true", "serve"}, true},
		{"hooksPool before serve does not bypass the guard", []string{"--hooksPool", "25", "serve"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isStoreTouchingInvocation(c.args); got != c.want {
				t.Errorf("isStoreTouchingInvocation(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}
