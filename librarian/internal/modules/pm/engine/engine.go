// Package engine is the PM module's single transition path (spec §4.1): the code path every
// surface (the D4 MCP/CLI/TUI tools) routes through. It owns the §4.1 refusal sequence
// (machine → blocked → claim → gates → write + audit + cascade), the §3.5 cascade semantics,
// the §3.6 optimistic-concurrency version token + claim TTL, and the §3.8 desk_config
// loading. Documents are consulted ONLY through the core/schema seam (§2.5) — this package
// never imports modules/librarian (test lane §10.5 guards that).
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/schema"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/gates"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/statemachine"
)

// Engine binds the store, config, and the injected document validator (module.Registry
// captures it at registration; nil = documented gates fail closed, §2.5).
type Engine struct {
	App       core.App
	Cfg       *config.Config
	Validator schema.DocumentValidator
}

// Refusal re-exports the gates refusal type: every "no" this engine says is one of these,
// carrying the exact human-readable reasons (R3.1); anything else is an internal error.
type Refusal = gates.Refusal

func refuse(format string, args ...any) error {
	return &Refusal{Reasons: []string{fmt.Sprintf(format, args...)}}
}

// IsRefusal reports whether err is a refusal (vs a broken store/config).
func IsRefusal(err error) bool {
	var r *Refusal
	return errors.As(err, &r)
}

// Actor identifies who acts, for the §3.6 audit trail.
type Actor struct {
	Name             string // a human handle or an agent id; "" is recorded as-is (plain strings, R2.5)
	Kind             string // "human" | "agent"
	DelegationParent string // parent agent/session id when acting under delegation
}

// deskConfig is the resolved per-desk workflow config (§3.8): stored row > shipped defaults.
type deskConfig struct {
	rules    *gates.Config
	labels   map[string]statemachine.Phase
	claimTTL time.Duration
}

// loadDeskConfig resolves the desk's workflow config. A stored-but-invalid rules YAML is a
// loud error (fail-loud, §4.2) — it never silently falls back to defaults, because that would
// silently disable the desk's own gates.
func (e *Engine) loadDeskConfig() (*deskConfig, error) {
	dc := &deskConfig{
		labels:   statemachine.DefaultStatusLabels(),
		claimTTL: 30 * time.Minute,
	}
	if e.Cfg != nil && e.Cfg.PMClaimTTL > 0 {
		dc.claimTTL = e.Cfg.PMClaimTTL
	}
	defaultRules, err := gates.DefaultConfig()
	if err != nil {
		return nil, err
	}
	dc.rules = defaultRules

	rec, ferr := e.App.FindFirstRecordByFilter("desk_config", "desk = {:d}", map[string]any{"d": e.desk()})
	if ferr != nil {
		return dc, nil // no stored row: the shipped defaults apply (§4.2 seed)
	}
	if rulesYAML := rec.GetString("rules"); rulesYAML != "" {
		parsed, perr := gates.ParseRules(rulesYAML)
		if perr != nil {
			return nil, fmt.Errorf("desk_config for desk %q holds invalid gate rules: %w", e.desk(), perr)
		}
		dc.rules = parsed
	}
	if labelsRaw := rec.Get("status_labels"); labelsRaw != nil {
		if labels, lerr := gates.ParseLabels(rec.GetString("status_labels")); lerr != nil {
			return nil, fmt.Errorf("desk_config for desk %q holds invalid status_labels: %w", e.desk(), lerr)
		} else if len(labels) > 0 {
			dc.labels = labels
		}
	}
	if mins := rec.GetInt("claim_ttl_minutes"); mins > 0 {
		dc.claimTTL = time.Duration(mins) * time.Minute
	}
	return dc, nil
}

func (e *Engine) desk() string {
	if e.Cfg == nil {
		return ""
	}
	return e.Cfg.DeskName
}

// CreateItemInput seeds one work item (§3.1). Phase defaults to queue.
type CreateItemInput struct {
	// ID, when non-empty, pins the new record's id instead of auto-generating one. It exists
	// for the deterministic import path (§8.1/§8.2 rebuild reproducibility): a manifest-derived
	// id makes a rebuild byte-identical, ids included. It must be a valid PocketBase record id
	// (15 chars of [a-z0-9]); the importer derives one that fits. "" keeps the normal
	// auto-generated id, so every non-import caller is unaffected.
	ID       string
	Title    string
	Type     string
	Parent   string // parent item id; "" = a root
	Court    string
	Pointer  string
	Severity string
	Priority int
	Actor    Actor
}

// CreateItem writes one items row. The root denormalization (§3.1) is derived from the
// parent chain; a root item is its own subtree head (root left empty, mirroring parent).
func (e *Engine) CreateItem(ctx context.Context, in CreateItemInput) (*core.Record, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, refuse("cannot create an item without a title")
	}
	if in.Type != "" {
		// Empty type stays legal (a scope call, ADR 0012): only a NON-EMPTY, unrecognized
		// type is refused, so `type` remains optional across every caller-facing shape.
		// A pure vocabulary read, so it fails before any transaction is opened.
		vocab, verr := schema.Vocab()
		if verr != nil {
			return nil, verr
		}
		if !vocab.KnownType(in.Type) {
			return nil, refuse("unknown item type %q (known types: %s; see schema/doctypes.yaml)",
				in.Type, strings.Join(vocab.TypeNames(), ", "))
		}
	}
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		col, err := txe.App.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		if in.ID != "" {
			rec.Id = in.ID // deterministic import id (§8.2); "" leaves PocketBase to auto-generate
		}
		rec.Set("desk", txe.desk())
		rec.Set("title", in.Title)
		rec.Set("type", in.Type)
		rec.Set("phase", string(statemachine.Queue))
		rec.Set("status_label", statemachine.DefaultLabelFor(statemachine.Queue))
		rec.Set("court", in.Court)
		rec.Set("pointer", in.Pointer)
		rec.Set("severity", in.Severity)
		rec.Set("priority", in.Priority)
		rec.Set("version", 1)
		if in.Parent != "" {
			// The parent read + the child write share this transaction: a concurrent delete of the
			// parent can't slip between them.
			parent, perr := txe.loadItem(in.Parent)
			if perr != nil {
				return perr
			}
			rec.Set("parent", parent.Id)
			root := parent.GetString("root")
			if root == "" {
				root = parent.Id
			}
			rec.Set("root", root)
		}
		if err := txe.App.Save(rec); err != nil {
			return err
		}
		out = rec
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// loadItem fetches an item by id, desk-scoped (§3.1 desk field; ADR 0002 discipline).
func (e *Engine) loadItem(id string) (*core.Record, error) {
	rec, err := e.App.FindRecordById("items", id)
	if err != nil {
		return nil, refuse("no item %q on this desk", id)
	}
	if rec.GetString("desk") != e.desk() {
		return nil, refuse("item %q belongs to desk %q, not %q", id, rec.GetString("desk"), e.desk())
	}
	return rec, nil
}

// checkVersion enforces the §3.6 optimistic-concurrency token: every mutating call takes the
// version its caller read and refuses on mismatch.
func checkVersion(rec *core.Record, version int) error {
	if current := rec.GetInt("version"); current != version {
		return &Refusal{Reasons: []string{fmt.Sprintf(
			"item %q changed since you read it (version %d, you had %d); re-read and retry",
			rec.Id, current, version)}}
	}
	return nil
}

// liveForeignClaim reports the holder of a live claim by someone other than actor ("" = none).
// An expired claim is treated as free (§3.6).
func liveForeignClaim(rec *core.Record, actor Actor, now time.Time) string {
	holder := rec.GetString("claimed_by")
	if holder == "" || holder == actor.Name {
		return ""
	}
	expires := rec.GetDateTime("claim_expires").Time()
	if expires.IsZero() || !expires.After(now) {
		return ""
	}
	return holder
}

// bump increments the version token on a mutation.
func bump(rec *core.Record) { rec.Set("version", rec.GetInt("version")+1) }

// txFailpoint, when non-nil, is invoked inside a mutating transaction immediately after the
// primary record write and before the audit/cascade writes. It exists ONLY so tests can force a
// mid-sequence failure and prove the load->version-check->mutate->save->audit->cascade sequence
// commits or rolls back as one unit (§3.6). It is nil on every shipped path; production never
// sets it. It is a plain package-level var with no synchronization: tests that set it must run
// serially (the engine tests do) — a future t.Parallel() case must not touch it. The seam is
// wired ONLY in transitionCore (the longest write sequence); the other RunInTransaction methods
// share the same commit-or-rollback mechanics but have no forced-failure test — add a
// runFailpoint() call there for parity if one is ever wanted.
var txFailpoint func() error

func runFailpoint() error {
	if txFailpoint != nil {
		return txFailpoint()
	}
	return nil
}

// withApp returns a tx-scoped copy of the engine bound to app (the RunInTransaction callback's
// txApp). Every inner read AND write of a mutating method runs through this copy, so the
// version-guard read and the write it authorizes share one transaction — closing the §3.6
// check-then-act TOCTOU. Cfg/Validator are immutable and safely shared.
func (e *Engine) withApp(app core.App) *Engine {
	return &Engine{App: app, Cfg: e.Cfg, Validator: e.Validator}
}

// pendingAudit is a gate_refused transitions row (§4.1) captured inside a transaction but written
// AFTER it settles. A gate refusal mutates nothing, so its transaction rolls back; the row must
// still persist (observable, not silent), which means writing it outside the rolled-back tx —
// exactly the pre-transaction behavior (a single, non-atomic audit write; the refusal stands even
// if that write fails).
type pendingAudit struct {
	itemID, fromPhase, toPhase, event, detail string
	actor                                     Actor
}

// audit appends one transitions row (§3.6 append-only; nothing in this engine ever updates
// or deletes one).
func (e *Engine) audit(itemID, fromPhase, toPhase, event string, actor Actor, detail string) error {
	col, err := e.App.FindCollectionByNameOrId("transitions")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("item", itemID)
	rec.Set("from_phase", fromPhase)
	rec.Set("to_phase", toPhase)
	rec.Set("event", event)
	rec.Set("actor", actor.Name)
	rec.Set("actor_kind", actor.Kind)
	rec.Set("delegation_parent", actor.DelegationParent)
	rec.Set("detail", detail)
	return e.App.Save(rec)
}

// TransitionInput requests one legal phase transition — advance, demote, or reopen — via
// (item, target phase); the machine derives the edge kind (§4.1, one generic tool).
type TransitionInput struct {
	ItemID      string
	TargetPhase string
	Version     int
	Actor       Actor
}

// Transition is THE §4.1 sequence. On success: phase written (status_label kept mapped,
// §3.3), version bumped, a transitions row appended, the cascade scan run (§3.5). On a gate
// refusal: a gate_refused transitions row is appended (observable, not silent) and the
// *Refusal returned names exactly what is missing (R3.1).
func (e *Engine) Transition(ctx context.Context, in TransitionInput) (*core.Record, error) {
	var out *core.Record
	var pending *pendingAudit
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		rec, p, err := e.withApp(txApp).transitionCore(ctx, in)
		pending = p
		if err != nil {
			return err
		}
		out = rec
		return nil
	})
	// The gate_refused row is written after the tx settles (§4.1): a refusal rolls the tx back,
	// but the row must persist regardless.
	if pending != nil {
		_ = e.audit(pending.itemID, pending.fromPhase, pending.toPhase, pending.event, pending.actor, pending.detail)
	}
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// transitionCore runs THE §4.1 sequence and MUST be called on a tx-scoped engine (withApp) that
// is already inside a transaction — it opens none of its own, so a caller can compose it with
// further writes (SetStatusLabel's label pin) in the same atomic unit. On a gate refusal it
// returns (nil, pendingAudit, *Refusal): the pendingAudit is the gate_refused row the caller must
// write after the transaction settles; no mutation has occurred, so the tx safely rolls back.
func (e *Engine) transitionCore(ctx context.Context, in TransitionInput) (*core.Record, *pendingAudit, error) {
	item, err := e.loadItem(in.ItemID)
	if err != nil {
		return nil, nil, err
	}
	if err := checkVersion(item, in.Version); err != nil {
		return nil, nil, err
	}
	from := statemachine.Phase(item.GetString("phase"))
	to, perr := statemachine.ParsePhase(in.TargetPhase)
	if perr != nil {
		return nil, nil, refuse("%v", perr)
	}

	// 1. The machine admits the edge, else refuse before gates are even consulted (§3.2).
	event, legal := statemachine.Edge(from, to)
	if !legal {
		return nil, nil, refuse("no legal transition %s->%s", from, to)
	}
	// 2. Blocked refuses forward edges (§3.2: advance is refused while blocked).
	if item.GetBool("blocked") && event == statemachine.Advance {
		return nil, nil, refuse("item %q is blocked; unblock it (or resolve its blockers) before advancing", item.Id)
	}
	// 3. A live foreign claim refuses advance/demote (§3.6/R2.6). Reopen too: it is a
	// mutation of a claimed item all the same.
	if holder := liveForeignClaim(item, in.Actor, time.Now()); holder != "" {
		return nil, nil, refuse("item %q is claimed by %q until %s", item.Id, holder,
			item.GetDateTime("claim_expires").Time().Format(time.RFC3339))
	}
	// 4. The gate engine evaluates whatever the desk's config binds to (type, edge) — forward
	// edges by default, demote/reopen only when the config names them (§4.1 step 4).
	dc, derr := e.loadDeskConfig()
	if derr != nil {
		return nil, nil, derr
	}
	edgeKey := statemachine.EdgeKey(from, to)
	reqs := dc.rules.Effective(item.GetString("type"), edgeKey, e.fieldLookup(ctx, item))
	if gerr := gates.Evaluate(ctx, e.Validator, reqs, e.pointerResolver(item)); gerr != nil {
		var r *Refusal
		if errors.As(gerr, &r) {
			msg := fmt.Sprintf("cannot %s %s item to %s: %s",
				event, item.GetString("type"), to, strings.Join(r.Reasons, "; "))
			// A refusal is recorded as a gate_refused transitions row (§4.1) — observable audit,
			// deferred to the caller (written after the tx settles; the refusal still stands even
			// if that write later fails).
			return nil, &pendingAudit{
				itemID: item.Id, fromPhase: string(from), toPhase: string(to),
				event: "gate_refused", actor: in.Actor, detail: msg,
			}, &Refusal{Reasons: append([]string{}, r.Reasons...)}
		}
		return nil, nil, gerr
	}

	// 5. Success: write the new phase, keep the label mapped (§3.3), audit, cascade (§3.5). All
	// through this tx-scoped engine, so the phase write, the audit row, and the cascade's
	// side-writes commit or roll back together.
	item.Set("phase", string(to))
	if dc.labels[item.GetString("status_label")] != to {
		item.Set("status_label", statemachine.DefaultLabelFor(to))
	}
	bump(item)
	if err := e.App.Save(item); err != nil {
		return nil, nil, err
	}
	if err := runFailpoint(); err != nil {
		return nil, nil, err
	}
	if err := e.audit(item.Id, string(from), string(to), string(event), in.Actor, ""); err != nil {
		return nil, nil, err
	}
	if err := e.cascade(ctx, item, from, to, in.Actor); err != nil {
		return nil, nil, err
	}
	return item, nil, nil
}

// fieldLookup resolves a trait predicate field (§4.2): a first-class item field, then the
// properties overflow, then the pointed document's frontmatter through the seam's optional
// FrontmatterReader (never by reading librarian collections).
func (e *Engine) fieldLookup(ctx context.Context, item *core.Record) gates.FieldLookup {
	return func(field string) (string, bool) {
		for _, f := range []string{"title", "type", "phase", "status_label", "court", "pointer", "severity"} {
			if f == field {
				return item.GetString(field), true
			}
		}
		if props := item.GetString("properties"); props != "" {
			// properties is a JSON overflow; a flat string field in it can match.
			if v, ok := jsonStringField(props, field); ok {
				return v, true
			}
		}
		if reader, ok := e.Validator.(schema.FrontmatterReader); ok && item.GetString("pointer") != "" {
			if fm, err := reader.Frontmatter(ctx, item.GetString("pointer")); err == nil {
				if s, ok := fm[field].(string); ok {
					return s, true
				}
			}
		}
		return "", false
	}
}

// pointerResolver maps a gate rule's pointer spec to a document path (§4.2): "item" = the
// item's own pointer; "note:<key>" = the body of the item's note with that key.
func (e *Engine) pointerResolver(item *core.Record) func(spec string) (string, error) {
	return func(spec string) (string, error) {
		switch {
		case spec == "item":
			if p := item.GetString("pointer"); p != "" {
				return p, nil
			}
			return "", fmt.Errorf("item %q has no document pointer set", item.Id)
		case strings.HasPrefix(spec, "note:"):
			key := strings.TrimPrefix(spec, "note:")
			note, err := e.App.FindFirstRecordByFilter("notes",
				"item = {:item} && key = {:key}", map[string]any{"item": item.Id, "key": key})
			if err != nil {
				return "", fmt.Errorf("item %q has no note with key %q", item.Id, key)
			}
			if body := strings.TrimSpace(note.GetString("body")); body != "" {
				return body, nil
			}
			return "", fmt.Errorf("item %q note %q is empty", item.Id, key)
		}
		return "", fmt.Errorf("unknown pointer spec %q", spec)
	}
}

// --- blocked side-state (§3.2) ---

// Block sets the blocked flag, preserving the phase and recording restore_phase.
func (e *Engine) Block(ctx context.Context, itemID string, version int, actor Actor, reason string) (*core.Record, error) {
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		item, err := txe.loadItem(itemID)
		if err != nil {
			return err
		}
		if err := checkVersion(item, version); err != nil {
			return err
		}
		if statemachine.Phase(item.GetString("phase")) == statemachine.Terminal {
			return refuse("item %q is terminal; a terminal item cannot be blocked", item.Id)
		}
		if item.GetBool("blocked") {
			out = item // already blocked: idempotent
			return nil
		}
		txe.setBlocked(item, true)
		bump(item)
		if err := txe.App.Save(item); err != nil {
			return err
		}
		phase := item.GetString("phase")
		if err := txe.audit(item.Id, phase, phase, "block", actor, reason); err != nil {
			return err
		}
		out = item
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// Unblock clears the flag and restores the item to its recorded restore_phase.
func (e *Engine) Unblock(ctx context.Context, itemID string, version int, actor Actor, reason string) (*core.Record, error) {
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		item, err := txe.loadItem(itemID)
		if err != nil {
			return err
		}
		if err := checkVersion(item, version); err != nil {
			return err
		}
		if !item.GetBool("blocked") {
			out = item // not blocked: idempotent
			return nil
		}
		txe.setBlocked(item, false)
		bump(item)
		if err := txe.App.Save(item); err != nil {
			return err
		}
		phase := item.GetString("phase")
		if err := txe.audit(item.Id, phase, phase, "unblock", actor, reason); err != nil {
			return err
		}
		out = item
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// setBlocked flips the side-state in place (shared by Block/Unblock and the cascade, which
// carries its own audit rows + version bumps).
func (e *Engine) setBlocked(item *core.Record, blocked bool) {
	if blocked {
		item.Set("blocked", true)
		item.Set("restore_phase", item.GetString("phase"))
		return
	}
	item.Set("blocked", false)
	if rp := item.GetString("restore_phase"); rp != "" {
		item.Set("phase", rp) // §3.2: return the item to the phase it held (normally a no-op)
	}
	item.Set("restore_phase", "")
}

// --- claims (§3.6) ---

// Claim sets claimed_by + claim_expires (TTL from desk_config / PM_CLAIM_TTL / 30m default).
// A live foreign claim refuses; an expired one is free; re-claiming your own claim renews it.
func (e *Engine) Claim(ctx context.Context, itemID string, version int, actor Actor) (*core.Record, error) {
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		item, err := txe.loadItem(itemID)
		if err != nil {
			return err
		}
		if err := checkVersion(item, version); err != nil {
			return err
		}
		if holder := liveForeignClaim(item, actor, time.Now()); holder != "" {
			return refuse("item %q is already claimed by %q", item.Id, holder)
		}
		dc, derr := txe.loadDeskConfig()
		if derr != nil {
			return derr
		}
		item.Set("claimed_by", actor.Name)
		item.Set("claim_expires", time.Now().Add(dc.claimTTL))
		bump(item)
		if err := txe.App.Save(item); err != nil {
			return err
		}
		phase := item.GetString("phase")
		if err := txe.audit(item.Id, phase, phase, "claim", actor, ""); err != nil {
			return err
		}
		out = item
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// Release clears a claim. Only the holder may release a live claim; anyone may clear an
// expired one.
func (e *Engine) Release(ctx context.Context, itemID string, version int, actor Actor) (*core.Record, error) {
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		item, err := txe.loadItem(itemID)
		if err != nil {
			return err
		}
		if err := checkVersion(item, version); err != nil {
			return err
		}
		if holder := liveForeignClaim(item, actor, time.Now()); holder != "" {
			return refuse("item %q is claimed by %q; only the holder can release a live claim", item.Id, holder)
		}
		item.Set("claimed_by", "")
		item.Set("claim_expires", "")
		bump(item)
		if err := txe.App.Save(item); err != nil {
			return err
		}
		phase := item.GetString("phase")
		if err := txe.audit(item.Id, phase, phase, "release", actor, ""); err != nil {
			return err
		}
		out = item
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// --- dependency edges + cascade (§3.4, §3.5) ---

// LinkInput creates one typed dependency edge. Kind accepts the §3.4 surface vocabulary
// (blocks | is-blocked-by | relates-to); is-blocked-by is STORED as the inverse blocks edge
// (canonical direction: the blocker is `from`), so the graph has one representation.
type LinkInput struct {
	From      string
	To        string
	Kind      string
	UnblockAt string // gating edges: the blocker phase at which the block releases
	Cascade   string // auto | manual | auto-reopen | permanent
	Actor     Actor
}

// Link creates the edge and applies its initial block effect: a gating edge whose blocker has
// not yet reached unblock_at blocks the target (a permanent edge always blocks — it is a
// structural gate for the item's life).
func (e *Engine) Link(ctx context.Context, in LinkInput) (*core.Record, error) {
	fromID, toID, kind := in.From, in.To, in.Kind
	if kind == "is-blocked-by" {
		fromID, toID, kind = in.To, in.From, "blocks" // canonicalize (§3.4)
	}
	switch kind {
	case "blocks":
		if _, err := statemachine.ParsePhase(in.UnblockAt); err != nil || in.UnblockAt == string(statemachine.Queue) {
			return nil, refuse("a blocks edge needs unblock_at of work, review, or terminal")
		}
		switch in.Cascade {
		case "auto", "manual", "auto-reopen", "permanent":
		default:
			return nil, refuse("a blocks edge needs cascade of auto, manual, auto-reopen, or permanent")
		}
	case "relates-to":
		// non-gating informational link (§3.4); unblock_at/cascade are ignored
	default:
		return nil, refuse("unknown dependency kind %q (blocks, is-blocked-by, relates-to)", in.Kind)
	}
	if fromID == toID {
		return nil, refuse("an item cannot depend on itself")
	}

	// The endpoint reads, the edge write, and the target's initial-block write are one atomic
	// unit: the edge never lands referencing an item that a concurrent delete removed between the
	// read and the write, and a target is never left half-blocked (edge saved, block not).
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		blocker, err := txe.loadItem(fromID)
		if err != nil {
			return err
		}
		target, err := txe.loadItem(toID)
		if err != nil {
			return err
		}

		col, cerr := txe.App.FindCollectionByNameOrId("dependencies")
		if cerr != nil {
			return cerr
		}
		edge := core.NewRecord(col)
		edge.Set("from", blocker.Id)
		edge.Set("to", target.Id)
		edge.Set("kind", kind)
		edge.Set("desk", txe.desk())
		if kind == "blocks" {
			edge.Set("unblock_at", in.UnblockAt)
			edge.Set("cascade", in.Cascade)
		}
		if err := txe.App.Save(edge); err != nil {
			return err
		}

		if kind == "blocks" && !target.GetBool("blocked") &&
			statemachine.Phase(target.GetString("phase")) != statemachine.Terminal {
			unsatisfied := statemachine.Rank(statemachine.Phase(blocker.GetString("phase"))) <
				statemachine.Rank(statemachine.Phase(in.UnblockAt))
			if unsatisfied || in.Cascade == "permanent" {
				txe.setBlocked(target, true)
				bump(target)
				if err := txe.App.Save(target); err != nil {
					return err
				}
				phase := target.GetString("phase")
				detail := fmt.Sprintf("blocked by %q (unblock at %s, cascade %s)", blocker.Id, in.UnblockAt, in.Cascade)
				if err := txe.audit(target.Id, phase, phase, "block", in.Actor, detail); err != nil {
					return err
				}
			}
		}
		out = edge
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// cascade is the §3.5 scan a phase change on A drives: every outgoing gating edge applies its
// rule to its target B.
//
//   - auto:        A reaching unblock_at clears B (one-shot; later regression does not re-block)
//   - auto-reopen: like auto, and A regressing BELOW unblock_at re-blocks B (standing workstream)
//   - manual:      never auto-clears (a surface shows B as unblockable; a human/agent clears it)
//   - permanent:   never auto-clears; the edge must be deleted
//
// B auto-clears only when EVERY gating edge into it is satisfied and auto-clearable — a still-
// unsatisfied second blocker, or any manual/permanent edge, keeps B blocked.
func (e *Engine) cascade(ctx context.Context, blocker *core.Record, from, to statemachine.Phase, actor Actor) error {
	edges, err := e.App.FindRecordsByFilter("dependencies",
		"from = {:from} && kind = 'blocks'", "", 0, 0, map[string]any{"from": blocker.Id})
	if err != nil {
		return err
	}
	for _, edge := range edges {
		unblockAt := statemachine.Phase(edge.GetString("unblock_at"))
		reached := statemachine.Rank(to) >= statemachine.Rank(unblockAt)
		regressed := statemachine.Rank(from) >= statemachine.Rank(unblockAt) && !reached
		switch edge.GetString("cascade") {
		case "auto", "auto-reopen":
			if reached {
				if err := e.tryAutoUnblock(ctx, edge.GetString("to"), actor, blocker.Id); err != nil {
					return err
				}
			} else if regressed && edge.GetString("cascade") == "auto-reopen" {
				if err := e.reblock(ctx, edge.GetString("to"), actor, blocker.Id); err != nil {
					return err
				}
			}
		case "manual", "permanent":
			// manual: surfacing "unblockable" is a read-side concern (D4 get_context);
			// permanent: never auto-clears (§3.5).
		}
	}
	return nil
}

// tryAutoUnblock clears B's blocked flag iff every gating edge into B is satisfied AND
// auto-clearable (cascade auto/auto-reopen).
func (e *Engine) tryAutoUnblock(ctx context.Context, targetID string, actor Actor, causeID string) error {
	target, err := e.App.FindRecordById("items", targetID)
	if err != nil {
		return err
	}
	// Desk scope (defense in depth): Link desk-validates both ends, but an edge written
	// outside the engine must never let a cascade mutate another desk's item.
	if target.GetString("desk") != e.desk() || !target.GetBool("blocked") {
		return nil
	}
	incoming, err := e.App.FindRecordsByFilter("dependencies",
		"to = {:to} && kind = 'blocks'", "", 0, 0, map[string]any{"to": targetID})
	if err != nil {
		return err
	}
	for _, edge := range incoming {
		switch edge.GetString("cascade") {
		case "manual", "permanent":
			return nil // a manual/permanent gate keeps B blocked regardless (§3.5)
		}
		blocker, berr := e.App.FindRecordById("items", edge.GetString("from"))
		if berr != nil {
			return berr
		}
		if statemachine.Rank(statemachine.Phase(blocker.GetString("phase"))) <
			statemachine.Rank(statemachine.Phase(edge.GetString("unblock_at"))) {
			return nil // another blocker is still short of its release phase
		}
	}
	e.setBlocked(target, false)
	bump(target)
	if err := e.App.Save(target); err != nil {
		return err
	}
	phase := target.GetString("phase")
	return e.audit(target.Id, phase, phase, "unblock", actor,
		fmt.Sprintf("auto-unblocked: %q reached its release phase", causeID))
}

// reblock re-establishes B's blocked flag when an auto-reopen blocker regresses (§3.5).
func (e *Engine) reblock(ctx context.Context, targetID string, actor Actor, causeID string) error {
	target, err := e.App.FindRecordById("items", targetID)
	if err != nil {
		return err
	}
	// Desk scope: mirror tryAutoUnblock — never re-block a foreign desk's item.
	if target.GetString("desk") != e.desk() || target.GetBool("blocked") ||
		statemachine.Phase(target.GetString("phase")) == statemachine.Terminal {
		return nil
	}
	e.setBlocked(target, true)
	bump(target)
	if err := e.App.Save(target); err != nil {
		return err
	}
	phase := target.GetString("phase")
	return e.audit(target.Id, phase, phase, "block", actor,
		fmt.Sprintf("auto-re-blocked: %q regressed below its release phase (cascade auto-reopen)", causeID))
}

// --- status labels (§3.3) ---

// SetStatusLabel applies the friendly-vocabulary discipline: a label mapping to the item's
// current phase is a plain field write; a label mapping to a DIFFERENT phase is a transition
// request routed through the machine + gates (the label and the machine cannot drift); an
// unknown label is refused.
func (e *Engine) SetStatusLabel(ctx context.Context, itemID, label string, version int, actor Actor) (*core.Record, error) {
	var out *core.Record
	var pending *pendingAudit
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		item, err := txe.loadItem(itemID)
		if err != nil {
			return err
		}
		// This check serves the SAME-PHASE fast path below (plain label write, transitionCore
		// never runs) and fails fast before the desk-config load. On the cross-phase path
		// transitionCore re-checks the same tx-snapshot value — always agreeing, harmlessly.
		if err := checkVersion(item, version); err != nil {
			return err
		}
		dc, derr := txe.loadDeskConfig()
		if derr != nil {
			return derr
		}
		phase, known := dc.labels[label]
		if !known {
			return refuse("unknown status label %q for this desk", label)
		}
		if statemachine.Phase(item.GetString("phase")) == phase {
			item.Set("status_label", label)
			bump(item)
			if err := txe.App.Save(item); err != nil {
				return err
			}
			out = item
			return nil
		}
		// Cross-phase: route through the machine + gates via transitionCore in THIS transaction,
		// so the phase change and the label pin below commit or roll back as one unit.
		moved, p, terr := txe.transitionCore(ctx, TransitionInput{
			ItemID: itemID, TargetPhase: string(phase), Version: version, Actor: actor,
		})
		pending = p
		if terr != nil {
			return terr
		}
		// transitionCore already bumped the version and wrote the audit row; this save only pins
		// the exact requested label (transitionCore set the phase's default) — no second bump, so
		// the caller's version+1 expectation and the audit trail stay aligned.
		moved.Set("status_label", label)
		if err := txe.App.Save(moved); err != nil {
			return err
		}
		out = moved
		return nil
	})
	// A gate refusal from the cross-phase transition records its gate_refused row after the tx
	// settles (§4.1), same as the direct Transition path.
	if pending != nil {
		_ = e.audit(pending.itemID, pending.fromPhase, pending.toPhase, pending.event, pending.actor, pending.detail)
	}
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}

// jsonStringField extracts a flat string field from a JSON object string ("" , false when the
// JSON is invalid or the field is absent/non-string).
func jsonStringField(raw, field string) (string, bool) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", false
	}
	s, ok := m[field].(string)
	return s, ok
}

// AddNote attaches a phase-scoped keyed note (§3.7). A single insert would not strictly need a
// transaction, but the tx keeps the note's `phase` snapshot consistent with the item read (a
// non-tx read could record a stale phase if a transition lands in between) and keeps every
// mutating method on the same withApp(txApp) discipline (§3.6).
func (e *Engine) AddNote(ctx context.Context, itemID, key, body string, actor Actor) (*core.Record, error) {
	var out *core.Record
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		txe := e.withApp(txApp)
		item, err := txe.loadItem(itemID)
		if err != nil {
			return err
		}
		col, cerr := txe.App.FindCollectionByNameOrId("notes")
		if cerr != nil {
			return cerr
		}
		rec := core.NewRecord(col)
		rec.Set("item", item.Id)
		rec.Set("phase", item.GetString("phase"))
		rec.Set("key", key)
		rec.Set("body", body)
		rec.Set("actor", actor.Name)
		rec.Set("actor_kind", actor.Kind)
		if err := txe.App.Save(rec); err != nil {
			return err
		}
		out = rec
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return out, nil
}
