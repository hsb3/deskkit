// The engine's read side (spec §5.1 read tools + §5.2 get_context): the SAME core functions
// every surface (MCP/CLI/TUI) calls, so §10.10 surface parity holds by construction — the
// surfaces are thin adapters over these, never re-implementations. All queries are desk-scoped
// (§3.1 desk field; R4.2 "all scoped to the desk").
package engine

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/schema"
)

// ItemSummary is the shared surface shape for one work item (§5.2 "item summary").
type ItemSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type,omitempty"`
	Phase       string `json:"phase"`
	StatusLabel string `json:"status_label,omitempty"`
	Blocked     bool   `json:"blocked,omitempty"`
	Court       string `json:"court,omitempty"`
	Pointer     string `json:"pointer,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Parent      string `json:"parent,omitempty"`
	Root        string `json:"root,omitempty"`
	ClaimedBy   string `json:"claimed_by,omitempty"`
	Version     int    `json:"version"`
}

// BlockedItem is one §5.2 blocked entry: the summary plus the resolved blocking state.
type BlockedItem struct {
	ItemSummary
	BlockedReason string   `json:"blocked_reason,omitempty"`
	BlockingItems []string `json:"blocking_items"`
}

// StalledItem is one §5.2 stalled entry.
type StalledItem struct {
	ItemSummary
	DaysSinceLastTransition int `json:"days_since_last_transition"`
}

// TransitionRow is one audit row for the surfaces (§5.2 recent_transitions; §3.6 fields).
type TransitionRow struct {
	Item             string `json:"item"`
	Event            string `json:"event"`
	From             string `json:"from,omitempty"`
	To               string `json:"to,omitempty"`
	Actor            string `json:"actor,omitempty"`
	ActorKind        string `json:"actor_kind,omitempty"`
	DelegationParent string `json:"delegation_parent,omitempty"`
	Detail           string `json:"detail,omitempty"`
	At               string `json:"at"`
}

// ContextResult is the §5.2 single-call cold-start briefing.
type ContextResult struct {
	Desk              string              `json:"desk"`
	GeneratedAt       string              `json:"generated_at"`
	Active            []ItemSummary       `json:"active"`
	Blocked           []BlockedItem       `json:"blocked"`
	Stalled           []StalledItem       `json:"stalled"`
	RecentTransitions []TransitionRow     `json:"recent_transitions"`
	Ancestors         map[string][]string `json:"ancestors"`
	Counts            ContextCounts       `json:"counts"`
}

// ContextCounts is the §5.2 counts block.
type ContextCounts struct {
	ByPhase map[string]int `json:"by_phase"`
	ByCourt map[string]int `json:"by_court"`
}

// recentTransitionsCap bounds the §5.2 recent_transitions set (newest first).
const recentTransitionsCap = 20

// Summarize maps an items record to the shared surface shape (exported for the pm tools'
// stable mutation-result payloads).
func Summarize(rec *core.Record) ItemSummary { return summarize(rec) }

// summarize maps an items record to the shared surface shape.
func summarize(rec *core.Record) ItemSummary {
	return ItemSummary{
		ID:          rec.Id,
		Title:       rec.GetString("title"),
		Type:        rec.GetString("type"),
		Phase:       rec.GetString("phase"),
		StatusLabel: rec.GetString("status_label"),
		Blocked:     rec.GetBool("blocked"),
		Court:       rec.GetString("court"),
		Pointer:     rec.GetString("pointer"),
		Severity:    rec.GetString("severity"),
		Priority:    rec.GetInt("priority"),
		Parent:      rec.GetString("parent"),
		Root:        rec.GetString("root"),
		ClaimedBy:   rec.GetString("claimed_by"),
		Version:     rec.GetInt("version"),
	}
}

// GetContext is THE cold-start briefing (§5.2, R4.2): one call returns the desk's working
// state. stalledDays <= 0 uses the configured default (PM_STALLED_DAYS, default 14).
func (e *Engine) GetContext(ctx context.Context, stalledDays int) (*ContextResult, error) {
	if stalledDays <= 0 {
		stalledDays = 14
		if e.Cfg != nil && e.Cfg.PMStalledDays > 0 {
			stalledDays = e.Cfg.PMStalledDays
		}
	}
	items, err := e.deskItems()
	if err != nil {
		return nil, err
	}

	res := &ContextResult{
		Desk:        e.desk(),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Active:      []ItemSummary{},
		Blocked:     []BlockedItem{},
		Stalled:     []StalledItem{},
		Ancestors:   map[string][]string{},
		Counts:      ContextCounts{ByPhase: map[string]int{}, ByCourt: map[string]int{}},
	}

	byID := map[string]*core.Record{}
	for _, it := range items {
		byID[it.Id] = it
	}
	lastTransition, recent, err := e.transitionIndex(byID)
	if err != nil {
		return nil, err
	}
	res.RecentTransitions = recent

	now := time.Now()
	var surfaced []*core.Record // active + blocked + stalled — the set the ancestors map covers
	for _, it := range items {
		phase := it.GetString("phase")
		res.Counts.ByPhase[phase]++
		if court := it.GetString("court"); court != "" {
			res.Counts.ByCourt[court]++
		}
		if phase == "terminal" {
			continue
		}
		if it.GetBool("blocked") {
			bi := BlockedItem{ItemSummary: summarize(it), BlockingItems: []string{}}
			bi.BlockingItems, bi.BlockedReason = e.blockingState(it.Id)
			res.Blocked = append(res.Blocked, bi)
			surfaced = append(surfaced, it)
			continue
		}
		res.Active = append(res.Active, summarize(it))
		surfaced = append(surfaced, it)
		// stalled = non-terminal whose last transitions row is older than the threshold (§5.2).
		// A blocked item is already surfaced as blocked; active items can ALSO be stalled.
		last := lastTransition[it.Id]
		if last.IsZero() {
			last = it.GetDateTime("created").Time()
		}
		if !last.IsZero() && now.Sub(last) > time.Duration(stalledDays)*24*time.Hour {
			res.Stalled = append(res.Stalled, StalledItem{
				ItemSummary:             summarize(it),
				DaysSinceLastTransition: int(now.Sub(last).Hours() / 24),
			})
		}
	}

	// active ordered by (court, priority) (§5.2).
	sort.SliceStable(res.Active, func(i, j int) bool {
		if res.Active[i].Court != res.Active[j].Court {
			return res.Active[i].Court < res.Active[j].Court
		}
		return res.Active[i].Priority < res.Active[j].Priority
	})

	for _, it := range surfaced {
		if chain := ancestorChain(it, byID); len(chain) > 0 {
			res.Ancestors[it.Id] = chain
		}
	}
	return res, nil
}

// deskItems fetches every item on this desk (small local store; one query).
func (e *Engine) deskItems() ([]*core.Record, error) {
	return e.App.FindRecordsByFilter("items", "desk = {:d}", "-created", 0, 0,
		map[string]any{"d": e.desk()})
}

// transitionIndex scans this desk's transitions newest-first and returns each item's last
// transition time plus the capped recent set (§5.2). transitions carries no desk column;
// desk scope is pushed into the QUERY via relation traversal (item.desk — the same filter
// language the API rules use), so other desks' rows are never loaded (review finding: the
// table is append-only and only ever grows). The byID membership check stays as cheap
// defense in depth.
func (e *Engine) transitionIndex(byID map[string]*core.Record) (map[string]time.Time, []TransitionRow, error) {
	recs, err := e.App.FindRecordsByFilter("transitions", "item.desk = {:d}", "-created", 0, 0,
		map[string]any{"d": e.desk()})
	if err != nil {
		return nil, nil, err
	}
	last := map[string]time.Time{}
	recent := []TransitionRow{}
	for _, tr := range recs {
		itemID := tr.GetString("item")
		if _, mine := byID[itemID]; !mine {
			continue
		}
		created := tr.GetDateTime("created").Time()
		if cur, ok := last[itemID]; !ok || created.After(cur) {
			last[itemID] = created
		}
		if len(recent) < recentTransitionsCap {
			recent = append(recent, TransitionRow{
				Item:             itemID,
				Event:            tr.GetString("event"),
				From:             tr.GetString("from_phase"),
				To:               tr.GetString("to_phase"),
				Actor:            tr.GetString("actor"),
				ActorKind:        tr.GetString("actor_kind"),
				DelegationParent: tr.GetString("delegation_parent"),
				Detail:           tr.GetString("detail"),
				At:               created.UTC().Format(time.RFC3339),
			})
		}
	}
	return last, recent, nil
}

// blockingState resolves a blocked item's §5.2 fields: the blocking items via the dependency
// graph (incoming gating edges whose rule still holds it), and the blocked_reason from the
// latest block audit row's detail.
func (e *Engine) blockingState(itemID string) ([]string, string) {
	blocking := []string{}
	edges, err := e.App.FindRecordsByFilter("dependencies",
		"to = {:to} && kind = 'blocks'", "", 0, 0, map[string]any{"to": itemID})
	if err != nil {
		// Log rather than silently return empty (review finding): a failed query would
		// otherwise read as "no blockers" on a blocked item — misleading surfaced data.
		e.App.Logger().Error("pm: blockingState load dependencies", "item", itemID, "err", err)
	}
	if err == nil {
		for _, edge := range edges {
			holds := false
			switch edge.GetString("cascade") {
			case "permanent":
				holds = true
			case "manual":
				holds = true // manual never auto-clears; it holds until explicitly released (§3.5)
			default: // auto / auto-reopen: holds while the blocker is short of unblock_at
				if blocker, berr := e.App.FindRecordById("items", edge.GetString("from")); berr == nil {
					holds = rankOf(blocker.GetString("phase")) < rankOf(edge.GetString("unblock_at"))
				}
			}
			if holds {
				blocking = append(blocking, edge.GetString("from"))
			}
		}
	}
	// blocked_reason = the newest block audit row's detail (§3.6 detail carries the why).
	reason := ""
	if rows, lerr := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'block'", "-created", 1, 0, map[string]any{"i": itemID}); lerr == nil && len(rows) > 0 {
		reason = rows[0].GetString("detail")
	}
	sort.Strings(blocking)
	return blocking, reason
}

// ancestorChain returns the root..parent id chain for an item (§5.2 ancestors). A broken or
// cyclic parent link terminates the walk rather than looping.
func ancestorChain(it *core.Record, byID map[string]*core.Record) []string {
	var reversed []string
	seen := map[string]bool{it.Id: true}
	cur := it.GetString("parent")
	for cur != "" && !seen[cur] {
		seen[cur] = true
		reversed = append(reversed, cur)
		parent, ok := byID[cur]
		if !ok {
			break
		}
		cur = parent.GetString("parent")
	}
	// reversed is parent..root; the spec's shape is root..parent.
	chain := make([]string, len(reversed))
	for i, id := range reversed {
		chain[len(reversed)-1-i] = id
	}
	return chain
}

// ListFilter is the §5.1 list_items filter set (by phase, court, type, blocked, parent).
type ListFilter struct {
	Phase   string
	Court   string
	Type    string
	Blocked string // "" = any; "true"/"false" filter the blocked flag
	Parent  string
}

// ListItems is the filtered graph query (§5.1), ordered by (phase rank, court, priority).
func (e *Engine) ListItems(ctx context.Context, f ListFilter) ([]ItemSummary, error) {
	items, err := e.deskItems()
	if err != nil {
		return nil, err
	}
	out := []ItemSummary{}
	for _, it := range items {
		if f.Phase != "" && it.GetString("phase") != f.Phase {
			continue
		}
		if f.Court != "" && it.GetString("court") != f.Court {
			continue
		}
		if f.Type != "" && it.GetString("type") != f.Type {
			continue
		}
		if f.Blocked == "true" && !it.GetBool("blocked") {
			continue
		}
		if f.Blocked == "false" && it.GetBool("blocked") {
			continue
		}
		if f.Parent != "" && it.GetString("parent") != f.Parent {
			continue
		}
		out = append(out, summarize(it))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if a, b := rankOf(out[i].Phase), rankOf(out[j].Phase); a != b {
			return a < b
		}
		if out[i].Court != out[j].Court {
			return out[i].Court < out[j].Court
		}
		return out[i].Priority < out[j].Priority
	})
	return out, nil
}

// NoteRow is one §3.7 note for the surfaces.
type NoteRow struct {
	Phase     string `json:"phase,omitempty"`
	Key       string `json:"key"`
	Body      string `json:"body"`
	Actor     string `json:"actor,omitempty"`
	ActorKind string `json:"actor_kind,omitempty"`
	At        string `json:"at"`
}

// DependencyRow is one §3.4 edge for the surfaces (canonical direction: from blocks to).
type DependencyRow struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Kind      string `json:"kind"`
	UnblockAt string `json:"unblock_at,omitempty"`
	Cascade   string `json:"cascade,omitempty"`
}

// ItemDetail is the §5.1 get_item shape: one item + its notes, deps, recent transitions,
// ancestor chain.
type ItemDetail struct {
	ItemSummary
	Body              string          `json:"body,omitempty"`
	Properties        json.RawMessage `json:"properties,omitempty"`
	Notes             []NoteRow       `json:"notes"`
	Dependencies      []DependencyRow `json:"dependencies"`
	RecentTransitions []TransitionRow `json:"recent_transitions"`
	Ancestors         []string        `json:"ancestors"`
}

// GetItem returns one item with its graph context (§5.1).
func (e *Engine) GetItem(ctx context.Context, itemID string) (*ItemDetail, error) {
	item, err := e.loadItem(itemID)
	if err != nil {
		return nil, err
	}
	d := &ItemDetail{
		ItemSummary:       summarize(item),
		Body:              item.GetString("body"),
		Notes:             []NoteRow{},
		Dependencies:      []DependencyRow{},
		RecentTransitions: []TransitionRow{},
		Ancestors:         []string{},
	}
	if props := strings.TrimSpace(item.GetString("properties")); props != "" && props != "null" {
		d.Properties = json.RawMessage(props)
	}

	notes, err := e.App.FindRecordsByFilter("notes", "item = {:i}", "created", 0, 0,
		map[string]any{"i": item.Id})
	if err == nil {
		for _, n := range notes {
			d.Notes = append(d.Notes, NoteRow{
				Phase:     n.GetString("phase"),
				Key:       n.GetString("key"),
				Body:      n.GetString("body"),
				Actor:     n.GetString("actor"),
				ActorKind: n.GetString("actor_kind"),
				At:        n.GetDateTime("created").Time().UTC().Format(time.RFC3339),
			})
		}
	}

	// Both directions of the graph (§3.4: one stored representation, both presented).
	edges, err := e.App.FindRecordsByFilter("dependencies",
		"from = {:id} || to = {:id}", "", 0, 0, map[string]any{"id": item.Id})
	if err == nil {
		for _, edge := range edges {
			d.Dependencies = append(d.Dependencies, DependencyRow{
				From:      edge.GetString("from"),
				To:        edge.GetString("to"),
				Kind:      edge.GetString("kind"),
				UnblockAt: edge.GetString("unblock_at"),
				Cascade:   edge.GetString("cascade"),
			})
		}
	}

	trs, err := e.App.FindRecordsByFilter("transitions", "item = {:i}", "-created",
		recentTransitionsCap, 0, map[string]any{"i": item.Id})
	if err == nil {
		for _, tr := range trs {
			d.RecentTransitions = append(d.RecentTransitions, TransitionRow{
				Item:             tr.GetString("item"),
				Event:            tr.GetString("event"),
				From:             tr.GetString("from_phase"),
				To:               tr.GetString("to_phase"),
				Actor:            tr.GetString("actor"),
				ActorKind:        tr.GetString("actor_kind"),
				DelegationParent: tr.GetString("delegation_parent"),
				Detail:           tr.GetString("detail"),
				At:               tr.GetDateTime("created").Time().UTC().Format(time.RFC3339),
			})
		}
	}

	d.Ancestors = e.ancestorChainByHop(item)
	return d, nil
}

// ancestorChainByHop walks the parent chain with one targeted read per hop (review finding:
// chains are a few items deep, so per-hop lookups beat loading the whole desk on every
// get_item). Cycle/broken links terminate the walk; returns root..parent order.
func (e *Engine) ancestorChainByHop(item *core.Record) []string {
	var reversed []string
	seen := map[string]bool{item.Id: true}
	cur := item.GetString("parent")
	for cur != "" && !seen[cur] {
		seen[cur] = true
		reversed = append(reversed, cur)
		parent, err := e.App.FindRecordById("items", cur)
		if err != nil {
			break
		}
		cur = parent.GetString("parent")
	}
	chain := make([]string, len(reversed))
	for i, id := range reversed {
		chain[len(reversed)-1-i] = id
	}
	return chain
}

// UpdateItemInput edits an item's first-class fields (§5.1 update_item; version-checked,
// R2.6). Only the Set* flags' fields are written, so an empty value can be set deliberately.
// phase and blocked are NOT editable here — the machine owns them (transition/block tools);
// a status_label change routes through SetStatusLabel (§3.3: a label mapping to a different
// phase is a transition request).
type UpdateItemInput struct {
	ItemID      string
	Version     int
	Title       *string
	Type        *string
	Court       *string
	Pointer     *string
	Body        *string
	Severity    *string
	Priority    *int
	Properties  *string
	StatusLabel *string
	Actor       Actor
}

// UpdateItem applies the field edits (§5.1). The version token is checked once up front; a
// status_label change that implies a phase change routes through the machine + gates.
func (e *Engine) UpdateItem(ctx context.Context, in UpdateItemInput) (*core.Record, error) {
	item, err := e.loadItem(in.ItemID)
	if err != nil {
		return nil, err
	}
	if err := checkVersion(item, in.Version); err != nil {
		return nil, err
	}
	// A live foreign claim is authoritative over every direct mutation (ADR 0020). This check
	// sits up front so it covers BOTH the field-edit path below AND the status_label path that
	// delegates to SetStatusLabel, and refuses a non-holder naming the holder and the expiry in
	// the same shape the transition path uses.
	if holder := liveForeignClaim(item, in.Actor, time.Now()); holder != "" {
		return nil, refuse("item %q is claimed by %q until %s", item.Id, holder,
			item.GetDateTime("claim_expires").Time().Format(time.RFC3339))
	}
	if in.Severity != nil && *in.Severity != "" {
		switch *in.Severity {
		case "low", "medium", "high":
		default:
			return nil, refuse("unknown severity %q (low, medium, high)", *in.Severity)
		}
	}
	if in.Court != nil && *in.Court != "" {
		switch *in.Court {
		case "owner", "desk", "crew", "vendor", "external-session":
		default:
			return nil, refuse("unknown court %q (owner, desk, crew, vendor, external-session)", *in.Court)
		}
	}
	// update_item must reject an unknown items.type with the exact same
	// schema-v1 vocabulary check create_item already applies (engine.go CreateItem) — before
	// this fix, update_item was the one write path that let a caller silently move a gated
	// item onto a bogus type string, and gates bind on this same field (engine.go
	// transitionCore's dc.rules.Effective(item.GetString("type"), ...) below). Clearing the
	// type ("") stays legal, mirroring create_item's "absent type stays legal" scope call
	// (ADR 0012); only a non-empty, unrecognized type is refused.
	if in.Type != nil && *in.Type != "" {
		vocab, verr := schema.Vocab()
		if verr != nil {
			return nil, verr
		}
		if !vocab.KnownType(*in.Type) {
			return nil, refuse("unknown item type %q (known types: %s; see schema/doctypes.yaml)",
				*in.Type, strings.Join(vocab.TypeNames(), ", "))
		}
	}
	if in.Title != nil {
		if strings.TrimSpace(*in.Title) == "" {
			return nil, refuse("an item's title cannot be emptied")
		}
		item.Set("title", *in.Title)
	}
	if in.Type != nil {
		item.Set("type", *in.Type)
	}
	if in.Court != nil {
		item.Set("court", *in.Court)
	}
	if in.Pointer != nil {
		item.Set("pointer", *in.Pointer)
	}
	if in.Body != nil {
		// A non-nil pointer to an empty string is a deliberate clear (mirrors the other *string
		// fields), so the body can be emptied on purpose.
		item.Set("body", *in.Body)
	}
	if in.Severity != nil {
		item.Set("severity", *in.Severity)
	}
	if in.Priority != nil {
		item.Set("priority", *in.Priority)
	}
	if in.Properties != nil {
		if *in.Properties != "" && !json.Valid([]byte(*in.Properties)) {
			return nil, refuse("properties must be a valid JSON object")
		}
		item.Set("properties", *in.Properties)
	}
	bump(item)
	if err := e.App.Save(item); err != nil {
		return nil, err
	}
	// A status_label change routes through the machine (§3.3) AFTER the field writes, using
	// the post-save version so the two-step edit reads as one caller intent.
	if in.StatusLabel != nil {
		return e.SetStatusLabel(ctx, item.Id, *in.StatusLabel, item.GetInt("version"), in.Actor)
	}
	return item, nil
}

// rankOf mirrors statemachine.Rank over raw strings for query-side ordering; unknown phases
// sort last.
func rankOf(phase string) int {
	switch phase {
	case "queue":
		return 0
	case "work":
		return 1
	case "review":
		return 2
	case "terminal":
		return 3
	}
	return 4
}
