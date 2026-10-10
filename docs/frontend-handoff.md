# Issue Refiner — Frontend Handoff

Primary source of truth for implementing the Issue Refiner frontend. Every contract below was verified against the Go source; file paths are given so claims can be re-checked. Fictional values are used in all examples. No secrets are included.

## 1. Product

Issue Refiner is an AI-assisted GitHub issue investigation product. Central workflow: authenticate with GitHub → load user/org → browse connected repos → import a repo → browse issues → explicitly choose provider + model → launch async investigation → poll job → read structured, evidence-backed result → review missing-information drafts and explicitly post them → revisit history / re-investigate. Differentiator: evidence grounding (every hypothesis cites `issue` or `comment:<id>` actually supplied), readiness (`READY`/`NEEDS_INFORMATION`), ranked hypotheses — not code generation. Backend-verified support covers steps 1–11 of that workflow end to end; there is no org switching, member management, usage analytics, or dashboard data beyond per-row token counts.

## 2. Repository & backend architecture

- Layout: `cmd/server/main.go` (entrypoint, wiring, graceful shutdown) + `internal/{auth,config,database,github,health,jobs,orgs,providers,repos,sessions,tokens,users,worker}`. Go module `issue-refiner` (see `go.mod`); Gin + pgx + `golang.org/x/oauth2`.
- Layering: `handler` (Gin, thin, `c.JSON` only) → `service` (business logic, sentinel errors) → `repository` (parameterized SQL via pgxpool) → Postgres 17 (`docker-compose.yml`, `migrations/000001–000012`).
- No frontend exists: no `frontend/`, no `docs/`, no Stitch HTML anywhere in the repo (checked root — absent). Stitch output referenced by planning lives externally and must be supplied/recreated, not assumed present.
- Tests documenting behavior: `internal/auth/{handler,middleware}_test.go`, `logout_live_test.go`, `internal/{github,repos,jobs,providers,investigation,worker,orgs,sessions,crypto}/..._test.go`.

## 3. Verified API contract inventory

Base URL: same origin as the Go server (default `http://localhost:8080` via `SERVER_HOST`/`SERVER_PORT`). All `/api/*` require the session cookie (see §4); cross-org access returns the same shape as not-found (no existence leak). Universal error envelope: `{"error": "<human message>"}` — never trust extras.

| Method + route | Auth/role | Request | Success | Errors |
|---|---|---|---|---|
| `GET /health` | none | — | 200 `{"status":"ok"}` (`internal/health/handler.go`) | — |
| `GET /auth/github` | none | — | 303 redirect to GitHub | 500 |
| `GET /auth/github/callback?code&state` | `oauth_state` cookie | — | 200 `{"status","message","user":{…},"organization":{…}}` | 400 state mismatch; 504 timeout; 500s |
| `GET /api/me` | session | — | 200 `{"user":{…},"organization":{…},"role":"owner"‖"member"}` | 401 |
| `POST /api/logout` | session | — | 200 `{"status":"success","message":"Logged out successfully"}`, cookie cleared | 401; 500 only if row delete fails |
| `GET /api/repos` | member | — | 200 `{"repositories":[...]}` (server-capped, newest? ordered by query — treat as unordered) | 401/500 |
| `POST /api/repos/import` | member; `{owner,repo}` | `{"owner":"octo","repo":"demo"}` | 200 `{"repository":{…}}` (note 200, not 201) | 400/403 no-token / 403 expired / 404 / 502 / 500 |
| `GET /api/repos/:id` | member | UUID | 200 `{"repository":{…}}` | 400/404/500 |
| `GET /api/repos/:id/contributors?page=&per_page=` | member | defaults `page=1&per_page=30`, clamped | 200 `{"contributors":[],"page":1,"per_page":30}` | 400/403/404/502/500 |
| `GET /api/repos/:id/issues?state=&page=&per_page=` | member | `state` default `open`; invalid state → 400 `"Invalid state"` | 200 `{"issues":[],"page":1,"per_page":30}` | as above |
| `POST /api/repos/:id/refine` | member; `{provider_config_id:uuid, model, issue_number:int, title, body}` | caller-supplied text | 202 `{"job":{…}}` | 400 incl. disabled-provider message text / 404 repo-or-provider / 503 queue-full / 500 |
| `POST /api/repos/:id/investigate` | member; `{provider_config_id:uuid, model, issue_number:int}` | issue fetched live from GitHub | 202 `{"job":{…}}` | +403 no-token/expired, 404 issue, 502 upstream |
| `POST .../issues/:number/investigations/:investigation_id/reinvestigate` | member; same body as investigate | source must be `done` + `NEEDS_INFORMATION` | 202 `{"job":{…}}` (new row; source untouched) | +409 `"Investigation cannot be reinvestigated"` / 404 |
| `GET /repos/:id/issues/:number/investigations` | member | — | 200 `{"investigations":[...]}` newest-first, capped, failures included | 400/404/500 |
| `GET .../investigations/:investigation_id/request-draft` | member | — | 200 `{"comment": string}` (deterministic template, side-effect free) | 404/409/500 |
| `POST .../request-information` | member; `{"comment": string}` non-empty, ≤6000 bytes | explicit post as the user | 201 `{"id":int,"html_url":string,"body":string}` | 400 incl. oversize / 403 / 404+reconnect-hint / 409 / 502 / 500. Duplicates allowed by design |
| `GET /api/jobs/:id` | member | UUID | 200 `{"job":{…},"repository":{…}‖null}` | 400/404/500 |
| `GET /api/providers` | member | — | 200 `{"providers":[{"provider":{…},"models":[…]}]}` | 401/500 |
| `GET /api/providers/:id` | member | UUID | 200 `{"provider":{…},"models":[...]}` | 400/404/500 |
| `POST /api/providers` | **owner**; `{provider_type:"openai"‖"gemini"‖"mock", display_name, api_key, models:[...]}` (key validated live; nothing stored on failure) | validates live | 201 `{"provider":{…}}` | 400 incl. echoed detail / 401 bad key / 429 / 502-504 / 500 |
| `PATCH /api/providers/:id` | owner; `{display_name?,enabled?,api_key?,models?}` (omitted `models` keeps existing) | 200 `{"provider":{…}}` | as create + 404 |
| `DELETE /api/providers/:id` | owner | — | 204 empty | 404 / 409 `"Provider is in use by jobs"` / 500 |
| `POST /api/providers/:id/test` | owner | — | 200 `{"provider":{…}}` (status refreshed) | +400/401/404/429/504 via safe messages |

Discrepancies recorded, not smoothed over: import returns 200 (not 201); provider validation errors echo `err.Error()` verbatim (safe — taxonomy-checked, but render as-is); `ListConfigs` N+1s model queries (latency, not contract).

## 4. Authentication, sessions, security

- OAuth: `GET /auth/github` sets `oauth_state` (5 min) → 303 → callback validates state, exchanges code (5s ctx), upserts user, `EnsurePersonalOrg("<username> / personal")`, creates session, clears state cookie, sets `session_token` (`HttpOnly`, `SameSite=Lax`, `Secure=false`, `Path=/`, 7 days) — `internal/auth/handler.go:44,96-100`.
- Every `/api/*` call must send the cookie: `fetch(url, {credentials: "include"})`. Middleware resolves cookie → session row (expiry-checked) → user → org+role, else uniform 401 (`internal/auth/middleware.go`). No bearer/API-key scheme exists.
- `/api/me` returns `{user:{id,github_id,username,display_name,email,…}, organization:{id,name,…}, role}` — sole source for nav gating (owner sees provider mutations). No org switching exists (session is pinned to one org); do not build a switcher.
- Owner vs member: enforced server-side (`RequireOwner` → 403 on provider POST/PATCH/DELETE/test); mirror in nav only, never as the security boundary.
- Expired/invalid session → 401 on any call → redirect to login. Logout requires a valid session (else 401 — treat as logged-out) and clears the cookie server-side; HttpOnly means JS can never clear it directly.
- Cookies: `Secure=false` + no CORS handling in `main.go` is correct for same-origin localhost via Vite proxy, broken cross-origin — keep frontend same-origin through the dev proxy; never store tokens/keys in localStorage/sessionStorage; never render `api_key` (API never returns it — verify, don't assume).
- CSRF note: `SameSite=Lax` + state-cookie OAuth flow; no backend CSRF tokens. Do not weaken; do not propose changes in this task.

## 5. Investigation lifecycle & result contracts

- **Creation:** frontend supplies repo UUID (path), `issue_number`, `provider_config_id`, `model` (investigate) or plus `title`/`body` (refine). Returns **202** with full job row. Queue-full → 503 (job marked failed server-side).
- **Polling (no SSE/websockets exist):** poll `GET /api/jobs/:id` every ~3s; statuses `queued → running → done|failed`. Terminal success = `status:"done"`; failure = `failed` + `error` string (safe message).
- **Job fields:** `id, organization_id, repository_id?, provider_config_id?, provider_type?, model?, created_by, kind ("refine"|"investigate"), issue_number, status, input_title, input_body` (rendered evidence snapshot incl. `--- Comment by @login (comment:<id>) ---` banners), `result: string|null`, `error: string|null`, `input_tokens/output_tokens: int|null` (**null** on failed/mock jobs — never invent zeros; tokens ≠ money).
- **Result parsing — critical: `job.result` is a JSON *string*, not an object.** `JSON.parse` it, then validate shape client-side. Schema (`internal/investigation/model.go`): `{summary, readiness: READY|NEEDS_INFORMATION, missing_information[], signals[], observed_behavior, expected_behavior, environment: strings ("" = absent), reproduction_steps[], hypotheses: [{statement, confidence: HIGH|MEDIUM|LOW, reasoning, evidence: ["issue"|"comment:<digits>"]}]}`. `NEEDS_INFORMATION` guarantees non-empty `missing_information`; any other shape (incl. fences) is malformed — render an explicit "unparseable result" state, never partial data. **A `NEEDS_INFORMATION` *done* job is success-with-more-questions; a `failed` job is system failure — never conflate them.**
- **History:** `GET .../investigations` returns newest-first, cap 50 (`LIMIT 50` in SQL), failures included, `refine` rows excluded; open a run by its job `id` via `GET /api/jobs/:id` (current vs historical distinguished by which id you fetched — no separate type).
- **Re-investigation:** allowed only from `done` + `NEEDS_INFORMATION` sources (else 409); supply provider+model explicitly (never inherited); creates an independent job with *fresh* GitHub evidence; source row immutable.
- **Evidence/provenance:** only citable refs are `"issue"` + `"comment:<github-id>"` matching banner IDs in the *same job's* `input_body` (validator: `EvidenceIDs` + `ValidateGrounding`). GitHub `html_url`s come from the issue/contributor payloads. Safe claim: "hypothesis cites evidence actually supplied." Unsafe (never render): file paths, code locations, or causes as established fact.
- **Info-request flow:** draft GET is deterministic template from `missing_information` (no provider call, ≤6000 bytes, @mentions neutralized); POST requires explicit non-empty `comment` (≤6000 bytes enforced server-side) and posts **as the user**; 201 returns `{id, html_url, body}`; duplicates allowed — the modal's explicit confirm button is the only guard, never auto-post on draft fetch.

## 6. Stack, layout, design system, sequence

- Stack/architecture/routes per the approved brief (React + TS + Vite + Tailwind v4 + Router + TanStack Query polling + Radix primitives + native fetch + Geist/JetBrains Mono; `frontend/` new dir; Go/Postgres untouched; relative URLs + Vite proxy for `/api` + `/auth`; no Next.js/Redux/WebSockets). No compatibility blockers found; only open decision is same-origin proxy (recommended) vs CORS work (deferred — do not add backend CORS in this task).
- Colors/typography/constraints/screens/layout proposal Haus (`app/`, `components/{ui,layout}/`, `features/{auth,repositories,issues,investigations,providers}/`, `lib/{api,utils}/`, `styles/`, `types/`) and frontend routes (`/`, `/repositories`, `/repositories/:id`, `/repositories/:id/issues/:n`, `/investigations/:jobId`, modal routes for launch/history/comment-review) are **proposals carried over verbatim from the brief** — none exist in code. Stitch HTML is **not in the repo** (verified: no `docs/`, `frontend/`, or HTML artifacts); it must be supplied/recreated externally.
- Implementation order per brief §§1–12 (shell → auth → repos → issues → launch → jobs/results → evidence/history → comment/rerun → states → verify), one vertical slice first.

## 7. Verified facts
- 22 routes enumerated above with exact verbs/paths/statuses from handler code; `401` uniform shape; cross-org reads 404 without leaking existence; owner-gating on provider mutations only.
- Proof-bearing invariants verified in code: `input_body` snapshot rendered at enqueue; evidence IDs extracted from the same stored text the validator checks; provider snapshot immutable per job; `MarkDone/MarkFailed` status-guarded with terminal-write retry; bounded worker pool (3/32, 30s job timeout) with graceful shutdown; AES-GCM GitHub tokens + provider credentials under `CREDENTIAL_KEY`.
- Test files exist per domain (`*_test.go` in auth/crypto/github/investigation/jobs/orgs/providers/repos/sessions/worker); `docker-compose.yml` runs Postgres 17; migrations `000001–000012` cover all tables; config requires `DB_PASSWORD`, `DB_NAME`, `GITHUB_CLIENT_ID/SECRET/CALLBACK_URL`, `CREDENTIAL_KEY` (fail-fast otherwise).

## 8. Known limitations
- No logout-without-session nuance aside: dead cookie → 401 (treat as logged out); no logout-all-devices; no org switching/members/settings/audit endpoints; no aggregates (counts, token totals, rates, latency) — dashboard numbers beyond capped lists are unimplementable without new backend work; no pagination params on DB lists (fixed cap 50); GitHub-proxied lists use `?page/per_page/state`; `Secure=false` cookies and absent CORS are localhost-only postures; OAuth scope is empty (`oauth.go`), so private repos 404 — public repos only.

## 9. Open questions (unanswerable from the repo)
- Exact production deployment topology (same-origin vs separate frontend origin — decides whether backend CORS work is ever needed).
- Whether Stitch HTML will be supplied or recreated from tokens.
- TanStack Query polling interval / stale-time preferences under real provider latency (30s+ jobs) — operational tuning, not a contract gap.

## 10. Source map
- `cmd/server/main.go` — wiring, shutdown, route registration.
- `internal/{auth,orgs,users,sessions}/` — OAuth, cookies, `RequireAuth`/`RequireOwner`, roles.
- `internal/{repos,github}/` — import/detail/issues/contributors, GitHub client + error taxonomy.
- `internal/{jobs,worker}/` — enqueue, polling states, history, reinvestigate, info-request, worker bounds.
- `internal/investigation/` — result schema, evidence banners/IDs, grounding validator, prompt contract.
- `internal/{providers,tokens}/` — configs/models/validation, AES-GCM credentials, GitHub token storage.
- `internal/{config,database,health}/`, `docker-compose.yml`, `migrations/000001–000012`, `START.MD`, `TEST.MD`, per-package `*_test.go`.

## 11. First task for the frontend session
Initialize `frontend/` with the approved stack, implement shared design tokens + application shell, then connect authentication (`GET /api/me`, logout) and the repository list — **only after re-verifying §§3–5 above against the live code**, since this handoff, however carefully verified, is a point-in-time document and the backend remains the authority.
