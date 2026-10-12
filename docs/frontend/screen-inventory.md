# Screen Inventory

## Overview

- Source HTML: `docs/design-references/overview/code.html`
- Screenshot: `docs/design-references/overview/screen.png` (note: currently
  corrupted — overlapping text layers; usable for color impression only, not
  for exact comparison; needs a clean re-export before visual verification)
- Design notes: `docs/design-references/overview/DESIGN.md` (system
  "Monochrome Engineering"); discrepancies between `DESIGN.md` and `code.html`
  are recorded in `docs/frontend/design-system.md` §1 — `code.html` governs.
- Intended route: `/` (and currently also `/repositories`,
  `/repositories/:id`, which render the same component as a placeholder).
- React implementation: exists — `frontend/src/features/repositories/Overview.tsx`
  inside `frontend/src/components/layout/Shell.tsx` (`frontend/src/App.tsx`),
  rebuilt from `code.html` (Stitch tokens, fixed sidebar + header, filter
  input, import dialog, 3-column repo cards; icons as inline SVG instead of
  the Material Symbols webfont).
- Visually compared against reference: **yes, with caveats** (2026-10-10).
  Rendered `code.html` in headless Chromium at 1500×1000 (clean render —
  more reliable than the corrupted `screen.png`) and the React Overview with
  a temporary fetch stub (real `/api/me` + `/api/repos` shapes, Stitch-like
  placeholder rows; stub deleted afterwards). Layout, sidebar, header, page
  head, filter, import button, and card grid match closely; Geist + JetBrains
  Mono load correctly. Comparison used stubbed data, not a live backend
  session, and only the default/empty investigations state was rendered.
- Outstanding differences (React vs approved source, from static inspection):
  - Palette mismatch: React uses an `ink-*` scale (`#0a0a0b`–`#f7f7f8`) and
    accent `#2f6fed`; source specifies `#080808`/`#121212`/`#161616`/
    `#222222`/`#f0f0f0`/`#888888`/`#555555` and accent `#2f81f7`.
  - Mono stack is `Geist Mono`; source specifies `JetBrains Mono`. No icon
    system; source uses Material Symbols Outlined (replacement undecided).
  - Missing: fixed top header with breadcrumb, workspace switcher, filter
    input with `⌘K` hint, `Import repository` accent button (React has an
    owner/repo import form instead), 3-column repo card grid (React renders a
    plain row list), recent-investigations table with status badges,
    `Manage integrations` link, section counts.
  - Unsupported by the backend (verified against route registration in
    `internal/jobs/handler.go` + handoff §3): cross-repo recent-investigations
    listing (history is per repo+issue only) → deliberate empty state;
    issue counts / relative times on cards (would need one request per repo)
    → card footer shows real `default_branch` + `html_url` "Open" link
    instead; org switching (session pinned) → static org indicator;
    `Manage integrations`, header search/notifications/avatar (no endpoints)
    → omitted; `⌘K` shortcut → omitted (filter is client-side over loaded
    repos only); import flow → compact dialog driving real
    `POST /api/repos/import`.
  - Unknowns carried over: `Open` / `View result` destinations for future
    screens (card "Open" currently links to the repo's GitHub `html_url`).

## Repositories

- Source HTML: `docs/design-references/repositories/code.html`
- Screenshot: `docs/design-references/repositories/screen.png`
- Design notes: None provided in export (`DESIGN.md` intentionally omitted; requirements derived from `code.html` and `screen.png`).
- Intended route: `/repositories`
- React implementation: `frontend/src/features/repositories/Repositories.tsx` wired at route `/repositories` in `frontend/src/App.tsx`.
  - Rebuilt faithfully from `code.html` and `screen.png`: title, subtitle, "Import repository" action button, filter input with search icon and `⌘K` badge, dynamic connected counter, and full-width card list layout.
  - Card elements: Code/Lock icon in `#202020` surface with `#acc7ff` tint, repository full name with hover highlight linking to `/repositories/:id`, Public/Private mono pill, git branch pill with branch icon, markdown/emoji rendered description, "Open" button linking to `/repositories/:id`, card footer with `Updated <time>` and `Synced <time>` with green check circle.
  - Loading skeleton state matching card geometries, error state with human-friendly message, empty state for 0 repositories, and no-search-results state with clear filter button.
  - Inline outline SVG icons in `frontend/src/components/icons.tsx` (`CodeIcon`, `LockIcon`, `BranchIcon`, `CheckCircleIcon`, `ArrowRightIcon`) matching Material Symbols Outlined without external webfont dependency.
- Visually compared against reference: **yes** (2026-10-10).
  - Rendered in headless Chromium at 1500×1000 and compared side-by-side with `screen.png`.
  - Verified default loaded state with connected repos, filter search matching "auth", non-matching filter search "xyz" empty state, zero-repos connected empty state, API error state, and import dialog modal overlay.
- Backend mapping & boundaries:
  - Backed by `GET /api/repos` returning `{ repositories: Repository[] }`.
  - Import dialog backed by real `POST /api/repos/import` validating owner and repository, invalidating `["repos"]` query.
  - Cards navigate to the existing repository-detail route `/repositories/:id`.
  - Unsupported backend concepts omitted or grounded: open issue counts and language tags (not provided on `Repository` or `GET /api/repos`) omitted to avoid fabricating mock data; "Triage daemon" status in footer replaced with real `synced_at` database timestamp.

## Repository Issues

- Source HTML: `docs/design-references/issues/code.html`
- Screenshot: `docs/design-references/issues/screen.png`
- Design notes: `docs/design-references/issues/DESIGN.md` ("Monochrome Engineering")
- Intended route: `/repositories/:id`
- React implementation: `frontend/src/features/issues/RepositoryIssues.tsx` wired at route `/repositories/:id` in `frontend/src/App.tsx`.
  - Rebuilt faithfully from `code.html` and `screen.png`:
    - Back link to `/repositories`.
    - Repository identity row with full name, Private/Public chip (with lock icon), default branch chip, and emoji-rendered description.
    - Controls bar with `/` keyboard-shortcut search input, and segmented state filter tabs (`Open` with green dot and `Closed` with subtle dot).
    - Issue table with headers (`# ID`, `Title & Context`, `State`, `Updated / Action`), issue row styling, label chips with color categorizations (`bug` coral, `enhancement` green, generic raised), author metadata, relative time update, state dot, and direct link to the GitHub issue.
    - Pagination controls for multi-page issue browsing via `page` and `per_page=30`.
    - Loading skeletons, empty state (0 issues for state), search empty state with "Clear search" CTA, and repository error/invalid ID states with back navigation.
    - Footer with `/ to search` shortcut guide and `Issue Sync: Active` status.
- Visually compared against reference: **yes** (2026-10-10).
  - Rendered in headless Chromium at 1500×1000 and compared side-by-side with `screen.png`.
  - Verified default loaded view with open issues, closed issues view, client search filtering matching specific issue numbers/titles, empty state, and error handling for invalid repository IDs.
- Backend mapping & boundaries:
  - Repository fetched via `GET /api/repos/:id` returning `{ repository: Repository }`.
  - Issues fetched via `GET /api/repos/:id/issues?state=${state}&page=${page}&per_page=30` returning `{ issues: Issue[], page: number, per_page: number }`.
  - Casing resilience: `Issue` type supports both Go's standard field serialization (PascalCase) and potential snake_case formatting.
  - Unsupported actions omitted: "Rule Config" and "New Investigation" on the repo level, and "Launch" / "View Brief" on issues (investigation flow remains an independent bounded workflow per specification; action column links directly to the real GitHub issue via `HTMLURL`).

## Investigation Detail

- Source HTML: `docs/design-references/investigation/code.html`
- Screenshot: `docs/design-references/investigation/screen.png`
- Design notes: `docs/design-references/investigation/DESIGN.md` ("Monochrome Engineering"); `code.html` governs on conflicts.
- Intended route: `/investigations/:investigationId` (deep-linkable, refresh-safe).
- React implementation: `frontend/src/features/investigations/InvestigationDetail.tsx`
  (+ `result.ts` parser, `useJobPoll.ts` shared hook) wired in `frontend/src/App.tsx`.
  - Data from `GET /api/jobs/:id` (`{job, repository|null}`); `job.result`
    parsed as a JSON string with backend-mirroring validation (malformed →
    controlled invalid state, never raw JSON as primary UI).
  - Renders summary, readiness badge, missing-information list, ranked
    hypothesis cards (confidence, evidence pills, reasoning details),
    collapsible signals, observed/expected/environment context, reproduction
    steps, collapsed issue snapshot, per-issue run history with current-run
    marker and cross-navigation, and an evidence panel (references only —
    no invented excerpts or interpretations).
  - Real workflows only: request-information draft/post modal and
    re-investigate dialog (provider picker), both gated to
    done + NEEDS_INFORMATION as the backend requires. No excerpt
    fabrication, durations, scores, or vendor-name mappings.
  - Mock-provider jobs show a dashed "mock provider · dev data" badge.
- Visually compared against reference: **yes, with caveats** (2026-10-12).
  Headless Chromium vs `screen.png`: hierarchy, badges, hypothesis cards,
  drawer, and history match; mock-labeled data used throughout. Covers
  done/queued/failed/malformed/empty/404 states, polling transition,
  refresh, and 500px narrow layout (drawer becomes on-demand overlay).
- Backend mapping & boundaries:
  - Polling reuses the shared `useJobPoll` hook (3s, stops at terminal
    states); `RepositoryIssues.tsx` imports it (no competing poller).
  - Deliberately omitted vs Stitch (no backend support): breadcrumb repo
    trail beyond the back link, "Synced" pill, priority labels, execution
    durations, comment excerpts/authors in the drawer, "Insert citation",
    label-application checkbox on the request modal.
  - Not built (out of scope): global investigation history/listing endpoint,
    sidebar nav item, provider/workspace settings.
