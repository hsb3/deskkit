# chat-light.tape — the chat TUI on a LIGHT terminal background: proves the answer text is
# readable with the light palette (--theme light; auto-detection does the same on a real light
# terminal), then shows the help toggle and the copy-answer toast. Needs a real LLM provider
# key, exactly like chat.tape — scripts/record-media.sh only records it when ANTHROPIC_API_KEY
# is set, and skips it otherwise. Regenerate: bash scripts/record-media.sh (never hand-edit the
# GIF).
Output docs/media/chat-light.gif

Require pocket-librarian

Set Shell "bash"
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 16
Set Theme "Catppuccin Latte"
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
Type "pocket-librarian chat --theme light" Enter
Sleep 1500ms

Type "Sweep this desk, then tell me in one sentence what's in it." Enter
Sleep 30s

# Full-help toggle, copy-answer toast, then a clean quit.
Ctrl+G
Sleep 3s
Ctrl+G
Sleep 1s
Ctrl+Y
Sleep 2s
Ctrl+C
Sleep 500ms
