---
type: technical-design
status: approved
created: 2026-05-12
updated: 2026-05-12
tags: [data, firestore, content]
author: robin
priority: P0
affects_workstreams: [engineering, product]
superseded_by:
---

# Firestore schema for content platform — multi-stage entry lifecycle

## Problem context

Quillpad ingests content (captured URLs, notes, papers, tools) into an inbox, then moves items through a development and polishing workflow before publication. The current schema conflates inbox triage with drafted content, making it hard to reason about which collections are ephemeral and which are canonical. As we add agent-driven enrichment (auto-classification, metadata enhancement, polish suggestions), we need clear ownership boundaries for collections and explicit audit trails. This TDD establishes the single source of truth for data shape across capture, triage, authoring, and publication phases.

Related product-specs: [[assistant-product]], [[capture-product]], [[authoring-product]].

## Goals & non-goals

### Goals

- Support three distinct lifecycle stages: capture (inbox), development (drafts), publication (entries) with clear transition semantics
- Enable agent operations (autonomous enrichment, suggestion proposals) while maintaining full audit trail of who/what changed each document
- Achieve sub-100ms read latency for feed, detail, and topic queries via composite indices
- Allow future migration to multi-author without schema redesign

### Non-goals

- We are not implementing full-text search shape here (Algolia is the indexing service; mirror contract is separate)
- We are not defining access control rules (those live in [[04-security-rules]])
- We are not specifying Cloud Function implementations (feature specs own those)
- We are not supporting granular field-level permissions in v1 (binary admin/viewer is sufficient)

## Proposed design

### High-level overview

Firestore is the primary persistent store. Ten collections organize content across three workflows: (1) `/inbox` for raw captured items, (2) `/drafts` for work-in-progress checkpoints, and (3) `/entries` for canonical published content. Each entry has a stable ID, slug, and status that gates visibility and mutation permissions.

Agent operations (classifiers, enrichers, polish-proposers) are asynchronous Cloud Functions that read from one collection and write proposals or mutations to audit collections. The `/agentOps` collection captures every autonomous action for reproducibility and rollback.

Data flows:
1. Capture: Item → `/inbox/{itemId}` (raw, agent-classifies immediately)
2. Triage: User or agent promotes inbox to `/entries` with status `draft`, status lifecycle begins
3. Development: Agent or user edits `/entries/{entryId}` in `developing` status, generating suggestions
4. Polish: Similar flow with `polishing` status; agent proposes final copy improvements
5. Publication: User moves to `published` status; entry is live to readers

### Data model

```
/entries/{entryId}
  ├─ id: UUID
  ├─ slug: string (unique)
  ├─ type: 'note' | 'tool' | 'paper' | 'person' | 'repo' | 'media'
  ├─ title, excerpt, body: strings
  ├─ status: 'draft' | 'developing' | 'polishing' | 'published' | 'archived'
  ├─ visibility: 'admin' | 'viewer' | 'public'
  ├─ topic: string (from canonical topics list)
  ├─ tags: string[]
  ├─ createdAt, updatedAt, publishedAt: Timestamp
  ├─ authorUid: string (always Robin in v1)
  └─ [type extensions: tool fields like repoUrl/stars, paper fields like doi/authors, etc.]

/inbox/{itemId}
  ├─ id: UUID
  ├─ type: 'note' | 'tool' | 'paper' | 'person' | 'repo' | 'media'
  ├─ rawContent: string (captured text / URL / metadata)
  ├─ sourceUrl?: string
  ├─ sourceKind: 'paste' | 'file-upload' | 'voice-memo' | 'web-bookmark'
  ├─ capturedAt: Timestamp
  ├─ classification: { type, confidence, proposedTopic }  (agent-filled)
  └─ promotedTo: entryId (set when user/agent promotes to draft)

/agentOps/{opId}
  ├─ id: UUID
  ├─ entryId: string (what did this touch)
  ├─ opKind: 'classify' | 'enrich' | 'suggest-polish' | 'generate-slug'
  ├─ status: 'proposed' | 'accepted' | 'rejected'
  ├─ before, after: JSON (document snapshots for diffs)
  ├─ generatedAt: Timestamp
  └─ acceptedAt?: Timestamp
```

### API & contracts

**Promote inbox item to draft entry**
- POST `/api/entries` with `{ inboxItemId, type, title, excerpt, topic, visibility }`
- Returns: `{ entryId, slug, status: "draft" }`

**Agent-propose Polish**
- POST `/api/agentOps` with `{ entryId, opKind: "suggest-polish", proposedChanges }`
- Returns: `{ opId, status: "proposed", before, after }`

**Accept agent proposal**
- POST `/api/agentOps/{opId}/accept`
- Returns: `{ status: "accepted", appliedAt }`

### Alternatives considered

**Option A: Single `/entries` collection for all stages**
- Trade-off: Simpler schema, but inbox clutter and status lifecycle becomes implicit
- Chosen: No

**Option B: Separate inbox + entries + drafts as above**
- Trade-off: Three collections, clearer concerns, triage is explicit
- Chosen: Yes

**Option C: Audit every field change in a separate collection**
- Trade-off: Full audit trail at storage cost; complex querying
- Chosen: No (cap audit to agent operations only in v1)

## Infrastructure & deployment

### Database

- Three main collections: `/inbox`, `/entries`, `/agentOps`
- Composite indices on `entries` for feed (status + publishedAt), topics (topic + status), search (slug contains)
- TTL policy on `/inbox`: documents auto-delete 30 days after promotion (ephemeral)
- TTL policy on `/agentOps`: documents auto-delete 90 days after acceptance (audit retention)

### Services

- Existing: Firestore (no new service)
- Existing: Cloud Functions for agents (no new service)
- New: Algolia index mirror on `/entries` (publish trigger → mirror)

### Configuration

No new environment variables. Firestore rules in [[04-security-rules]] gate access.

## Risks & mitigation

**Risk: Inbox cleanup TTL deletes items before user has triaged them**
→ Mitigation: TTL is 30 days post-promotion; if user hasn't promoted, TTL doesn't fire. Warn user if inbox is >14 days old.

**Risk: Agent proposal acceptance is lost if network fails mid-POST**
→ Mitigation: Acceptance is idempotent; opId is deterministic. Client can retry safely.

**Risk: Algolia mirror lags behind Firestore, causing stale search results**
→ Mitigation: Accept eventual consistency (documented); fallback to Firestore full-table scan if latency exceeds SLA.

## Open questions

- Should we version the schema? If so, do old entries keep old type extensions or migrate automatically? (Defer to schema evolution doc when needed)
- What is the TTL on successfully published entries? Never? (Assume never; archive on user request only)
- Should draft entries be visible to viewers? (Assume no; only published entries are visible to non-admin)
