# init-onramp.tape — the config-bootstrap UX: a store-touching command run in a BARE folder
# (no profile, no env) triggers the interactive first-run prompt; accepting scaffolds
# _knowledge/profile.yaml and the original command continues seamlessly. Fully offline (query
# needs no LLM), so this tape records in the default hermetic `make media` run. Regenerate:
# bash scripts/record-media.sh (never hand-edit the GIF).
Output docs/media/init-onramp.gif

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

# A bare folder: no profile, no DESK_* env — the command can't resolve config and offers init.
Type "ls -a" Enter
Sleep 1s
Type "pocket-librarian query summary" Enter
Sleep 1.5s
Type "y" Enter
Sleep 2.5s

# What the onramp wrote, and that the next run needs no prompt at all.
Type "cat _knowledge/profile.yaml" Enter
Sleep 3s
Type "pocket-librarian query summary" Enter
Sleep 2.5s
Sleep 2s
