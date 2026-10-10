# START.MD — Start Issue Refiner from scratch

## 0. Prerequisites

- Go toolchain, Docker running, `psql` client, `curl`, `jq` (optional but used below).

## 1. Start PostgreSQL

```bash
docker compose up -d postgres
docker exec issue-refiner-db pg_isready -U postgres   # expect: accepting connections
```

Defaults (see `docker-compose.yml`): host `localhost:5432`, db `issue_refiner`, user/password `postgres`/`postgres`, data in the `postgres_data` volume.

## 2. Environment file

The server reads `./.env` (fails fast via `log.Fatalf` if required keys are missing). Required keys:

```bash
DB_PASSWORD=postgres
DB_NAME=issue_refiner
GITHUB_CLIENT_ID=<your-oauth-app-id>
GITHUB_CLIENT_SECRET=<your-oauth-app-secret>
GITHUB_CALLBACK_URL=http://localhost:8080/auth/github/callback
CREDENTIAL_KEY=<64-hex-chars>   # 32 bytes, e.g. output of: openssl rand -hex 32
```

Optional (defaults shown): `SERVER_HOST=0.0.0.0` `SERVER_PORT=8080` `DB_HOST=localhost` `DB_PORT=5432` `DB_USER=postgres` `DB_SSLMODE=disable` `WORKER_COUNT=3` `WORKER_QUEUE_SIZE=32` `JOB_TIMEOUT_SECONDS=30`.

## 3. Apply migrations (no runner library — apply raw SQL in order)

```bash
export PGPASSWORD=postgres
for f in migrations/0000*.up.sql; do psql -h localhost -U postgres -d issue_refiner -v ON_ERROR_STOP=1 -f "$f"; done
psql -h localhost -U postgres -d issue_refiner -c '\dt'
# expect: users, sessions, jobs, organizations, organization_members,
#         repositories, github_tokens, provider_configs, provider_models, provider_credentials
```

## 4. Start the server

```bash
# port conflict kills it — check first:
ss -ltnp 2>/dev/null | grep ':8080' || echo "port free"

go run ./cmd/server
# expect: "✓ Database Connected" → "Worker pool started with 3 workers" → "Starting Server on 0.0.0.0:8080"

curl localhost:8080/health   # → {"status":"ok"}
```

## 5. Shutdown

- Server terminal → `Ctrl-C` (drains HTTP → workers → pool, prints "Server stopped").
- `docker compose stop postgres` — only if you started it just for this session.
- Confirm nothing lingers: `ps aux | grep '[e]xe/server'` should be empty.
