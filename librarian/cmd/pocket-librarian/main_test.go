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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firstSubcommand(c.args); got != c.want {
				t.Errorf("firstSubcommand(%v) = %q, want %q", c.args, got, c.want)
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
