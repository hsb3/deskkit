// The pure, terminal-free parts of the compose-in-$EDITOR escape hatch: resolving which editor to
// launch from the environment, and the temp-file round-trip that carries a draft out to the editor
// and the composed text back. Kept out of model.go so they are unit-tested without launching a real
// editor or standing up a Bubble Tea program (editor_test.go). The Update-side glue — the keybind,
// the tea.ExecProcess wiring, and folding the result back into the textarea — stays in model.go.
package tui

import (
	"os"
	"strings"
)

// editorCommand resolves the external editor invocation from two environment values, taking the
// first non-blank in the order the caller passes them (the caller passes $VISUAL before $EDITOR,
// the git/less convention where VISUAL wins for a full-screen-capable terminal). The value is split
// on whitespace so an editor configured with flags — e.g. "code --wait" or "emacsclient -nw" — is
// honored, not treated as one impossible executable name; the returned slice is the command
// followed by its leading args (the temp path is appended by the caller). It returns nil when
// neither value holds a non-blank string, which the caller treats as "no editor configured".
//
// Pure: it takes the two environment values as arguments rather than reading os.Getenv itself, so
// resolution is exercised across every combination without mutating the process environment.
func editorCommand(primary, secondary string) []string {
	for _, v := range []string{primary, secondary} {
		if fields := strings.Fields(v); len(fields) > 0 {
			return fields
		}
	}
	return nil
}

// writeDraft writes the current composing buffer to a fresh temp file with a .md suffix (so the
// editor applies markdown highlighting) and returns its path. The caller removes the file once the
// editor returns. On any failure the partial file is cleaned up so a failed compose never leaks a
// temp file.
func writeDraft(contents string) (string, error) {
	f, err := os.CreateTemp("", "deskkit-draft-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.WriteString(contents); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

// readDraft reads a composed draft back from the temp file, trimming the trailing newline(s) an
// editor conventionally appends so they do not linger as blank lines in the textarea. Pure over the
// file contents; the caller is responsible for removing the file.
func readDraft(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(b), "\n"), nil
}
