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
