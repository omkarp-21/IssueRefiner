# Backend Contract Audit — Repository Issues UI

Date: 2026-10-12. Scope: read-only audit of the Go backend behind the Issues
screen. No backend, frontend, migration, or test files were changed
(a throwaway verification test was created, run, and deleted).

Related: `docs/frontend-handoff.md` (API behavior source — confirmed accurate
on every point re-checked; this report does not overwrite it).

## 1. Executive verdict

**READY WITH CAVEATS** — the backend correctly supports the Issues screen.
Every reported symptom traces to frontend behavior, genuine GitHub data, or a
documented API limitation — not to a backend defect. No backend code changes
are needed. The caveats are three API limitations the UI must design around:
no total counts, no server-side search, and capitalized JSON keys with
name-only labels.

## 2. Verified API contract

### `GET /api/repos/:id/issues`

- Route: `internal/repos/handler.go:190`; handler `ListIssues` (`:149`);
  service `ListIssues` (`internal/repos/service.go:133`); GitHub fetch
  `Client.ListIssues` (`internal/github/repos.go:140`).
- Auth: session required; `401 {"error":"Unauthorized"}` when missing
  (`internal/auth/middleware.go:15-37`). Repository resolved via
  org-scoped SQL `WHERE id=$1 AND organization_id=$2`
  (`internal/repos/repository.go:132-138`) — cross-org UUIDs return
  `404 {"error":"Repository not found"}`, never an empty list.
- Parameters (all verified end to end, not just accepted):
  - `state`, default `"open"` (`handler.go:161`); allowed `open|closed|all`
    (`ValidateIssueState`, `service.go:68-73`); anything else → `400
    {"error":"Invalid state"}`.
  - `page` default 1, `per_page` default 30, clamped to 1–100
    (`NormalizePage`, `service.go:55-66`; `parsePagination`,
    `handler.go:27-31`). Non-numeric values decode as 0 → normalized
    (page→1, per_page→30). No silent truncation below 100.
  - Values are forwarded **verbatim** to GitHub
    (`repos.go:142-146`, `?state=&page=&per_page=`); confirmed by a
    throwaway httptest run (`state=closed&page=2&per_page=30` observed on
    the wire — test deleted afterwards).
- Freshness: every request hits GitHub live; nothing is cached or read from
  the local DB. No import step is involved.
- Post-fetch filter: items with a `pull_request` member are **dropped**
  (`repos.go:151-154`). A full GitHub page can therefore return fewer rows
  than `per_page`. Covered by `TestListIssuesFiltersPullRequests`
  (`internal/github/repos_test.go:59`).
- Envelope: `{"issues": [...], "page": <echo>, "per_page": <echo>}`
  (`handler.go:181`). **No total counts, no next-page flag.** `page`/`per_page`
  echo the *requested* (normalized) values, not result metadata. Clients must
  infer "more pages" heuristically (e.g. full page returned).
- Issue object — Go struct has **no JSON tags** (`github/repos.go:39-49`),
  so keys serialize capitalized exactly as below (illustrative values only):

```json
{"Number": 421, "Title": "Real issue", "Body": "body", "State": "open",
 "Author": "alice", "Labels": ["bug"],
 "CreatedAt": "2024-01-01T00:00:00Z", "UpdatedAt": "2024-01-02T00:00:00Z",
 "HTMLURL": "https://github.com/o/r/issues/421"}
```

  - Labels are **name strings only**; GitHub label colors are decoded
    (`issueResponse.Labels[].Name`, `repos.go:76-78`) and then discarded —
    color is unavailable by design, not by accident.
  - Author is `user.login`; PR numbers are unreachable (single-issue fetch
    returns 404 for PRs, `repos.go:179-181`).
- Errors (never 200-with-misleading-empty): bad UUID → 400; unknown state →
  400; unknown/cross-org repo → 404; no GitHub token → 403 "GitHub account
  not connected"; corrupt/legacy token → 403 reconnect hint; GitHub
  401/404/429/5xx → mapped via `githubStatus` (`handler.go:33-46`) to
  401/404/429/502/500 with safe messages. Auth failures always surface as
  non-2xx, so the UI cannot mistake them for an empty list.

### Investigation + Brief (both exist; neither needs a new route)

- New Investigation: `POST /api/repos/:id/investigate`
  (`internal/jobs/handler.go:151`, route `:144`) with
  `{provider_config_id: <uuid>, model: <string>, issue_number: <int>}`.
  Validation (`service.go:128-140`): all three required, `issue_number > 0`
  (else 400). The server fetches the live issue + comments itself, so repo
  ID + issue number are sufficient. Provider config must belong to the org,
  be enabled, and have the model enabled (`ResolveForJob`,
  `internal/providers/service.go:246-259`), else 404/400. Success → **202
  `{"job": {...}}`** (`handler.go:198`). Job row: `id, organization_id,
  repository_id, provider_config_id, provider_type, model, created_by,
  kind:"investigate", issue_number, status, input_title, input_body,
  result:null, error:null, input_tokens:null, output_tokens:null, attempts,
  created_at, updated_at` (`internal/jobs/model.go:18-38`).
- Brief: `POST /api/repos/:id/refine` (`handler.go:43`, route `:142`) with
  the same three fields **plus** `title` (required non-empty) and `body`;
  `title+body` ≤ 24KB (`MaxIssueBytes`, `service.go:27`). Success → **202
  `{"job": {...}}`** with `kind:"refine"`. This is the Brief action — do not
  conflate it with the GitHub external link or with investigate.
- Polling: `GET /api/jobs/:id` → `200 {"job": {...}, "repository": {...}|null}`
  (`handler.go:85-107`); cross-org job IDs return 404 via `checkOrg`
  (`service.go:95-100`). `status: queued|running|done|failed`
  (`model.go:12-16`); `result` is a JSON **string**; `error` is a safe
  message string. History per issue: `GET
  /api/repos/:id/issues/:number/investigations` → newest-first, `LIMIT 50`,
  `kind='investigate'` only (`internal/jobs/repository.go:107-115`).
- Provider picker for both flows: `GET /api/providers` already exists
  (member-readable per handoff §3).

## 3. Issue-by-issue diagnosis

1. **Only ~17 issues available** — Frontend-suspected, no backend defect.
   The endpoint returns up to `per_page` (30) GitHub items minus PRs; a page
   with 13 PRs yields exactly 17 rows. The UI fetches one page
   (`per_page=30` hardcoded, `RepositoryIssues.tsx:71-73`) and its tab badge
   shows `issues.length` (current-page count, `:245`) as if it were a total.
   Genuine-data hypothesis (repo actually has ~17 open non-PR issues) cannot
   be excluded without the repo name — but no backend cap, cache, or
   truncation exists that could produce this number.
2. **Search only searches loaded issues** — Frontend defect confirmed +
   missing backend capability confirmed. The filter runs over the fetched
   page only (`RepositoryIssues.tsx:97-112`); the backend exposes no `q` /
   search parameter and the GitHub Search API is not wired. Full-repo search
   would need a new backend capability; not recommended in this pass.
3. **Closed shows ~3 issues** — Backend cannot explain; frontend-suspected /
   insufficient evidence. `state=closed` reaches GitHub verbatim (proven by
   test); no closed-excluding default, DB query, or mapping exists. The count
   is GitHub's closed non-PR items on page 1 — plausibly genuinely 3, or
   skewed by closed PRs being filtered. Needs the repo's real numbers, not a
   code change.
4. **Open/Closed switch behaves incorrectly** — Frontend-suspected. Backend
   accepts `open|closed|all` correctly; the UI only offers open/closed, resets
   page on switch, and re-queries with the state in the key (`:69`), which is
   correct plumbing. The visible defect is the count badge rendering the
   current page length next to each tab, which reads as a wrong total.
5. **Labels `bug`/`enhancement` missing** — Frontend-suspected or
   data-dependent; one minor capability gap confirmed. The backend maps every
   label name (`repos.go:155-158`) and the mapping is test-covered
   (`Labels[0] != "bug"` assertion, `repos_test.go:73`). The UI reads
   `issue.Labels ?? issue.labels` (`:359`), matching the capitalized wire
   keys. If labels don't render, causes are upstream (GitHub returned none)
   or in UI rendering — not in backend mapping. Gap: label **colors** are
   dropped server-side, so the UI's name-heuristic coloring (`:40-49`) is the
   only option without backend work.
6. **Brief action missing** — Frontend gap confirmed; backend ready. `POST
   .../refine` exists with documented validation (title required, 24KB cap).
   The 499-line `RepositoryIssues.tsx` contains no Brief trigger (only the
   GitHub external link, `:437-451`). Wire the existing endpoint; no backend
   change needed.
7. **New Investigation not implemented** — Frontend gap confirmed; backend
   ready. `POST .../investigate` + `GET /api/jobs/:id` polling + `GET
   /api/providers` picker cover the whole flow. No additional route or
   capability is needed.

## 4. Minimal backend fixes

**None.** No backend code changes are necessary for the Issues UI. Do not
add `q` search, totals, label colors, or `state` handling changes without a
separate product decision — each would expand the GitHub-integration surface
for problems currently explained by UI behavior. If a later pass wants them,
the smallest touchpoints are `Client.ListIssues` (`internal/github/repos.go`)
for colors/search and the envelope in `repos/handler.go:181` for counts.

## 5. Frontend handoff (for the Issues screen build)

- `GET /api/repos/{uuid}/issues?state=open|closed|all&page=N&per_page=M`
  (per_page ≤ 100; default 30). Live GitHub data, one page per call.
  Response: `{"issues": [...], "page", "per_page"}` — no totals, no
  next-page flag; `page`/`per_page` echo the request.
- **Expect fewer rows than `per_page`: pull requests are removed
  server-side.** Never present `issues.length` as a total; label tab counts
  as page counts or remove them. Paginate with Next while a full page
  returns; handle a short final page.
- Issue keys are **capitalized**: `Number, Title, Body, State, Author,
  Labels[] (names only, no colors), CreatedAt, UpdatedAt, HTMLURL`. Keep the
  existing dual-casing reads.
- Search box = client-side filter over loaded pages only; do not imply
  server-side search. `state=all` is supported if the UI wants it.
- Brief per row: `POST /api/repos/{uuid}/refine`
  `{provider_config_id, model, issue_number, title, body}` → 202 `{job}`.
- New Investigation per row: `POST /api/repos/{uuid}/investigate`
  `{provider_config_id, model, issue_number}` → 202 `{job}`; poll `GET
  /api/jobs/{jobId}` (~3s) until `done|failed`; provider/model picker from
  `GET /api/providers`. GitHub link per row: `HTMLURL` in a new tab.
- Errors are always non-2xx `{"error": msg}` — render the message; 401 →
  login; 404 on a repo UUID means not-found-or-foreign-org (do not
  distinguish).

## 6. Verification results

- `go build ./...` — pass. `go vet` on `github`, `jobs`, `repos` — clean.
- `go test -count=1 ./internal/github/ ./internal/jobs/ ./internal/repos/
  ./internal/investigation/` — all `ok` (no DB required; these packages use
  httptest doubles and fakes).
- Throwaway httptest (`TestAuditListIssuesQueryPassthrough`, since deleted)
  proved `state=closed&page=2&per_page=30` arrive verbatim at the GitHub
  request; existing `TestListIssuesFiltersPullRequests` proves PR filtering
  + label-name mapping.
- Limitations: no live-GitHub or DB-backed end-to-end run (needs OAuth
  token / Postgres); handler-level `ListIssues` test coverage does not exist
  upstream (client + service validation are covered instead); exact per-repo
  issue/PR counts behind the reported numbers were not observable from code.
  No LLM key was needed (no provider execution was exercised).
