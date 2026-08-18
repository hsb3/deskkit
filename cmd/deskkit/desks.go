package main

import (
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/store"
)

// newDesksCmd builds `desks`: the answer to "which desks does this machine have?", read from the
// store home alone. It opens no store and creates nothing — it is registered on the RootCmd for
// discovery, and main() intercepts the real run before PocketBase can bootstrap one.
// SilenceErrors/Usage keep an error to runIntercepted's single line.
func newDesksCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "desks",
		Short:         "List the desks this machine has a store for",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeDesks(cmd.OutOrStdout())
		},
	}
}

// writeDesks renders the desk list. The desk the CURRENT directory resolves to is marked; a
// config that does not resolve is deliberately not an error here — `desks` is how an operator
// finds out what exists, so it must work from anywhere, just without a mark.
func writeDesks(w io.Writer) error {
	names, base, err := store.ListDesks()
	if err != nil {
		return err
	}

	current := ""
	if cfg, cerr := config.Load(); cerr == nil {
		if dir, derr := resolveStoreLocation(cfg, nil); derr == nil {
			current = dir
		}
	}

	if len(names) == 0 {
		fmt.Fprintf(w, "No desks yet — nothing under %s\n\n", base)
		fmt.Fprintf(w, "Make one: run `deskkit init` in the folder you want to use as a desk, then run any\n")
		fmt.Fprintf(w, "deskkit command from inside it (the store is created on first use).\n\n")
		fmt.Fprintf(w, "Browse a desk's data visually with `deskkit gui`.\n")
		return nil
	}

	fmt.Fprintf(w, "Desks with a store under %s:\n\n", base)
	marked := false
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, name := range names {
		path := filepath.Join(base, name)
		marker := " "
		if path == current {
			marker, marked = "*", true
		}
		fmt.Fprintf(tw, "  %s %s\t%s\n", marker, name, path)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(w)
	switch {
	case marked:
		fmt.Fprintf(w, "* = the desk this directory resolves to.\n")
	case current != "":
		fmt.Fprintf(w, "This directory resolves to %s, which has no store yet (it is created on first use).\n", current)
	default:
		fmt.Fprintf(w, "No desk resolves here — run deskkit from inside a desk, or set DESK_NAME.\n")
	}
	fmt.Fprintf(w, "Browse a desk's data visually with `deskkit gui`.\n")
	return nil
}
