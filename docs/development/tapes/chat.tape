# chat.tape — the full-screen chat TUI: a streamed answer with a visible tool call (sweep),
# a brief look at expanded step detail, then a clean quit. Needs a real LLM provider key (the
# model's own sweep call talks to Anthropic) — scripts/record-media.sh only records this tape
# when ANTHROPIC_API_KEY is set in the environment, and skips it otherwise so the default
# `make media` run stays hermetic. Regenerate: bash scripts/record-media.sh (never hand-edit
# the GIF); to record THIS one specifically: export ANTHROPIC_API_KEY=... and re-run.
Output docs/media/chat.gif

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
Type "pocket-librarian chat" Enter
Sleep 1500ms

Type "Sweep this desk, then tell me in one sentence what's in it." Enter
Sleep 30s

# Expand step detail briefly on the tool call the model just made, then quit clean.
Ctrl+T
Sleep 3s
Ctrl+C
Sleep 500ms
