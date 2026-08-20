// fixture.mjs — seed a throwaway desk the SPA checks can browse, search and edit.
//
// Thirty documents spanning all five non-meta status families, because the checks assert that
// the status picker offers a document's OWN family and nothing else — a fixture of one family
// would pass a picker that ignored the family entirely.
//
// Every document carries a `synopsis` key. That is load-bearing, not decoration: `synopsis` is
// read from frontmatter (sweep.go), never derived from the body, so a fixture without it renders
// zero preview rows and the "rows are fat enough to preview" check fails against a working app.
//
// Usage: node fixture.mjs <desk-root>
import { mkdirSync, rmSync, writeFileSync, existsSync } from 'node:fs'
import { join } from 'node:path'

export const PROFILE = `schema_version: 1

identity:
  name: "Demo Operator"
  github:
    personal: "demo-user"
  email: "demo@example.com"

repos:
  default: "demo-org/demo-repo"

board:
  provider: "github-projects"
  url: "https://example.com/board"
  number: 1

desk:
  name: "spa-desk"
  root: "."
  paths:
    decisions: "_structure/decisions"
    tasks: "tasks"
    analyses: "analyses"
    journal: "journal"
    secrets: "_meta/secrets"
    handoff: "_meta/HANDOFF.md"
    knowledge: "_knowledge"

machines:
  - name: "demo-machine"
    role: "primary"
    projects_root: "~/projects"

preferences:
  commit_style: "conventional"
  register: "explanatory"
  editor_url: "demo-editor://open?path={abs}"

custom: {}
`

// [dir, slug, doctype, status, extraFrontmatter[], title, synopsis]
export const DOCS = [
  // spec family — draft, in-review, approved, building, shipped, shelved
  ['specs', 'auth-rework-one-pager', 'one-pager', 'draft',
    ['author: demo-user', 'owner: demo-user', 'target_ship_date: 2026-10-01'],
    'Auth rework one-pager', 'Sign-in costs three round trips; this collapses it to one.'],
  ['specs', 'search-relevance-product-spec', 'product-spec', 'in-review',
    ['author: demo-user', 'parent_one_pager: auth-rework-one-pager', 'decomposes_to: search-ranking-feature-spec'],
    'Search relevance product spec', 'Ranking is lexical only, so recent work is hard to find.'],
  ['specs', 'search-ranking-feature-spec', 'feature-spec', 'approved',
    ['author: demo-user', 'parent_product_spec: search-relevance-product-spec', 'priority: high'],
    'Search ranking feature spec', 'Blend recency into the score above stale exact matches.'],
  ['specs', 'indexer-engineering-spec', 'engineering-spec', 'building',
    ['author: demo-user', 'parent_product_spec: search-relevance-product-spec', 'priority: high',
      'definition_of_done: index rebuild completes under 30s on a 10k-document desk',
      'test_plan_link: indexer-test-plan'],
    'Indexer engineering spec', 'Incremental updates keyed on checksum, so a sweep is not a rebuild.'],
  ['specs', 'indexer-test-plan', 'test-plan', 'approved',
    ['author: demo-user', 'priority: high', 'parent_engineering_spec: indexer-engineering-spec'],
    'Indexer test plan', 'Cold index, warm index, checksum collision, mid-sweep crash.'],
  ['specs', 'offline-mode-spec', 'product-spec', 'shelved',
    ['author: demo-user', 'parent_one_pager: auth-rework-one-pager', 'decomposes_to: none'],
    'Offline mode spec', 'Parked: the store is local already, so offline is the default.'],
  ['specs', 'export-pipeline-spec', 'engineering-spec', 'shipped',
    ['author: demo-user', 'parent_product_spec: search-relevance-product-spec', 'priority: medium',
      'definition_of_done: exports round-trip byte-exact', 'test_plan_link: indexer-test-plan'],
    'Export pipeline spec', 'Exports go through the same reversible door as every other write.'],
  ['specs', 'onboarding-ux-spec', 'ux-spec', 'draft',
    ['author: demo-user', 'parent_product_spec: search-relevance-product-spec'],
    'Onboarding UX spec', 'First run should show a populated desk, not an empty state.'],
  ['specs', 'billing-sow', 'sow', 'in-review',
    ['author: demo-user', 'owner: demo-user'],
    'Billing statement of work', 'Scope, deliverables and acceptance for the billing work.'],
  ['specs', 'rate-limit-technical-design', 'technical-design', 'approved',
    ['author: demo-user', 'priority: medium', 'affects_workstreams: platform'],
    'Rate limit technical design', 'A token bucket per desk, so a restart does not reset it.'],

  // reference family — draft, in-review, approved, archived
  ['_knowledge', 'glossary', 'reference', 'approved', [],
    'Glossary', 'Desk, sweep, patrol, finding, disposition: the words and what they mean.'],
  ['_knowledge', 'competitor-scan', 'research-synthesis', 'draft',
    ['author: demo-user', 'affects_workstreams: product'],
    'Competitor scan', 'Four tools in the adjacent space and where each stops short.'],
  ['_knowledge', 'persona-operator', 'user-journey', 'in-review',
    ['author: demo-user', 'persona: solo operator', 'related_product_specs: search-relevance-product-spec'],
    'Persona: the solo operator', 'One person, many projects, no team. Ceremony is a tax.'],
  ['_knowledge', 'legacy-import-notes', 'reference', 'archived', [],
    'Legacy import notes', 'How the old import worked. Kept for provenance only.'],
  ['analyses', 'latency-analysis', 'analysis', 'approved', ['author: demo-user'],
    'Latency analysis', 'p50 is fine; p99 is a cold rebuild, and it is avoidable.'],
  ['analyses', 'cost-analysis', 'analysis', 'draft', ['author: demo-user'],
    'Cost analysis', 'Token spend by surface. The agent loop dominates; browsing is free.'],

  // decision family — proposed, accepted, rejected, superseded
  ['_structure/decisions', '0001-files-are-authoritative', 'decision', 'accepted',
    ['decided_by: demo-user', 'affects_workstreams: platform'],
    'Files are authoritative', 'The store is a derived cache; a sweep overwrites it from disk.'],
  ['_structure/decisions', '0002-one-binary', 'decision', 'accepted',
    ['decided_by: demo-user', 'affects_workstreams: platform'],
    'One binary', 'CLI, MCP server, TUI and browser app in one process over one store.'],
  ['_structure/decisions', '0003-separate-web-service', 'decision', 'rejected',
    ['decided_by: demo-user', 'affects_workstreams: platform'],
    'Separate web service', 'Rejected: a second process is a second thing to get wrong.'],
  ['_structure/decisions', '0004-prose-editor-in-app', 'decision', 'proposed',
    ['decided_by: demo-user', 'affects_workstreams: product'],
    'Prose editor in the app', 'Proposed: the body hands off to an external editor instead.'],
  ['_structure/decisions', '0005-flat-doc-layout', 'decision', 'superseded',
    ['decided_by: demo-user', 'affects_workstreams: platform', 'superseded_by: 0001-files-are-authoritative'],
    'Flat document layout', 'Superseded once directories carried meaning.'],

  // cadence family — draft, published
  ['journal', '2026-08-17-weekly', 'weekly-checkin', 'published', ['week_of: 2026-08-17'],
    'Week of 17 August', 'Repo went public, board migrated, release cut.'],
  ['journal', '2026-08-10-weekly', 'weekly-checkin', 'published', ['week_of: 2026-08-10'],
    'Week of 10 August', 'Consolidation onto one binary landed; gates trimmed.'],
  ['journal', '2026-08-24-weekly', 'weekly-checkin', 'draft', ['week_of: 2026-08-24'],
    'Week of 24 August', 'Draft: the app phases, and the release scope question.'],
  ['journal', 'q3-retro', 'retro', 'draft', ['period_covered: 2026-Q3'],
    'Q3 retro', 'What shipped, what was shelved, and the one habit worth keeping.'],

  // project family — active, paused, completed, archived
  ['projects', 'app-overhaul', 'project', 'active', ['owner: demo-user'],
    'App overhaul', 'Three templates cover the whole app: CRUD, inbox, thread.'],
  ['projects', 'schema-next', 'project', 'paused', ['owner: demo-user'],
    'Schema next', 'Paused behind the element-model research loop.'],
  ['projects', 'board-migration', 'project', 'completed', ['owner: demo-user'],
    'Board migration', 'Tracked work moved off disk and onto the board.'],
  ['projects', 'container-distribution', 'project', 'archived', ['owner: demo-user'],
    'Container distribution', 'Archived: the released binary and the bundle cover this.'],

  // lightweight, no family
  ['tasks', 'wire-up-icons', 'task', 'draft', [],
    'Wire up the rail icons', 'Inline glyphs above the digit, never replacing it.'],
]

/** Write the profile and every fixture document under `root`, replacing anything already there. */
export function seed(root) {
  if (existsSync(root)) rmSync(root, { recursive: true, force: true })
  mkdirSync(join(root, '_knowledge'), { recursive: true })
  writeFileSync(join(root, '_knowledge', 'profile.yaml'), PROFILE)

  DOCS.forEach(([dir, slug, doctype, status, extra, title, synopsis], i) => {
    mkdirSync(join(root, dir), { recursive: true })
    const day = (i % 9) + 1
    const body = [
      '---',
      `type: ${doctype}`,
      `status: ${status}`,
      `created: 2026-08-0${day}`,
      `updated: 2026-08-1${day}`,
      `synopsis: "${synopsis}"`,
      `tags: [demo, ${doctype}]`,
      ...extra,
      '---',
      '',
      `# ${title}`,
      '',
      synopsis,
      '',
      '## Notes',
      '',
      'Fixture content. Real enough to browse, search and edit.',
      '',
    ].join('\n')
    writeFileSync(join(root, dir, `${slug}.md`), body)
  })
  return DOCS.length
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const root = process.argv[2]
  if (!root) {
    console.error('usage: node fixture.mjs <desk-root>')
    process.exit(2)
  }
  console.log(`seeded ${seed(root)} documents into ${root}`)
}
