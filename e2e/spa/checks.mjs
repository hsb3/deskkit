// checks.mjs — the SPA's verification: drive a REAL browser against a running `deskkit serve`
// and assert behaviour the source cannot show you.
//
// Why a real browser rather than a DOM harness (ruled; see the decision on the board): the
// defects that actually reached this codebase were a destructive action firing at the wrong
// level, a stolen OS shortcut, and a frontmatter key written as `""` on every save. Only the
// last one was catastrophic, and it is invisible to any harness that stops at the DOM — you
// have to read the file on disk afterwards. Several checks below do exactly that, and they are
// the reason this suite exists.
//
// Numbered PASS/FAIL lines, non-zero exit on any FAIL, matching verify.sh's idiom.
//
// Usage: node checks.mjs <base-url> <desk-root>

import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { readFileSync, existsSync } from 'node:fs'
import { join } from 'node:path'

const BASE = process.argv[2]
const DESK = process.argv[3]
if (!BASE || !DESK) {
  console.error('usage: node checks.mjs <base-url> <desk-root>')
  process.exit(2)
}

// --- resolve playwright ---------------------------------------------------------------------
// Deliberately NOT a devDependency of web/: the SPA build and the container image run `npm ci`,
// and neither has any use for a browser driver. It is resolved from this directory's own
// node_modules, else the machine's global root. Absent means EXIT NON-ZERO with the fix — a
// silent skip here would read as "the app is verified" when nothing ran.
// `require` and not `import()`: playwright's entry point is CommonJS, and importing it by
// absolute path yields a namespace whose named exports are undefined.
const require = createRequire(import.meta.url)
let chromium
try {
  let resolved
  try {
    resolved = require.resolve('playwright')
  } catch {
    const globalRoot = execFileSync('npm', ['root', '-g'], { encoding: 'utf8' }).trim()
    resolved = require.resolve('playwright', { paths: [globalRoot] })
  }
  ;({ chromium } = require(resolved))
  if (!chromium) throw new Error('playwright resolved but exposes no chromium')
} catch {
  console.error('playwright is not available, so the browser checks did not run.')
  console.error('Install it, then re-run:')
  console.error('  npm install --prefix e2e/spa')
  console.error('  npx --prefix e2e/spa playwright install chromium')
  process.exit(2)
}

// --- harness --------------------------------------------------------------------------------
let N = 0
let PASS = 0
let FAIL = 0
const check = (desc, okp) => {
  N += 1
  if (okp) {
    PASS += 1
    console.log(`[${String(N).padStart(2, '0')}] PASS  ${desc}`)
  } else {
    FAIL += 1
    console.log(`[${String(N).padStart(2, '0')}] FAIL  ${desc}`)
  }
}
const note = (m) => console.log(`         ${m}`)
const disk = (rel) => readFileSync(join(DESK, rel), 'utf8')

// --- waiting -------------------------------------------------------------------------------
// Every wait below is a CONDITION, never a sleep. A fixed sleep passes locally and then goes
// flaky the first time CI is under load, and a flaky gate is one somebody eventually disables —
// which would put us back where this suite started.
const POLL_MS = 10_000

/** Poll `fn` until it returns truthy, or give up. Returns whether it ever did — callers that
 * care assert on it, callers that are merely settling the UI ignore it. */
async function until(fn, timeout = POLL_MS) {
  const deadline = Date.now() + timeout
  for (;;) {
    try {
      if (await fn()) return true
    } catch {
      /* mid-render: the locator may not exist yet */
    }
    if (Date.now() > deadline) return false
    await page.waitForTimeout(50)
  }
}

const untilVisible = (sel, t) => until(() => page.locator(sel).isVisible(), t)
const untilHash = (h, t) => until(async () => (await page.evaluate(() => window.location.hash)) === h, t)
/** Settle on a row count that has stopped changing.
 *
 * `differFrom` is load-bearing, not an optimisation: the list refetches asynchronously, so for
 * the first few polls after a keystroke the count is still the OLD one and perfectly stable.
 * Settling on stillness alone therefore returns the pre-search count and every later step acts
 * on the wrong row. Pass the count this is expected to move away from.
 *
 * If it never moves, this times out and returns the unchanged count — so the caller's assertion
 * fails honestly rather than hanging or passing. */
async function untilRowsSettle(differFrom = null, t) {
  let last = -1
  let stable = 0
  await until(async () => {
    const n = await page.locator('.browse tbody tr').count()
    stable = n === last ? stable + 1 : 0
    last = n
    if (differFrom !== null && n === differFrom) return false
    return stable >= 3
  }, t)
  return last
}
/** Wait for the file at `rel` to differ from `was` — a save is not done when the button is. */
const untilDiskChanges = (rel, was, t) => until(() => disk(rel) !== was, t)

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
// Everything here talks to a local server, so nothing legitimate takes seconds. The default 30s
// only ever gets spent on a check that is already failing, and it spends it three times over
// while reporting a misleading "did not find some options" instead of the real problem.
page.setDefaultTimeout(8000)
const pageErrors = []
page.on('pageerror', (e) => pageErrors.push(e.message))

/** Unwind to the finder and open the first row matching `term`.
 *
 * ⌘B and not ⌘<digit>: re-pressing the mode you are already in is a no-op (App's level reset
 * keys on the mode id), so the digit cannot bring the finder back from inside an edit. */
async function openDoc(term) {
  await page.keyboard.press('Escape')
  await until(async () => !(await atEditLevel()), 2000)
  await page.keyboard.press('Escape')
  if (!(await untilVisible('.browse .list', 1500))) {
    await page.keyboard.press('Meta+b')
    await untilVisible('.browse .list')
  }
  await page.keyboard.press('Meta+k')
  await until(() => page.locator('.search').evaluate((el) => el === document.activeElement))
  await page.locator('.search').fill(term)
  // Wait for the row we came for, not merely for the list to stop moving: a stable list can
  // still be the pre-search one.
  await until(async () => (await page.locator('.browse tbody tr').first().innerText()).includes(term))
  await page.keyboard.press('Escape')
  await page.keyboard.press('j')
  await until(() => page.locator('.browse tbody tr[aria-selected="true"]').count().then((n) => n > 0))
  await page.keyboard.press('Enter')
  await untilVisible('.instance')
}

/** Are we at the EDITING level?
 *
 * Not `.edit` being present: that form renders at the reading level too, with a `<span>` where
 * the control goes. Only the editing level swaps in a real `<select>`, so the control's
 * existence is the level. Waiting on `.edit` instead is a wait that can never finish — it
 * silently burns its whole timeout, which is precisely why a suite full of blind sleeps
 * appeared to work. */
const atEditLevel = () => page.locator('.edit select').count().then((n) => n > 0)

/** Enter the edit level and wait until it is actually usable.
 *
 * The two waits are one step because separating them is a trap: the form renders before the
 * status picker has its options, since those are derived from the DRAFTED doctype and fetched.
 * A call site that waits only for the form gets an empty `<select>`, and `selectOption` then
 * spends its whole timeout reporting "did not find some options" — which reads as a broken
 * picker rather than a race. That is exactly what CI caught on the two sites that had the
 * shorter wait, so the wait lives here and no call site can forget it. */
async function enterEdit() {
  await page.keyboard.press('e')
  await until(atEditLevel)
  await until(() => page.locator('.edit select').first().locator('option').count().then((n) => n > 0))
}

/** Save the open edit, and wait for the app to actually finish saving.
 *
 * Not the same as waiting for the bytes to land. The server writes the file and only then
 * responds, and only on that response does the component drop back to the reading level. So the
 * file changes while the app is still in `editing` — where `e` is deliberately a no-op — and a
 * caller that proceeds on the disk change alone presses `e` into the void, then has the form
 * close underneath the `selectOption` it was about to run. CI failed exactly that way twice.
 *
 * Returns whether the edit level closed, which is the honest signal for "the save succeeded":
 * a refused write stays at the editing level with its error. */
async function saveEdit() {
  await page.keyboard.press('Meta+Enter')
  return until(async () => !(await atEditLevel()))
}

try {
  // --- shell ----------------------------------------------------------------------------
  await page.goto(BASE, { waitUntil: 'networkidle' })
  await page.waitForSelector('.shell', { timeout: 15000 })
  check('cold load reaches the shell (loopback bootstrap authed; no login form)', true)

  // --- every rail mode renders --------------------------------------------------------
  const MODES = ['queue', 'library', 'patrol', 'work', 'agent', 'config']
  for (let i = 0; i < MODES.length; i++) {
    await page.keyboard.press(`Meta+${i + 1}`)
    const hash = (await untilHash(`#/${MODES[i]}`))
      ? `#/${MODES[i]}`
      : await page.evaluate(() => window.location.hash)
    const len = (await page.locator('.shell').innerText()).trim().length
    check(`mode ${i + 1} (${MODES[i]}) navigates and renders`, hash === `#/${MODES[i]}` && len > 0)
  }

  // --- the finder ------------------------------------------------------------------------
  await page.keyboard.press('Meta+2')
  await untilVisible('.browse .list')
  const allRows = await untilRowsSettle(0)
  check('the Library finder lists documents', allRows > 0)

  const previews = await page.locator('.preview-row').count()
  check('rows carry their synopsis preview line (rows are fat)', previews > 0)
  note(`${previews} preview rows`)

  await page.keyboard.press('Meta+k')
  await until(() => page.evaluate(() => (document.activeElement?.className ?? '').includes('search')))
  const focusedClass = await page.evaluate(() => document.activeElement?.className ?? '')
  check('the search shortcut focuses the search field', focusedClass.includes('search'))

  await page.locator('.search').fill('decision')
  const narrowed = await untilRowsSettle(allRows)
  check('search narrows the list', narrowed > 0 && narrowed < allRows)
  note(`${allRows} rows -> ${narrowed} rows`)

  // --- space follows engagement -----------------------------------------------------------
  await page.keyboard.press('Escape')
  await page.keyboard.press('j')
  await until(() => page.locator('.browse tbody tr[aria-selected="true"]').count().then((n) => n > 0))
  await page.keyboard.press('Enter')
  check('opening an item shows the instance', await untilVisible('.instance'))
  check(
    'the finder leaves the screen on open (it minimises into its rail button)',
    !(await page.locator('.browse .list').isVisible().catch(() => false)),
  )

  const bodyHref = await page
    .locator('.instance a.verb')
    .first()
    .getAttribute('href')
    .catch(() => null)
  check(
    "the body hand-off renders the desk's editor_url template as a link",
    Boolean(bodyHref && bodyHref.startsWith('demo-editor://')),
  )

  // --- the status picker reads the document's own family ------------------------------------
  const familyOf = async (term, expected) => {
    await openDoc(term)
    await enterEdit()
    const opts = await page.locator('.edit select').first().locator('option').allInnerTexts()
    check(
      `the status picker offers the ${expected[0]}-family statuses for ${term}`,
      JSON.stringify(opts) === JSON.stringify(expected),
    )
    if (JSON.stringify(opts) !== JSON.stringify(expected)) note(`offered ${JSON.stringify(opts)}`)
  }
  await familyOf('app-overhaul', ['active', 'paused', 'completed', 'archived'])
  await familyOf('0004-prose-editor', ['proposed', 'accepted', 'rejected', 'superseded'])

  // --- the write path, checked against the bytes on disk ------------------------------------
  const rel = 'projects/app-overhaul.md'
  const before = disk(rel)

  await openDoc('app-overhaul')
  await enterEdit()
  await page.locator('.edit select').first().selectOption('paused')
  const savedCleanly = await saveEdit()
  check('the save completes and drops back out of the edit level', savedCleanly)
  check('the save reaches disk within the timeout', await untilDiskChanges(rel, before))

  const saved = disk(rel)
  check('saving writes through to the file on disk', /^status: paused$/m.test(saved))
  // The one that matters: frontmatter key is `type`, the indexed column is `doctype`. Confusing
  // them once wrote an empty type on every save.
  check('the frontmatter type survives a status save', /^type: project$/m.test(saved))
  check(
    'a frontmatter-only save leaves the prose body alone',
    saved.includes('Three templates cover the whole app'),
  )

  await enterEdit()
  await page.locator('.edit select').first().selectOption('active')
  await saveEdit()
  await until(() => disk(rel) === before)
  check('setting the status back yields a byte-identical file', disk(rel) === before)

  await enterEdit()
  await page.locator('.edit select').first().selectOption('archived')
  await page.keyboard.press('Escape')
  await until(async () => !(await atEditLevel()))
  // Deliberately NOT a poll: the assertion is that nothing EVER lands, so give a write the same
  // grace a real save gets and then require the file to be untouched.
  await page.waitForTimeout(1000)
  check('backing out of an edit reverts it; nothing reaches disk', disk(rel) === before)

  // --- the write boundary refuses, visibly --------------------------------------------------
  // NOTE: this asserts today's behaviour. A ruled change makes a decision's STRUCTURED fields
  // writable while its prose body stays protected — when that lands, this check inverts to
  // "the status saves and the body is byte-identical", and the refusal check moves to a
  // body-altering write.
  const relProtected = '_structure/decisions/0004-prose-editor-in-app.md'
  const protectedBefore = disk(relProtected)
  await openDoc('0004-prose-editor')
  await enterEdit()
  await page.locator('.edit select').first().selectOption('accepted')
  await page.keyboard.press('Meta+Enter')
  await untilVisible('.instance .error')
  const refusal = await page.locator('.instance .error').first().innerText().catch(() => '')
  check('a refused write is shown to the user rather than swallowed', refusal.length > 0)
  note(refusal.replace(/\n/g, ' '))
  check('the refused write left the protected file untouched', disk(relProtected) === protectedBefore)

  // --- delete arms rather than fires --------------------------------------------------------
  await openDoc('wire-up-icons')
  const del = page.locator('.instance .verb.danger')
  const label1 = (await del.innerText()).trim()
  await del.click()
  await until(async () => (await del.innerText()).trim() !== label1)
  const label2 = (await del.innerText()).trim()
  check('the first delete click arms a visible confirm rather than deleting', label1 !== label2)
  note(`${JSON.stringify(label1)} -> ${JSON.stringify(label2)}`)
  check('the file is still on disk after the first click', existsSync(join(DESK, 'tasks/wire-up-icons.md')))

  // --- nothing threw ------------------------------------------------------------------------
  check('no uncaught page errors during the whole run', pageErrors.length === 0)
  pageErrors.forEach((e) => note(`pageerror: ${e}`))
} catch (e) {
  check(`the suite ran to completion (threw: ${e.message.split('\n')[0]})`, false)
  // A bare timeout says nothing about WHY. Dump the state the failing step was looking at, so a
  // CI-only failure does not need a round trip to diagnose.
  try {
    note(`hash: ${await page.evaluate(() => window.location.hash)}`)
    note(`edit form present: ${await page.locator('.edit').count()}`)
    note(`instance present: ${await page.locator('.instance').count()}`)
    note(`finder visible: ${await page.locator('.browse .list').isVisible().catch(() => false)}`)
    note(`title: ${(await page.locator('.instance-head').first().innerText().catch(() => '—')).split('\n').join(' | ')}`)
    const opts = await page.locator('.edit select').first().locator('option').allInnerTexts().catch(() => [])
    note(`status options on screen: ${JSON.stringify(opts)}`)
  } catch {
    note('(could not read page state)')
  }
} finally {
  await browser.close()
}

console.log(`\n${PASS}/${N} passed, ${FAIL} failed`)
process.exit(FAIL === 0 ? 0 : 1)
