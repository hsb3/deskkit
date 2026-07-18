# propose-apply-restore.tape — the safety loop: propose a mechanical fix, apply it, then
# restore proves the change is byte-exact reversible (record-original-first, ADR/decision
# 0014). Regenerate: bash scripts/record-media.sh (never hand-edit the GIF).
Output docs/media/propose-apply-restore.gif

Require pocket-librarian

Set Shell "bash"
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 16
Set Theme "Dracula"
Set WindowBar Colorful
Set TypingSpeed 20ms
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
Sleep 400ms
Type "export DESK_NAME=example-desk" Enter
Sleep 500ms
Type "cat tasks/status-update.md" Enter
Sleep 1200ms
Type "pocket-librarian propose-fix" Enter
Sleep 1300ms
Type "pocket-librarian apply-fix" Enter
Sleep 1300ms
Type "cat tasks/status-update.md" Enter
Sleep 1200ms
Type "pocket-librarian restore --by-path tasks/status-update.md" Enter
Sleep 1300ms
Type "cat tasks/status-update.md" Enter
Sleep 1200ms
