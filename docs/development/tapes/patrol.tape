# patrol.tape — a patrol run over an already-swept desk, surfacing real R1/R2 rule findings.
# Regenerate: bash scripts/record-media.sh (never hand-edit the GIF).
Output docs/media/patrol.gif

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

# Scratch-path plumbing — never shown; the script supplies MEDIA_DESK_ROOT / MEDIA_XDG_HOME.
Hide
Type `export PS1='$ '`
Enter
Type `cd "$MEDIA_DESK_ROOT"`
Enter
Type `export XDG_DATA_HOME="$MEDIA_XDG_HOME"`
Enter
Type "clear" Enter
Show

Type `export DESK_ROOT="$PWD"`
Enter
Sleep 500ms
Type "export DESK_NAME=example-desk" Enter
Sleep 700ms
Type "pocket-librarian patrol" Enter
Sleep 2200ms
Type "pocket-librarian query findings --pretty" Enter
Sleep 2500ms
