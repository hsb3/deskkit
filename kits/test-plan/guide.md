---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — test-plan

## When to write one

You have an engineering-spec (the WHAT) and need to say HOW you'll prove it works before you trust it shipped. The spec's acceptance criteria tell you *what* done looks like; the test plan is *the evidence strategy* that makes each criterion checkable. Write one whenever a spec carries real risk — anything with branching logic, external dependencies, or a failure mode that would bite a user.

**The distinction that matters:** an `engineering-spec` describes the behavior to build; a `test-plan` describes the proof that the behavior is correct. They're a pair — the engineering-spec's `test_plan_link` field points here, and this note's `parent_engineering_spec` points back. If you're writing acceptance criteria, you're in the spec; if you're writing test cases, you're here.

**Granularity:** one test plan per engineering-spec. If a plan is sprawling across several specs, the specs were probably too big — split them, and the test plans fall out per unit.

## How to write one

1. **Start from the spec's acceptance criteria.** Every criterion must end up as a row in the coverage matrix. That mapping is the whole point — an untested criterion is an unproven claim.
2. **Set scope first — including what's *out*.** Name the adjacent concerns you are deliberately not testing and why (covered elsewhere, deferred, not a risk). A test plan with no out-of-scope list hasn't drawn its edges.
3. **Layer the approach.** Push each check to the cheapest level that can catch the failure: unit for logic, integration for seams, e2e for the user-visible flow, manual only for what a human must judge. Don't e2e what a unit test proves.
4. **Write concrete test cases.** Each case is a scenario + steps + expected result a friend could execute without asking you what you meant. "It works" is not an expected result.
5. **Fill the coverage matrix.** Requirement → test case → level. Blank cells are gaps, not decoration. This table is what turns "we wrote some tests" into "we proved the spec."
6. **State environments honestly.** Where each level runs and what data/keys/fixtures it needs. A plan that pretends e2e runs with no seed data is fiction.
7. **Make exit criteria binary.** Each is checkable, not aspirational. "Adequate coverage" is not an exit criterion; "every matrix row passing + zero open P0/P1" is.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Coverage matrix complete; shared for review |
| `in-review` | `approved` | Reviewer agrees the plan proves the spec |
| `approved` | `building` | Tests being written / executed |
| `building` | `shipped` | All exit criteria met; suites green |
| any | `shelved` | Feature pulled or plan superseded; reason noted in body |

## Anti-patterns

- **Coverage gaps hidden** — acceptance criteria with no matching test case. The matrix exists to surface these; a blank cell is a confession, not an omission.
- **Everything is e2e** — slow, flaky, and proves less than a focused unit test. Push each check down to the cheapest level that catches the failure.
- **Vague expected results** — "the page loads" / "it works." If a reviewer can't tell pass from fail without asking you, the case isn't written.
- **No out-of-scope list** — a plan that claims to test "everything" tests nothing in particular. Draw the edges.
- **Aspirational exit criteria** — "good coverage," "mostly passing." Exit criteria are binary gates or they don't gate.
- **Spec-as-test-plan** — re-listing the acceptance criteria with no test cases behind them. That's the spec; this is the proof.

## Example

See `example.md` in this folder for a complete, real test plan drawn from the Quillpad project.
