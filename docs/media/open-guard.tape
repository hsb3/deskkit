# open-guard.tape — the desk open-guard (ADR 0002 section 3): reopening the SAME store dir under a
# different DESK_NAME is refused, naming both desks in the error.
# Regenerate: bash scripts/record-media.sh (never hand-edit the GIF).
Output docs/media/open-guard.gif

Require pocket-librarian

Set Shell "bash"
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 16
Set Theme "Dracula"
Set WindowBar Colorful
Set TypingSpeed 35ms
Set Framerate 24

# Scratch-path plumbing (desk location, store home, the first desk's own store dir for the
# reopen-with---dir step below) — never shown; the script supplies MEDIA_DESK_ROOT /
# MEDIA_XDG_HOME / MEDIA_STORE_ONE as env vars before invoking vhs.
Hide
Type `export PS1='$ '`
Enter
Type `cd "$MEDIA_DESK_ROOT"`
Enter
Type `export XDG_DATA_HOME="$MEDIA_XDG_HOME"`
Enter
Type `export STORE_ONE_DIR="$MEDIA_STORE_ONE"`
Enter
Type "clear" Enter
Show

Type `export DESK_ROOT="$PWD"`
Enter
Sleep 500ms
Type "export DESK_NAME=example-desk" Enter
Sleep 700ms
Type "pocket-librarian query summary --pretty" Enter
Sleep 1800ms

Type "export DESK_NAME=another-desk" Enter
Sleep 700ms
Type `pocket-librarian query summary --dir "$STORE_ONE_DIR"`
Enter
Sleep 2800ms
