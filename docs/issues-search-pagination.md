# Issues Search & Pagination — Backend Contract (follow-up)

Supersedes §2 (issues endpoint) of `docs/issues-backend-audit.md`; all other
audit verdicts stand. Live GitHub semantics spot-checked against the public
API on 2026-10-12 (no auth); all behavior tests below are httptest-based.

## Endpoint

`GET /api/repos/:id/issues?state=&q=&page=&per_page=`

| Param | Semantics |
|---|---|
| `state` | `open` (default) `\| closed \| all`; anything else → `400 {"error":"Invalid state"}` |
| `q` | Optional. Absent or blank (whitespace-only) → recent-issues listing. Non-blank → repository-wide search (see below). Clearing `q` returns to the listing. |
| `page` | 1-based, `<1` → 1. |
| `per_page` | Default **7**, clamp 1–100. Missing/invalid/non-positive → 7. |

## Listing path (no `q`)

- Source: `GET /repos/{owner}/{repo}/issues?state=&sort=updated&direction=desc`
  (upstream `per_page=100`), walked page by page. **Ordering changed** from
  GitHub's default (created-desc) to most-recently-updated-first, per product
  requirement. Equal `updated_at` ties keep GitHub's order (documented,
  deterministic in practice).
- Each app page holds exactly `per_page` **real issues** (PRs excluded)
  whenever that many remain; the walk fetches further upstream pages as
  needed (max 10 upstream pages ≈ 1000 raw items per request).
- Applies identically to `open`, `closed`, and `all`.

## Search path (`q` present)

- Source: `GET /search/issues?q={q} repo:{owner}/{repo} type:issue
  [state:{state}]&sort=updated&order=desc` (upstream `per_page=100`), same
  fill-up walk. Ordering: most recently updated (documented recency policy,
  not relevance ranking).
- Scope qualifiers are appended server-side: user input cannot escape the
  repository (an injected `repo:other` ANDs to empty) or override the state
  filter (a conflicting `state:` qualifier ANDs to empty — fail-closed).
- Numeric `q` (e.g. `172`) additionally resolves `GET
  /repos/{owner}/{repo}/issues/172` directly and prepends it when it exists,
  is not a PR, and matches `state` — text search alone cannot match a number
  absent from title/body (verified live). Direct-fetch 404s are ignored; other
  upstream failures surface from the text search instead of as empty results.
- PRs are excluded twice: by the `type:issue` qualifier and by the same
  client-side guard as the listing path.
- Limits: GitHub exposes only the first 1000 search results; paging beyond
  that window surfaces GitHub's error through the normal taxonomy, never an
  empty page. `total_count` is intentionally unused — the PR filter and
  numeric merge make it unreliable for page accounting.

## Envelope (additive — old fields unchanged)

```json
{"issues": [...], "page": 1, "per_page": 7, "has_more": true}
```

- Issue objects are byte-identical to before (capitalized keys, name-only
  `Labels`, no colors).
- `has_more` is computed empirically, never from page fullness: true only
  when a further real issue was observed (one-item lookahead) or when the
  10-page safety walk stopped over full upstream pages. A short or empty
  upstream page proves exhaustion (`false`). No total count is fabricated.
- Errors unchanged: 400 bad id/state, 404 unknown or foreign-org repo, 403
  token problems, 401/404/429/502/500 mapped from GitHub via the existing
  taxonomy. Upstream failures are never returned as empty lists.

## Why closed issues "disappeared" (no separate defect)

`state=closed` already reached GitHub verbatim (proven in the audit). The
connected repo's closed pages are ~93% pull requests, so a single 10-item
page can contain zero real issues. The fill-up walk fixes this as a side
effect: closed page 1 now skips PRs until 7 real closed issues are collected.
No closed-specific code path was wrong; no database or mapping filter exists.

## Frontend instructions (exact)

- Send `per_page=7` (or omit it — 7 is the default); always send `page`.
- Render `issues` as-is; drive Next/Previous off `has_more` (`Next` enabled
  ⇔ `has_more`; `Previous` enabled ⇔ `page > 1`). Never display a total;
  none exists.
- Search box → `q` param (URL-encoded by `fetch`/`URLSearchParams`); empty
  box → omit `q` entirely to return to the listing. Search is repository-wide
  and state-aware; label the box accordingly (it is no longer page-scoped).
- `has_more: false` with `issues: []` is a genuine end — render the empty
  state, not an error. `has_more: true` with a short page is possible near
  the safety bound; keep Next available per the flag.
- New keys consumed: `has_more: boolean`. All other keys unchanged, still
  capitalized (`Number`, `Labels`, `HTMLURL`, …).
- No polling, auth, or flow changes. Provider/investigation contracts
  untouched.

## Tests (all httptest/fake-based, no keys, no DB)

`internal/github/issues_pagination_test.go`: PR-heavy fill-up to 7 with
lookahead `has_more`; cross-upstream-page advance; adjacent-page
no-duplicates; final short + empty pages; open/closed/all propagation;
`sort=updated&direction=desc`; 10-page safety cap with bounded request count;
500/429 surfacing (never empty); text search with qualifier + label mapping;
numeric direct-hit-first + state-mismatch exclusion; search PR exclusion +
`type:issue`; scope/state qualifiers; blank-`q` routing to the listing;
search pagination without duplicates. `internal/repos/service_test.go`:
`NormalizePageWithDefault` incl. default 7 and 100-cap.
`internal/repos/handler_test.go`: query-less → `(1,7)`; invalid params.
Existing `TestListIssuesFiltersPullRequests`, `TestNormalizePage`,
`TestValidateIssueState` updated/preserved and passing.

## Unresolved edge cases

- `updated_at` ties order by GitHub's internal order (stable, undocumented).
- Search `incomplete_results:true` (GitHub timeout) still returns partial
  items with empirically computed `has_more`; flagged by GitHub, not hidden.
- Upstream `per_page=100` + `maxUpstreamPages=10` are constants in
  `internal/github/repos.go`; raise only with rate-limit analysis.
- Very deep app pages on PR-dense repos cost up to 10 upstream requests per
  call; typical browsing (pages 1–3) costs 1–2.
