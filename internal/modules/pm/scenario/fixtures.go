package scenario

// Fixtures is the starter set of realistic desk workflows (foreman amendment), each exercising
// a distinct mechanism against the spec's real phases/edges/labels/courts. They are
// identity-neutral (R5.3): generic item titles, no real desk/person/repo. They run against the
// SHIPPED default gate rules (gates.DefaultRulesYAML): decision review→terminal needs an
// accepted decision document; task work→review needs an active task document.
//
//	a) decision-gate    — a decision item is refused entry to terminal while its decision doc is
//	                      missing, then while it is at the wrong status, and is admitted only once
//	                      the doc is accepted (the gate spine, §4).
//	b) auto-unblock     — a work item blocked by another auto-releases exactly when the blocker
//	                      reaches its unblock_at phase, not before (cascade, §3.5).
//	c) reopen           — an item walked into terminal can be reopened back to work (§3.2 reopen).
//	d) concurrency      — a live foreign claim blocks a foreign advance, and a stale version
//	                      token is refused (§3.6 optimistic concurrency + claim).
func Fixtures() []Scenario {
	false_ := false
	decPtr := "_structure/decisions/0021-example.md"

	return []Scenario{
		{
			Name: "decision-gate",
			Steps: []Step{
				{Name: "create the decision", Op: Create, Item: "dec", Title: "Rule the thing",
					Type: "decision", Court: "owner", Pointer: decPtr,
					Expect: Expect{Phase: "queue", StatusLabel: "backlog"}},
				{Name: "start work", Op: Transition, Item: "dec", To: "work",
					Expect: Expect{Phase: "work", StatusLabel: "active", AuditEvent: "advance"}},
				{Name: "move to review", Op: Transition, Item: "dec", To: "review",
					Expect: Expect{Phase: "review", StatusLabel: "in-review"}},
				{Name: "complete refused: doc absent", Op: Transition, Item: "dec", To: "terminal",
					Expect: Expect{Refused: true, RefusalContains: "does not exist", Phase: "review", AuditEvent: "gate_refused"}},
				{Name: "author the doc but at draft", Op: SetDoc, Pointer: decPtr, DocStatus: "draft", DocValid: true},
				{Name: "complete refused: wrong status", Op: Transition, Item: "dec", To: "terminal",
					Expect: Expect{Refused: true, RefusalContains: `needs "accepted"`, Phase: "review"}},
				{Name: "accept the doc", Op: SetDoc, Pointer: decPtr, DocStatus: "accepted", DocValid: true},
				{Name: "complete admitted", Op: Transition, Item: "dec", To: "terminal",
					Expect: Expect{Phase: "terminal", StatusLabel: "done", AuditEvent: "advance"}},
			},
		},
		{
			Name: "auto-unblock",
			Steps: []Step{
				{Name: "create the blocker", Op: Create, Item: "blocker", Title: "Finish the prerequisite",
					Type: "analysis", Court: "desk", Expect: Expect{Phase: "queue"}},
				{Name: "create the dependent", Op: Create, Item: "victim", Title: "Do the follow-on work",
					Type: "analysis", Court: "crew", Expect: Expect{Phase: "queue"}},
				{Name: "link blocker→dependent (auto, release at review)", Op: Link,
					From: "blocker", LinkTo: "victim", Kind: "blocks", UnblockAt: "review", Cascade: "auto",
					Expect: Expect{StillBlocked: []string{"victim"}}},
				{Name: "blocker to work: not yet released", Op: Transition, Item: "blocker", To: "work",
					Expect: Expect{Phase: "work", StillBlocked: []string{"victim"}}},
				{Name: "blocker to review: releases the dependent", Op: Transition, Item: "blocker", To: "review",
					Expect: Expect{Phase: "review", AutoUnblocked: []string{"victim"}}},
			},
		},
		{
			Name: "reopen",
			Steps: []Step{
				{Name: "create an item", Op: Create, Item: "t", Title: "A unit of work",
					Type: "analysis", Court: "desk", Expect: Expect{Phase: "queue"}},
				{Name: "start work", Op: Transition, Item: "t", To: "work", Expect: Expect{Phase: "work"}},
				{Name: "into review", Op: Transition, Item: "t", To: "review", Expect: Expect{Phase: "review"}},
				{Name: "complete", Op: Transition, Item: "t", To: "terminal",
					Expect: Expect{Phase: "terminal", StatusLabel: "done", AuditEvent: "advance"}},
				{Name: "reopen from terminal", Op: Transition, Item: "t", To: "work",
					Expect: Expect{Phase: "work", StatusLabel: "active", AuditEvent: "reopen"}},
			},
		},
		{
			Name: "concurrency",
			Steps: []Step{
				{Name: "create the contested item", Op: Create, Item: "c", Title: "A contested item",
					Type: "analysis", Court: "crew", Expect: Expect{Phase: "queue"}},
				{Name: "alice claims it", Op: Claim, Item: "c",
					Actor:  Actor{Name: "agent-alice", Kind: "agent"},
					Expect: Expect{AuditEvent: "claim"}},
				{Name: "bob's advance is blocked by the live claim", Op: Transition, Item: "c", To: "work",
					Actor:  Actor{Name: "agent-bob", Kind: "agent"},
					Expect: Expect{Refused: true, RefusalContains: "claimed by", Phase: "queue", Blocked: &false_}},
				{Name: "alice's stale-version advance is refused", Op: Transition, Item: "c", To: "work",
					Actor:        Actor{Name: "agent-alice", Kind: "agent"},
					StaleVersion: intp(1),
					Expect:       Expect{Refused: true, RefusalContains: "changed since you read it", Phase: "queue"}},
				{Name: "alice advances with the live token", Op: Transition, Item: "c", To: "work",
					Actor:  Actor{Name: "agent-alice", Kind: "agent"},
					Expect: Expect{Phase: "work", AuditEvent: "advance", Blocked: &false_}},
			},
		},
	}
}

func intp(n int) *int { return &n }
