---
type: sow
status: approved
created: 2026-05-06
updated: 2026-05-13
tags: [client-engagement]
author: robin
owner: robin
commissioned_by: Northvale Roasters
target_ship_date: 2026-07-31
---

# Statement of Work: Northvale Roasters — Marketing Site Rebuild

_Prepared by Bramble Studio for Northvale Roasters. This Statement of Work defines the rebuild of the Northvale Roasters marketing website and its handover to the Northvale team._

## Context / background

Northvale Roasters is a single-location specialty coffee roaster expanding into wholesale and a small online wholesale-inquiry channel. Their current site is a five-page template build from 2021 hosted on a page-builder plan: slow on mobile, uneditable by the Northvale team without the original freelancer, and with no path to capture wholesale leads beyond a generic contact form that lands in a personal inbox.

Northvale wants a site they can run themselves — update opening hours, swap the seasonal coffee lineup, post the occasional journal entry — and a clean wholesale-inquiry flow that routes leads somewhere they can track. Speed on mobile matters: most of their traffic is on phones, in or near the cafe.

The owner cares about two things above all: the site must be fast on a phone, and the team must be able to edit copy and the coffee menu without calling anyone.

## Objectives

- A marketing site the Northvale team can edit themselves — hours, menu, and journal posts — with no developer involvement.
- A wholesale-inquiry flow that captures structured leads and routes them to a tracked shared inbox.
- A fast, mobile-first experience that loads quickly on a phone over cafe wifi or cellular.

## Scope

### In scope

- Rebuild of the five existing marketing pages (Home, Our Coffee, Wholesale, Journal, Visit/Contact) on a modern static stack.
- A lightweight CMS so the Northvale team can edit page copy, the coffee menu, hours, and journal posts.
- A structured wholesale-inquiry form (business name, volume, contact) routing to a shared, trackable inbox.
- Migration of the existing journal entries (currently 9 posts) into the new CMS.
- Mobile-first responsive design across the five pages, on the brand colors and type Northvale already uses.
- Deployment to managed hosting, on Northvale's existing domain, with one round of training for the team.

### Out of scope

- **E-commerce / online retail sales.** No cart, checkout, or payment processing — wholesale is inquiry-only under this engagement. A retail store is a possible later engagement.
- **Brand / logo / visual identity design.** We use Northvale's existing brand assets as provided. Creating new ones is separate.
- **Ongoing content writing.** We migrate existing copy and journal posts; writing new marketing copy is the Northvale team's job (the CMS exists so they can).
- **SEO / ad campaigns.** No paid acquisition, keyword strategy, or analytics-funnel work beyond basic page-level analytics setup.
- **Email marketing / newsletter.** No mailing-list integration. Deferred.

## Deliverables

| ID | Deliverable | Format | Acceptance |
|----|-------------|--------|------------|
| DEL-1 | Rebuilt five-page marketing site | Deployed site on Northvale's domain | Meets AC-1, AC-2, AC-5 |
| DEL-2 | Editable CMS with team accounts | Hosted CMS + 2 editor logins | Meets AC-3 |
| DEL-3 | Wholesale-inquiry form + routing | Form on the Wholesale page → shared inbox | Meets AC-4 |
| DEL-4 | Migrated journal archive | 9 existing posts live in the CMS | All 9 posts visible and editable |
| DEL-5 | Team training + handover doc | 1 live session (recorded) + 1-page how-to | Session delivered; Northvale can self-edit unaided |

## Approach / phases

| Phase | What happens | Output | Client checkpoint |
|-------|--------------|--------|-------------------|
| 1 — Setup & content audit | Stand up the stack; inventory existing copy, images, and journal posts; confirm brand assets and domain access | Working skeleton + content inventory | Review inventory; confirm nothing's missing |
| 2 — Build | Build the five pages, wire the CMS, build the inquiry form, migrate journal posts | Full site on a staging URL | **Sign-off on staging** before go-live |
| 3 — Launch & handover | Point the domain, verify on real devices, run the training session, hand over the how-to doc | Live site + trained team | Final acceptance against the criteria |

## Acceptance criteria

- **AC-1** — All five pages load with a Largest Contentful Paint under 2.5s on a mid-range phone over a throttled (Fast 3G) connection, verified on the staging URL.
- **AC-2** — The site renders correctly with no layout breakage on a phone, tablet, and desktop width (tested at 375px, 768px, 1280px).
- **AC-3** — A Northvale team member can, unaided, edit page copy, change a coffee on the menu, update hours, and publish a new journal post — demonstrated live during training.
- **AC-4** — A test wholesale inquiry submitted through the form arrives in the shared inbox within one minute, with all submitted fields intact.
- **AC-5** — All 9 existing journal posts are present, correctly formatted, and reachable from the Journal page.

## Assumptions & dependencies

| ID | Assumption or dependency | Type | Provided by | If unmet |
|----|--------------------------|------|-------------|----------|
| A-1 | Northvale's existing brand assets (logo, colors, fonts, photos) are usable as-is at adequate resolution | assumption | — | Design phase slips; may need a separate asset cleanup |
| A-2 | The five-page structure is final; no new pages or sections added mid-build | assumption | — | New pages are a change request (see Commercial terms) |
| D-1 | Domain registrar access (or DNS delegation) for go-live | dependency | Northvale | Phase 3 launch blocked; site stays on staging |
| D-2 | Final coffee-menu content and current hours, in writing | dependency | Northvale | Build proceeds with placeholders; launch waits on real content |
| D-3 | The shared inbox / destination for wholesale leads exists and is accessible | dependency | Northvale | DEL-3 routing can't be verified against AC-4 |

## Commercial terms

- **Model:** Fixed fee.
- **Total:** 8,500 USD.
- **Payment schedule:** 40% on signature, 60% on final acceptance (DEL-1 through DEL-5 accepted against the criteria).
- **Change requests:** Work outside the scope above (new pages, e-commerce, new brand assets, copywriting) is quoted separately and approved in writing before it starts. No change proceeds on a verbal request.
- **Expenses:** Third-party hosting and CMS subscription costs are billed to Northvale directly at cost (estimated under 30 USD/month); not included in the fixed fee.

## Sign-off

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Client | Dana Okafor, Owner — Northvale Roasters | | |
| Bramble Studio | Robin Vale | | |
