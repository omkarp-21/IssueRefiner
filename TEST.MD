# TEST.MD — Manually test Issue Refiner

Assumes the server is running per `START.MD`. Auth is cookie-only (`session_token`, `HttpOnly`, 7 days) — no bearer headers exist. Every API call below needs the cookie; without it you get `401 {"error":"Unauthorized"}`.

## 0. Authenticate (browser required, once)

1. Visit `http://localhost:8080/auth/github` → authorize on github.com.
2. Callback returns `200 {"status":"success","user":{…},"organization":{…}}` and sets the `session_token` cookie.
3. Copy it: DevTools → Application → Cookies → `session_token`, then verify:
```bash
curl -b "session_token=<paste-value>" localhost:8080/api/repos
# → 200 {"repositories":[...]} (empty list is fine — it proves auth works)
```
Note: OAuth requests the `public_repo` scope, so **use public repos only**; private repos 404. Tokens issued before the scope change need a fresh login.

## 1. Repositories

```bash
S='session_token=<paste-value>'
# Import (idempotent upsert; needs your GitHub token to have access):
curl -b "$S" -X POST localhost:8080/api/repos/import \
  -H 'Content-Type: application/json' \
  -d '{"owner":"home-assistant","repo":"core"}'
# → 200 {"repository":{"id":"<REPO_UUID>",...}} — save REPO_UUID
R=<REPO_UUID>

curl -b "$S" localhost:8080/api/repos                        # list org repos
curl -b "$S" localhost:8080/api/repos/$R                     # detail (foreign-org UUID → 404)
curl -b "$S" "localhost:8080/api/repos/$R/contributors?page=1&per_page=5"
curl -b "$S" "localhost:8080/api/repos/$R/issues?state=open&page=1&per_page=5"
```

## 2. Providers (mutations are owner-only; login makes you owner of your personal org)

```bash
# Mock first (deterministic, offline, any api_key string works):
curl -b "$S" -X POST localhost:8080/api/providers \
  -H 'Content-Type: application/json' \
  -d '{"provider_type":"mock","display_name":"Mock","api_key":"x","models":["mock-investigate-ready","mock-investigate-ungrounded"]}'
# → 201 {"provider":{"id":"<PROV_UUID>","status":"healthy",...}} — save PROV_UUID (api_key never returned)
P=<PROV_UUID>

# Real providers (key validated live before saving; bad key → 401, nothing stored):
# -d '{"provider_type":"openai","display_name":"OpenAI prod","api_key":"<REAL_KEY>","models":["gpt-4o-mini"]}'
# -d '{"provider_type":"gemini","display_name":"Gemini prod","api_key":"<REAL_KEY>","models":["<model-from-discovery>"]}'
# Gemini validates via live model discovery (GET /v1beta/models); unknown names → 400.

curl -b "$S" localhost:8080/api/providers                        # list (safe metadata + nested models)
curl -b "$S" localhost:8080/api/providers/$P                     # detail
curl -b "$S" -X POST localhost:8080/api/providers/$P/test        # re-validate; updates status
curl -b "$S" -X PATCH localhost:8080/api/providers/$P \
  -H 'Content-Type: application/json' \
  -d '{"models":["mock-investigate-ready"]}'                     # replace model set (revalidates)
# DELETE → 204; 409 if jobs reference the config.
```

## 3. Investigations (async: 202 queued → poll to done/failed)

(Legacy synchronous-input variant also exists: `POST /api/repos/$R/refine` with caller-supplied `{"issue_number","title","body","provider_config_id","model"}` — same response and polling shape as below.)

```bash
# Start (issue fetched live from GitHub; needs your stored GitHub token):
curl -s -b "$S" -X POST localhost:8080/api/repos/$R/investigate \
  -H 'Content-Type: application/json' \
  -d '{"issue_number":<N>,"provider_config_id":"'$P'","model":"mock-investigate-ready"}' \
  | tee /tmp/inv.json | jq -r '.job.id'
# → 202 {"job":{...,"status":"queued","kind":"investigate",...}} ; save JOB_UUID
# 400 bad input/disabled provider · 404 repo/provider/issue · 403 GitHub not connected · 502 GitHub failed · 503 queue full

J=<JOB_UUID>
# Poll:
for i in $(seq 1 20); do S_=$(curl -s -b "$S" localhost:8080/api/jobs/$J | jq -r .job.status); echo "$S_"; [ "$S_" = done -o "$S_" = failed ] && break; sleep 3; done

# Inspect result + usage + exact stored input:
curl -s -b "$S" localhost:8080/api/jobs/$J | jq '{status:.job.status, kind:.job.kind, usage:[.job.input_tokens,.job.output_tokens], error:.job.error}'
curl -s -b "$S" localhost:8080/api/jobs/$J | jq -r '.job.result|fromjson'
curl -s -b "$S" localhost:8080/api/jobs/$J | jq -r '.job.input_body'   # exact rendered evidence (issue + --- Comment by @login (comment:<id>) --- banners)
```

## 4. Grounding checks (Phase 6)

- Valid refs: hypotheses' `evidence` must be `"issue"` or `"comment:<id>"` where each ID appears in a banner in `input_body`. Cross-check:
```bash
curl -s -b "$S" localhost:8080/api/jobs/$J | jq -r '.job.result|fromjson|.hypotheses[].evidence[]' | sort -u
curl -s -b "$S" localhost:8080/api/jobs/$J | jq -r '.job.input_body' | grep -o '(comment:[0-9]*)' | sort -u
```
- Negative: run with model `mock-investigate-ungrounded` (canned `file:…` ref) → job `failed`, `result` null.
- `NEEDS_INFORMATION` path: model `mock-investigate-needs-info` → `done` with empty hypotheses.

## 5. History & re-investigation

```bash
curl -s -b "$S" localhost:8080/api/repos/$R/issues/<N>/investigations | jq '.investigations | length'
# → newest-first, investigate-kind only, failures included

# Re-investigate from a NEEDS_INFORMATION job (new independent job, source untouched):
curl -s -b "$S" -X POST localhost:8080/api/repos/$R/issues/<N>/investigations/<JOB_UUID>/reinvestigate \
  -H 'Content-Type: application/json' \
  -d '{"provider_config_id":"'$P'","model":"mock-investigate-ready"}'
# → 202 new job; READY/failed/queued sources or foreign IDs → 409/404 respectively
```

## 6. Information requests (developer-reviewed GitHub comments)

```bash
# Draft (deterministic, side-effect free; only from done NEEDS_INFORMATION jobs):
curl -s -b "$S" localhost:8080/api/repos/$R/issues/<N>/investigations/<JOB_UUID>/request-draft
# → 200 {"comment":"To help investigate..."} (@mentions neutralized, ≤6000 bytes)

# Post (explicit body required; posts as YOU via your GitHub token):
curl -s -b "$S" -X POST localhost:8080/api/repos/$R/issues/<N>/investigations/<JOB_UUID>/request-information \
  -H 'Content-Type: application/json' \
  -d '{"comment":"<edited-or-draft-text>"}'
# → 201 {"id":...,"html_url":...,"body":...}; empty/oversized (>6000B) → 400; stale-scope token → 404 with reconnect hint
# Duplicate posts are allowed by design — the review-then-post UI step is the guard.
```

## 7. Automated checks (after any code change)

```bash
gofmt -l ./cmd ./internal   # expect no output
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
go build ./...
```

## 8. Cleanup

- Test jobs/rows: delete directly in Postgres if you want a clean DB (no delete endpoints exist).
- Never paste real `session_token` or API keys into chat/logs; rotate any exposed key and update it via `PATCH /api/providers/<id> {"api_key":"<NEW>"}`.
