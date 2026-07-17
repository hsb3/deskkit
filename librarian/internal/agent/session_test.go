package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/example/pocket-librarian/internal/config"

	// Blank-import registers this project's Go migrations into the same global registry
	// PocketBase's built-in migrations use, so tests.NewTestApp's RunAllMigrations() creates
	// our collections (agent_runs, messages, prompts, ...) too.
	_ "github.com/example/pocket-librarian/migrations"
)

// fakeChatModel is a stub model.ToolCallingChatModel that never emits tool calls and echoes
// the last user message back as the assistant reply. It records the length of every message
// slice it is asked to Generate over, so a test can prove the history grew across turns
// (multi-turn = later Generate calls receive strictly more messages than earlier ones).
type fakeChatModel struct {
	mu       sync.Mutex
	genLens  []int // number of messages received on each Generate call
	lastUser []string
}

func (f *fakeChatModel) reply(input []*schema.Message) *schema.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.genLens = append(f.genLens, len(input))
	last := ""
	for i := len(input) - 1; i >= 0; i-- {
		if input[i].Role == schema.User {
			last = input[i].Content
			break
		}
	}
	f.lastUser = append(f.lastUser, last)
	return schema.AssistantMessage("echo: "+last, nil)
}

func (f *fakeChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return f.reply(input), nil
}

func (f *fakeChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg := f.reply(input)
	return schema.StreamReaderFromArray([]*schema.Message{msg}), nil
}

// WithTools returns the same instance (the fake ignores tools entirely — it never calls one).
func (f *fakeChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}

// newSessionTestEnv boots a fresh PocketBase test app (own temp data dir, all migrations
// applied) plus a Config pointed at a fresh temp desk root, mirroring the tools-package
// scaffolding pattern (which cannot be imported across packages).
func newSessionTestEnv(t *testing.T) (*tests.TestApp, *config.Config) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	deskRoot := t.TempDir()
	cfg := &config.Config{
		DeskRoot:     deskRoot,
		DeskName:     "test-desk",
		DecisionsDir: "_structure/decisions",
		TasksDir:     "tasks",
		AnalysesDir:  "analyses",
		JournalDir:   "journal",
		SecretsDir:   "_meta/secrets",
		IgnoreConfig: filepath.Join(deskRoot, ".librarian-ignore"),
		HandoffPath:  "_meta/HANDOFF.md",
		LLMProvider:  "openai", // any non-anthropic value avoids the streaming tool-call checker
		LLMModel:     "test-model",
		LLMMaxTokens: 256,
		AgentMaxStep: 12,
	}
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("# empty — nothing ignored\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}
	return app, cfg
}

// TestSession_MultiTurn drives two Turns against the fake model and asserts (a) each Turn
// returns the canned/derived content, and (b) the second Turn's model input carried strictly
// more messages than the first — proving the conversation history is replayed and grows.
func TestSession_MultiTurn(t *testing.T) {
	fake := &fakeChatModel{}
	orig := chatModelFactory
	chatModelFactory = func(ctx context.Context, cfg *config.Config) (model.ToolCallingChatModel, error) {
		return fake, nil
	}
	t.Cleanup(func() { chatModelFactory = orig })

	app, cfg := newSessionTestEnv(t)
	ctx := context.Background()

	sess, err := NewSession(ctx, app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	out1, err := sess.Turn(ctx, "first question")
	if err != nil {
		t.Fatalf("Turn 1: %v", err)
	}
	if out1 != "echo: first question" {
		t.Fatalf("Turn 1 content = %q, want %q", out1, "echo: first question")
	}

	out2, err := sess.Turn(ctx, "second question")
	if err != nil {
		t.Fatalf("Turn 2: %v", err)
	}
	if out2 != "echo: second question" {
		t.Fatalf("Turn 2 content = %q, want %q", out2, "echo: second question")
	}

	// The fake records the message-slice length on each Generate. The FIRST Generate call of
	// turn 2 must have seen more messages than the FIRST call of turn 1, because turn 1's
	// user+assistant messages are replayed. (A single-step loop yields exactly one Generate
	// per Turn; guard for that but assert on the first call of each turn regardless.)
	fake.mu.Lock()
	lens := append([]int(nil), fake.genLens...)
	fake.mu.Unlock()
	if len(lens) < 2 {
		t.Fatalf("expected at least 2 Generate calls, got %d (%v)", len(lens), lens)
	}
	firstTurnLen := lens[0]
	// Find the first Generate call whose input length exceeds firstTurnLen — that is turn 2
	// seeing turn 1's accumulated history.
	grew := false
	for _, l := range lens[1:] {
		if l > firstTurnLen {
			grew = true
			break
		}
	}
	if !grew {
		t.Fatalf("history did not grow across turns; Generate input lengths = %v", lens)
	}

	if err := sess.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Sanity: the run row was finalized to a terminal (non-running) status.
	rec, err := app.FindRecordById("agent_runs", sess.run.Id)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if status := rec.GetString("status"); status == "running" || status == "" {
		t.Fatalf("run status after Close = %q, want a terminal status", status)
	}
	// Guard the neutrality note that this test is desk-stewardship scoped, not a chat toy:
	// the assistant content must be exactly our derived echo, never free-form.
	if !strings.HasPrefix(out2, "echo: ") {
		t.Fatalf("assistant content not derived from user input: %q", out2)
	}
}

// TestCapHistory_BoundsGrowth proves the sliding-window cap keeps the session history bounded
// no matter how many turns accumulate, and retains the most recent messages (drops the oldest).
func TestCapHistory_BoundsGrowth(t *testing.T) {
	var h []*schema.Message
	var newest *schema.Message
	for i := 0; i < maxHistoryMessages*3; i++ {
		m := schema.UserMessage("m")
		newest = m
		h = append(h, m)
		h = capHistory(h)
		if len(h) > maxHistoryMessages {
			t.Fatalf("history grew past the cap: len=%d > %d", len(h), maxHistoryMessages)
		}
	}
	if len(h) != maxHistoryMessages {
		t.Fatalf("final history len = %d, want %d", len(h), maxHistoryMessages)
	}
	if h[len(h)-1] != newest {
		t.Fatalf("cap dropped the most recent message; the retained tail must keep the newest turns")
	}
}
