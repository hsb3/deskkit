// Package importer is the minimal, deterministic import seam (spec §8.1 / §8.2): it turns a
// manifest of work items + dependency edges into a desk-scoped graph by driving the SAME
// engine every surface uses — never a second write path. It exists for two consumers:
//
//   - test lane §10.8 (rebuild reproducibility): fresh store → migrate → import → identical
//     graph; a second rebuild into another fresh store is byte-identical. This package supplies
//     the "import" step and the GraphSnapshot oracle the §10.8 test compares.
//   - D8 adoption (spec §8.1): the one-time seed that populates a scratch store from the desk's
//     existing work surfaces. D6 builds ONLY the seam §10.8 needs — the real desk manifest and
//     any document-driven derivation of it are D8's (this package takes a manifest as given).
//
// Determinism (§8.2 "stable ids, deterministic"): each item's record id is derived from
// (desk, manifest key), so importing the same manifest into two fresh stores yields identical
// ids and therefore an identical graph — parent/root/edge relations included. Import is
// idempotent and desk-scoped: re-running it against a store that already holds an item (by its
// deterministic id) skips it rather than duplicating.
//
// The importer is a THIN driver: every mutation goes through engine.CreateItem / engine.Link.
// It owns no transition, gate, or cascade logic (those stay in the engine, spec §4.1/§3.5).
package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hsb3/deskkit/internal/modules/pm/engine"
)

// ManifestItem is one work item to import. Key is the manifest-local stable identity (the
// slug the desk's documents/threads carry); the record id is derived from it. Parent, when
// set, names another item's Key (a parent must be present in the same manifest).
type ManifestItem struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Type     string `json:"type,omitempty"`
	Court    string `json:"court,omitempty"`
	Pointer  string `json:"pointer,omitempty"`
	Body     string `json:"body,omitempty"` // long-form body: narrative, acceptance criteria, or spec (§3.1)
	Severity string `json:"severity,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Parent   string `json:"parent,omitempty"` // parent item's Key; "" = a root
}

// ManifestDep is one dependency edge, both ends named by manifest Key. Kind/UnblockAt/Cascade
// carry the §3.4 surface vocabulary the engine's Link validates (is-blocked-by is
// canonicalized by the engine).
type ManifestDep struct {
	From      string `json:"from"` // blocker Key (for blocks) or source Key (relates-to)
	To        string `json:"to"`   // target Key
	Kind      string `json:"kind"`
	UnblockAt string `json:"unblock_at,omitempty"`
	Cascade   string `json:"cascade,omitempty"`
}

// Manifest is the whole import payload: items first, then the edges between them.
type Manifest struct {
	Items []ManifestItem `json:"items"`
	Deps  []ManifestDep  `json:"deps,omitempty"`
}

// Result reports what an import did (idempotent counts) and the key→id map D8 uses to build a
// scenario over the imported items.
type Result struct {
	IDs          map[string]string `json:"ids"` // manifest Key → record id
	CreatedItems int               `json:"created_items"`
	SkippedItems int               `json:"skipped_items"` // already present (idempotent re-import)
	CreatedDeps  int               `json:"created_deps"`
	SkippedDeps  int               `json:"skipped_deps"`
}

// ItemID derives a deterministic record id for (desk, key). The id is 15 lowercase hex chars,
// which satisfies PocketBase's record-id constraint (Min=Max=15, pattern ^[a-z0-9]+$), so two
// imports of the same manifest into fresh stores produce identical ids (§8.2). Desk-scoped so
// the same key on two desks never collides.
func ItemID(desk, key string) string {
	sum := sha256.Sum256([]byte(desk + "\x00" + key))
	return hex.EncodeToString(sum[:])[:15]
}

// Import writes the manifest into the engine's desk, idempotently and deterministically.
// Items are created parents-first (a self-referencing graph, §3.1); dependency edges follow.
// It never mutates a live desk on its own — the caller points the engine at the store it
// intends to write (D8 uses a scratch store; §8.1 step 5).
func Import(ctx context.Context, eng *engine.Engine, m Manifest) (*Result, error) {
	if eng == nil || eng.Cfg == nil {
		return nil, fmt.Errorf("importer: engine and its config are required")
	}
	desk := eng.Cfg.DeskName

	// Validate the manifest up front so a bad reference fails loudly before any write.
	keyIndex := make(map[string]ManifestItem, len(m.Items))
	for _, it := range m.Items {
		if it.Key == "" {
			return nil, fmt.Errorf("importer: an item has an empty key")
		}
		if _, dup := keyIndex[it.Key]; dup {
			return nil, fmt.Errorf("importer: duplicate item key %q", it.Key)
		}
		keyIndex[it.Key] = it
	}
	for _, it := range m.Items {
		if it.Parent != "" {
			if _, ok := keyIndex[it.Parent]; !ok {
				return nil, fmt.Errorf("importer: item %q names unknown parent %q", it.Key, it.Parent)
			}
		}
	}
	for _, d := range m.Deps {
		if _, ok := keyIndex[d.From]; !ok {
			return nil, fmt.Errorf("importer: dependency names unknown from %q", d.From)
		}
		if _, ok := keyIndex[d.To]; !ok {
			return nil, fmt.Errorf("importer: dependency names unknown to %q", d.To)
		}
	}

	res := &Result{IDs: make(map[string]string, len(m.Items))}
	for _, it := range m.Items {
		res.IDs[it.Key] = ItemID(desk, it.Key)
	}

	// Create items parents-first. A deterministic pass order (sorted keys within each wave)
	// keeps the create sequence — and so any incidental ordering the store records — stable
	// across rebuilds, independent of the manifest's own ordering.
	created := make(map[string]bool, len(m.Items))
	pending := make([]string, 0, len(m.Items))
	for k := range keyIndex {
		pending = append(pending, k)
	}
	sort.Strings(pending)

	for len(pending) > 0 {
		progressed := false
		next := pending[:0]
		for _, key := range pending {
			it := keyIndex[key]
			if it.Parent != "" && !created[it.Parent] {
				next = append(next, key) // parent not created yet; defer to a later wave
				continue
			}
			skipped, err := e_createItem(ctx, eng, desk, it, res.IDs)
			if err != nil {
				return nil, err
			}
			if skipped {
				res.SkippedItems++
			} else {
				res.CreatedItems++
			}
			created[key] = true
			progressed = true
		}
		pending = append([]string(nil), next...)
		if !progressed && len(pending) > 0 {
			return nil, fmt.Errorf("importer: unresolvable parent cycle among %v", pending)
		}
	}

	for _, d := range m.Deps {
		skipped, err := e_createDep(ctx, eng, d, res.IDs)
		if err != nil {
			return nil, err
		}
		if skipped {
			res.SkippedDeps++
		} else {
			res.CreatedDeps++
		}
	}
	return res, nil
}

// e_createItem creates one item at its deterministic id, or reports skipped=true when a record
// with that id already exists on the desk (idempotent re-import).
func e_createItem(ctx context.Context, eng *engine.Engine, desk string, it ManifestItem, ids map[string]string) (skipped bool, err error) {
	id := ids[it.Key]
	if existing, ferr := eng.App.FindRecordById("items", id); ferr == nil && existing != nil {
		if existing.GetString("desk") != desk {
			return false, fmt.Errorf("importer: id %q for key %q already belongs to desk %q",
				id, it.Key, existing.GetString("desk"))
		}
		return true, nil
	}
	parentID := ""
	if it.Parent != "" {
		parentID = ids[it.Parent]
	}
	if _, err := eng.CreateItem(ctx, engine.CreateItemInput{
		ID: id, Title: it.Title, Type: it.Type, Parent: parentID, Court: it.Court,
		Pointer: it.Pointer, Body: it.Body, Severity: it.Severity, Priority: it.Priority,
		Actor: engine.Actor{Name: "import", Kind: "agent"},
	}); err != nil {
		return false, fmt.Errorf("importer: create item %q: %w", it.Key, err)
	}
	return false, nil
}

// e_createDep creates one dependency edge, or reports skipped=true when a canonically-equal
// edge already exists (idempotent re-import). Canonicalization mirrors the engine's Link
// (is-blocked-by stores as the inverse blocks edge) so the existence check matches what Link
// would write.
func e_createDep(ctx context.Context, eng *engine.Engine, d ManifestDep, ids map[string]string) (skipped bool, err error) {
	fromID, toID, kind := ids[d.From], ids[d.To], d.Kind
	if kind == "is-blocked-by" {
		fromID, toID, kind = ids[d.To], ids[d.From], "blocks"
	}
	existing, ferr := eng.App.FindRecordsByFilter("dependencies",
		"from = {:f} && to = {:t} && kind = {:k}", "", 1, 0,
		map[string]any{"f": fromID, "t": toID, "k": kind})
	if ferr != nil {
		return false, fmt.Errorf("importer: check dep existence %s->%s: %w", d.From, d.To, ferr)
	}
	if len(existing) > 0 {
		return true, nil
	}
	if _, err := eng.Link(ctx, engine.LinkInput{
		From: ids[d.From], To: ids[d.To], Kind: d.Kind, UnblockAt: d.UnblockAt, Cascade: d.Cascade,
		Actor: engine.Actor{Name: "import", Kind: "agent"},
	}); err != nil {
		return false, fmt.Errorf("importer: link %s->%s: %w", d.From, d.To, err)
	}
	return false, nil
}

// --- reproducibility oracle (§10.8) ---

// ItemProjection is the deterministic, comparison-friendly shape of one imported item: every
// first-class field that the import writes, id included. Timestamps are excluded (they are the
// one non-deterministic column and are not part of the graph's identity).
type ItemProjection struct {
	ID          string `json:"id"`
	Desk        string `json:"desk"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Phase       string `json:"phase"`
	StatusLabel string `json:"status_label"`
	Blocked     bool   `json:"blocked"`
	Court       string `json:"court"`
	Pointer     string `json:"pointer"`
	Body        string `json:"body"`
	Severity    string `json:"severity"`
	Priority    int    `json:"priority"`
	Parent      string `json:"parent"`
	Root        string `json:"root"`
	Version     int    `json:"version"`
}

// DepProjection is the deterministic shape of one dependency edge.
type DepProjection struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Kind      string `json:"kind"`
	UnblockAt string `json:"unblock_at"`
	Cascade   string `json:"cascade"`
}

// Snapshot is a canonical, order-stable projection of the desk's imported graph (§8.2). Two
// stores built from the same manifest have equal Snapshots.
type Snapshot struct {
	Desk  string           `json:"desk"`
	Items []ItemProjection `json:"items"`
	Deps  []DepProjection  `json:"deps"`
}

// GraphSnapshot reads the desk's items + dependencies into a canonical Snapshot, sorted by id
// so the result is deterministic regardless of store iteration order.
func GraphSnapshot(ctx context.Context, eng *engine.Engine) (Snapshot, error) {
	desk := eng.Cfg.DeskName
	snap := Snapshot{Desk: desk, Items: []ItemProjection{}, Deps: []DepProjection{}}

	items, err := eng.App.FindRecordsByFilter("items", "desk = {:d}", "id", 0, 0,
		map[string]any{"d": desk})
	if err != nil {
		return snap, err
	}
	for _, it := range items {
		snap.Items = append(snap.Items, ItemProjection{
			ID: it.Id, Desk: it.GetString("desk"), Title: it.GetString("title"),
			Type: it.GetString("type"), Phase: it.GetString("phase"),
			StatusLabel: it.GetString("status_label"), Blocked: it.GetBool("blocked"),
			Court: it.GetString("court"), Pointer: it.GetString("pointer"),
			Body:     it.GetString("body"),
			Severity: it.GetString("severity"), Priority: it.GetInt("priority"),
			Parent: it.GetString("parent"), Root: it.GetString("root"),
			Version: it.GetInt("version"),
		})
	}
	sort.Slice(snap.Items, func(i, j int) bool { return snap.Items[i].ID < snap.Items[j].ID })

	deps, err := eng.App.FindRecordsByFilter("dependencies", "desk = {:d}", "", 0, 0,
		map[string]any{"d": desk})
	if err != nil {
		return snap, err
	}
	for _, d := range deps {
		snap.Deps = append(snap.Deps, DepProjection{
			From: d.GetString("from"), To: d.GetString("to"), Kind: d.GetString("kind"),
			UnblockAt: d.GetString("unblock_at"), Cascade: d.GetString("cascade"),
		})
	}
	sort.Slice(snap.Deps, func(i, j int) bool {
		if snap.Deps[i].From != snap.Deps[j].From {
			return snap.Deps[i].From < snap.Deps[j].From
		}
		if snap.Deps[i].To != snap.Deps[j].To {
			return snap.Deps[i].To < snap.Deps[j].To
		}
		// (From,To) can repeat: e_createDep dedups by the (from,to,kind) triple, so two edges
		// with the same endpoints but different Kind legitimately coexist. Kind completes the
		// identity and is the deterministic tiebreaker — without it their relative order is
		// undefined and Canonical() becomes non-deterministic (§8.2).
		return snap.Deps[i].Kind < snap.Deps[j].Kind
	})
	return snap, nil
}

// Canonical renders a Snapshot as stable JSON — the exact bytes two rebuilds must match. A
// marshal failure panics rather than returning "": two empty strings would compare equal and
// turn the §10.8 reproducibility assertion into a silent false-pass. This is a test oracle, so
// a panic is the right failure mode.
func (s Snapshot) Canonical() string {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic("importer: Canonical: " + err.Error())
	}
	return string(b)
}
