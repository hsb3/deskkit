---
type: decision
status: accepted
created: 2026-05-12
updated: 2026-05-12
tags: [quillpad, tech, infrastructure]
decided_by: [robin]
superseded_by:
affects_workstreams: [engineering, product]
---

# Choose Firebase over Lexington/Keystatic + Better Auth for Quillpad v1

## Context

Quillpad consolidates two stalled projects: `agent_quillpad` (Astro + Firebase, production-ready infrastructure) and `capture_kit` (Python ingestion pipeline, local-only). We need a single unified stack for the merged product.

Robin's original constraint: "I'm not trying to make a product. I want to share some thoughts with friends." This rules out unnecessary vendor spread and replatforming complexity.

We evaluated three coherent stacks. All could work; the choice is about risk vs. novelty.

## Options Considered

### Option A: Firebase (Firestore + Auth + Functions + Hosting)

Reuse infrastructure from `agent_quillpad`. Single cloud vendor, unified billing, familiar Admin SDK patterns. Supports dynamic content (Firestore), serverless compute (Functions for Genkit flows), and hybrid SSG + React islands (Astro integration solid).

**Pros:**
- Already running production in `agent_quillpad`; patterns proven
- Single cloud vendor → one login, one billing portal, no vendor spread
- Firestore security rules + Firebase Auth handle all auth/data gating
- Firebase Functions native to Genkit (Google's own framework)
- Hosting includes automatic HTTPS, CDN, rollback
- Cost predictable; Robin pays for what he uses (generous free tier covers v1 usage)

**Cons:**
- Google lock-in; leaving Firebase later requires data migration + rewrite
- No git-native content model; everything lives in Firestore (not versionable in Git)
- Firestore pricing scales with reads/writes; chatty clients can get expensive
- No built-in backup export tool; requires custom Cloud Functions for snapshots
- Security rules have a learning curve; misconfigured rules are a common failure mode

### Option B: Lexington Hemingway + Keystatic + Better Auth

Git-native content workflow. Content lives in Git; Keystatic's TypeScript SDK syncs structured content to a database. Better Auth provides passwordless, multi-factor auth. Astro integration solid.

**Pros:**
- Content versioned in Git; full history, blame, easy diffs
- Keystatic's structured sync is elegant; content shape validated on write
- Better Auth is modern, self-hosted, zero vendor lock-in for auth
- Astro integration excellent (Keystatic is built for it)
- Content is human-readable Markdown in Git; easy to copy/paste, fork, redistribute

**Cons:**
- Three vendors: GitHub (hosting content + CI), Keystatic (content sync middleware), Better Auth (self-hosted identity)
- Better Auth requires a separate deployable service; adds ops burden for a solo author
- Keystatic and Better Auth are smaller, less battle-tested ecosystems; smaller communities
- Self-hosting Better Auth means Robin is responsible for uptime, secrets rotation, schema migrations
- Data in Keystatic still lives in a database (not Git-only); defeats the "versionable" claim partially
- Git-native workflow is *better for teams*; for solo author, the versioning benefit is marginal (drafts in localStorage work fine)

### Option C: Payload CMS 3 + Astro

Payload's admin UI is best-in-class; structured content model with custom fields. Payload provides auth, roles, content versioning.

**Pros:**
- Admin UI is genuinely polished; a joy to write in
- Structured content schema validated at write time
- Built-in version history per document
- Granular permissions system (who can edit what)
- Excellent Astro integration docs

**Cons:**
- Requires separate Node.js backend + database (separate from Astro static site)
- Two deployables: Astro frontend + Payload backend
- Adds complexity: Robin pays for two services, manages two deploys, two databases
- Overkill for a personal publishing site; the polished UI is wasted on an audience of one
- Payload is database-agnostic (MySQL, Postgres, SQLite); adds one more config decision
- Cost compounds: backend hosting + database + Astro hosting

## Decision

**Firebase wins.**

Reason: Robin's single hard constraint is "I don't want to replatform. I want one cloud account and one deploy target." Firebase is already battle-tested in `agent_quillpad`. Firebase Auth + Firestore handle auth and data access. Firebase Functions natively run Genkit flows. The Admin SDK is mature and Robin has production patterns he can reuse (see `/agent_quillpad` for proof).

Git-native content would be nice, but the marginal benefit for a solo author writing at his own pace doesn't justify three vendors + Better Auth ops burden. Payload's UI is lovely, but a backend service + separate database is overkill for "share thoughts with 20 friends."

The trade-off we're accepting: no git-history on content. Content lives in Firestore. Future versions of Quillpad that export to Git are possible (Firestore snapshot → MD files → Git commit is a straightforward Cloud Function). But for v1, simplicity wins.

## Consequences

1. **Infrastructure:** Quillpad standardizes on Firebase for compute, storage, auth, and hosting (see `tech/01-stack.md` for full stack). This is now a hard constraint.

2. **Content model:** Content is persisted in Firestore, not Git. Version history must be implemented in the app layer (if at all); it's not free.

3. **Vendor lock-in:** If we ever want to leave Google, we'll need to export all Firestore data + rewrite the backend. This is a real cost. The bet is that Firebase's stability + Robin's risk-aversion make this acceptable.

4. **Genkit binding:** The agent service uses Firebase Genkit (Google's serverless AI framework). However, `tech/01-stack.md` maintains a `/api/agent/*` boundary contract, so swapping Genkit for a different inference service later is possible without frontend changes.

5. **Data safety:** Robin is responsible for Firestore backups. The system should include a scheduled Cloud Function that snapshots Firestore → Cloud Storage monthly (deferred to post-v1 if not critical).

## Open questions

- Should we implement Firestore point-in-time recovery as part of v1 setup, or defer as a post-launch ops task?
- Do we need a Cloud Functions-based export task to dump Firestore → JSON snapshots for archival?
