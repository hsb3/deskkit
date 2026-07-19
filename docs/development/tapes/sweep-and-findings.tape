# sweep-and-findings.tape — index a desk tree, then inspect open findings and orphaned docs.
# Regenerate: bash scripts/record-media.sh (never hand-edit the GIF).
Output docs/media/sweep-and-findings.gif

Require deskkit

Set Shell "bash"
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 16
Set Theme "Dracula"
Set WindowBar Colorful
Set TypingSpeed 35ms
Set Framerate 24

# Scratch-path plumbing (desk location, store home, clean prompt) — never shown; the script
# supplies MEDIA_DESK_ROOT / MEDIA_XDG_HOME as env vars before invoking vhs.
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
Type "deskkit sweep" Enter
Sleep 2200ms

# A hidden patrol pass gives `query findings` real content to show — findings/orphans are
# populated by patrol, not sweep — without pulling `patrol` into the command sequence this
# tape is actually teaching (that's patrol.tape's job).
Hide
Type "deskkit patrol > /dev/null 2>&1" Enter
Sleep 400ms
Type "clear" Enter
Show

Type "deskkit query findings --pretty" Enter
Sleep 2200ms
Type "deskkit query orphans --pretty" Enter
Sleep 2200ms
