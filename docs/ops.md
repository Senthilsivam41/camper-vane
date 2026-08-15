# Camper Vane — Operations Guide

Production and local operations reference for auth, persistence, provider keys, cookies, and reverse-proxy deployment.

For a faster local loop, see [`quick_start.md`](../quick_start.md).

---

## 1. Runtime topology

```text
[Browser] → reverse proxy / Vite (:5173) → Go API (:8080) → SQLite or PostgreSQL
                                         ↘ OpenAI / Anthropic / Gemini / Perplexity
```

- **API process:** `go run ./cmd/server/main.go` (or built binary from `cmd/server`)
- **UI:** Vite dev server (proxies `/api` → `:8080`) or static build served behind the same origin as the API
- **Containers:** `docker compose up --build` — Caddy (UI + `/api`) → Go API (see §4b)

Same-origin (or proxied `/api`) is required so the `session_token` cookie is sent with chat/config requests.

---

## 2. Environment variable matrix

### Required in production (`APP_ENV=production`)

| Variable | Purpose |
| :--- | :--- |
| `APP_ENV` | Must be `production` (unset → `development`: default JWT secret + mocks allowed) |
| `JWT_SECRET` | Non-default HMAC secret for session JWTs (server refuses to start with empty/default) |
| `FRONTEND_URL` | Post-login redirect; **must** be your production origin (unset → `http://localhost:5173`) |
| `OAUTH_REDIRECT_URI` | Exact IdP whitelist entry (unset → localhost callback; OAuth breaks if mismatched) |
| At least one IdP pair | `GOOGLE_CLIENT_ID`+`SECRET` and/or `GITHUB_CLIENT_ID`+`SECRET` |
| `ALLOW_MOCK_AUTH` | Must be `false` (prod default already disables mock; never set `true` in prod) |
| `ALLOW_MOCK_PROVIDERS` | Must be `false` so missing provider keys fail loud instead of mock streams |

Cookie/CORS knobs have safe same-origin defaults (`COOKIE_SECURE=true`, `COOKIE_SAMESITE=lax`, CORS off when empty). Set them explicitly for cross-origin FE/API (see §4).

### Identity (OAuth)

| Variable | Notes |
| :--- | :--- |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google Cloud OAuth client |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth App |
| `OAUTH_REDIRECT_URI` | Must **exactly** match IdP app settings. Local default: `http://localhost:5173/api/v1/auth/callback` |
| `FRONTEND_URL` | Post-login redirect target. Local default: `http://localhost:5173` |
| `ALLOW_MOCK_AUTH` | `true` enables mock login; default in development when no IdP credentials are set |

**Google setup**
1. Create OAuth client (Web application).
2. Authorized redirect URI = `OAUTH_REDIRECT_URI`.
3. Export client ID/secret; restart API.

**GitHub setup**
1. Create OAuth App.
2. Authorization callback URL = `OAUTH_REDIRECT_URI`.
3. Export client ID/secret; restart API.

### Persistence

| Variable | Notes |
| :--- | :--- |
| `DATABASE_PATH` | SQLite file path (default `camper_vane.db`) when `DATABASE_URL` unset |
| `DATABASE_URL` | PostgreSQL DSN; when set, SQLite is not used. Example: `postgres://user:pass@localhost:5432/camper_vane?sslmode=disable` |
| `PORT` | HTTP listen port (default `8080`) |

Migrations run automatically on process start (SQLite and Postgres). Fresh/empty databases need no action — schema is created through the latest version.

**Upgrading a pre-v3 database:** migration v3 adds `usage_events` and `session_messages.user_id`. Migration v4 backfills:

- One `usage_events` row per non-zero `daily_usage` day that has no events yet; `created_at` is **UTC noon** on that `date_str` (daily aggregates have no intra-day timestamps).
- Empty `session_messages.user_id` is set only when exactly one row exists in `user_configs`; with zero or multiple users, rows stay empty (multi-user case logs a warning — ownership is not invented).

v4 is idempotent (skips days that already have events; only updates empty `user_id`). Days that already have post-v3 events are not re-derived from `daily_usage`, so a mixed day can still undercount the pre-v3 portion.

### Provider keys (server-held only)

Never collect these in the UI.

| Variable | Models |
| :--- | :--- |
| `OPENAI_API_KEY` | `gpt-*` |
| `ANTHROPIC_API_KEY` | `claude-*` |
| `GEMINI_API_KEY` | `gemini-*` |
| `PERPLEXITY_API_KEY` | `sonar*` / `perplexity*` |
| `ALLOW_MOCK_PROVIDERS` | Dev default `true`; production default `false` |

---

## 3. Session cookie policy

| Attribute | Value |
| :--- | :--- |
| Name | `session_token` |
| HttpOnly | `true` |
| SameSite | `Lax` |
| Secure | `true` when `COOKIE_SECURE=true` or `APP_ENV=production` (unless overridden) |
| Path | `/` |
| TTL | 24 hours |

Checklist:
- [ ] HTTPS termination in front of the API (or end-to-end TLS)
- [ ] `COOKIE_SECURE=true`
- [ ] Browser origin can reach `/api` on the **same site** that set the cookie (Vite proxy or reverse proxy)
- [ ] `JWT_SECRET` rotated and stored in a secret manager

---

## 4. Reverse proxy examples

### Caddy (API + static UI, same origin)

```caddyfile
camper.example.com {
  handle /api/* {
    reverse_proxy localhost:8080 {
      flush_interval -1   # important for SSE
    }
  }
  handle {
    root * /var/www/camper-vane
    try_files {path} /index.html
    file_server
  }
}
```

Build UI with `npm --prefix frontend run build` and publish `frontend/dist`.

### nginx sketch

```nginx
location /api/ {
  proxy_pass http://127.0.0.1:8080;
  proxy_http_version 1.1;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_buffering off;           # important for SSE
  proxy_read_timeout 3600s;
}

location / {
  root /var/www/camper-vane;
  try_files $uri /index.html;
}
```

SSE notes:
- Disable response buffering on `/api/v1/chat/stream`
- Keep long read timeouts
- Prefer HTTP/1.1 to the upstream for streaming

### Cross-origin FE/API (optional)

Prefer same-origin reverse proxy. If the SPA and API must be on different origins:

```bash
export CORS_ALLOWED_ORIGINS='https://app.example.com'
export COOKIE_SAMESITE=none   # forces Secure cookies
export COOKIE_SECURE=true
export FRONTEND_URL='https://app.example.com'
export OAUTH_REDIRECT_URI='https://api.example.com/api/v1/auth/callback'
```

Behavior:
- `CORS_ALLOWED_ORIGINS` empty → no CORS headers (same-origin / proxied `/api` mode)
- Exact origins only (comma-separated). Never use `*` — the process exits on boot with a config error if `*` appears (wildcard disables credentials and invites cookie-auth misconfig)
- Listed origin + credentialed fetch → `Access-Control-Allow-Origin` echoes that origin + `Allow-Credentials: true`
- `OPTIONS` preflight answered with `204`
- `COOKIE_SAMESITE=none` is required for cross-site cookie sends; browsers also require `Secure`

Deploy example configs under `deploy/`:
- [`deploy/Caddyfile`](../deploy/Caddyfile) — bare-metal / VM
- [`deploy/Caddyfile.docker`](../deploy/Caddyfile.docker) — Compose (`api:8080`, UI at `/srv`)
- [`deploy/nginx.conf`](../deploy/nginx.conf)

---

## 4b. Container deploy

Smallest useful path: multi-stage `Dockerfile` + `docker-compose.yml` (Go API + Caddy with baked `frontend/dist`). Same-origin `/api`, SSE buffering off (`flush_interval -1`). Not a full CD/k8s platform.

### Build / run

```bash
# Optional: cp .env.example .env and fill secrets for real OAuth/providers.
# Without .env, Compose uses development defaults (HTTP :80, COOKIE_SECURE=false).
docker compose up --build
```

Open `http://localhost/` (UI) — browser calls `/api/...` on the same origin.

For production-shaped runs, copy [`.env.example`](../.env.example) → `.env` and set the gates in §2 / §6b (`APP_ENV=production`, `JWT_SECRET`, IdP pair, provider keys, HTTPS URLs). Compose sets `DATABASE_PATH=/data/camper_vane.db` (SQLite volume) unless you set `DATABASE_URL`.

Individual image targets:

```bash
docker build --target api -t camper-vane-api:local .
docker build --target proxy -t camper-vane-proxy:local .
```

### TLS

| Mode | How |
| :--- | :--- |
| Local / TLS elsewhere | Default `CAMPER_SITE_ADDRESS=:80` (HTTP). Terminate TLS at a load balancer and keep same-origin `/api`. |
| Caddy automatic HTTPS | `CAMPER_SITE_ADDRESS=camper.example.com` (ports 80+443 published). Set `COOKIE_SECURE=true`, `FRONTEND_URL` / `OAUTH_REDIRECT_URI` to `https://camper.example.com...`. |

Residual: secrets and IdP whitelist remain manual; Compose does not provision cloud certs or a managed database.

---

## 5. Production start template

```bash
export APP_ENV=production
export JWT_SECRET='replace-with-long-random-secret'
export COOKIE_SECURE=true
export ALLOW_MOCK_AUTH=false
export ALLOW_MOCK_PROVIDERS=false
export FRONTEND_URL='https://camper.example.com'
export OAUTH_REDIRECT_URI='https://camper.example.com/api/v1/auth/callback'
export GOOGLE_CLIENT_ID=...
export GOOGLE_CLIENT_SECRET=...
# optional: GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET
export DATABASE_URL='postgres://user:pass@db:5432/camper_vane?sslmode=require'
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...
export GEMINI_API_KEY=...
export PERPLEXITY_API_KEY=...
export PORT=8080

go build -o bin/camper-vane ./cmd/server
./bin/camper-vane
```

---

## 6. Health / smoke checks

```bash
# Auth status
curl -s "$BASE/api/v1/auth/login?intent=status"

# After login cookie present:
curl -s -b cookies.txt "$BASE/api/v1/auth/me"
curl -s -b cookies.txt "$BASE/api/v1/user/usage"
curl -s -b cookies.txt "$BASE/api/v1/user/config"
```

Expect:
- Production boot fails without a real `JWT_SECRET`
- Chat without provider keys returns SSE `error` (not mock text) when mocks are disabled

---

## 6b. Go-live secrets confirmation

Operators must confirm these **outside the repo** (secret manager / host env / IdP console). Placeholders live in [`.env.example`](../.env.example); never commit real values.

| Check | Pass criteria |
| :--- | :--- |
| `APP_ENV` | Exactly `production` in the running process |
| `JWT_SECRET` | Set, non-default; process starts (missing/default → fatal at boot) |
| Mocks | `ALLOW_MOCK_AUTH=false` and `ALLOW_MOCK_PROVIDERS=false` (or rely on prod defaults); status JSON has `"mock": false` |
| IdP whitelist | `OAUTH_REDIRECT_URI` equals Google/GitHub authorized redirect URL character-for-character |
| Public URLs | `FRONTEND_URL` and `OAUTH_REDIRECT_URI` are production HTTPS hosts (not `localhost`) |
| Provider keys | Keys present for every model family you expose; missing key + mocks off → SSE error, not mock text |
| Cookies | After real login, `session_token` is `HttpOnly` + `Secure` on HTTPS |

Boot does **not** validate IdP redirect match or provider key presence — confirm those via IdP console + smoke chat.

---

## 7. CI

| Workflow | Triggers | Checks |
| :--- | :--- | :--- |
| `.github/workflows/go.yml` | push/PR → `main` | `go build`, `go test ./...` |
| `.github/workflows/frontend.yml` | push/PR → `main` (frontend paths) | `npm ci`, lint, typecheck (`npm test`), build |

---

## 8. Related docs

| Doc | Role |
| :--- | :--- |
| [README.md](../README.md) | Product + architecture |
| [quick_start.md](../quick_start.md) | Local install + curl cookbook |
| [`.env.example`](../.env.example) | Non-secret env template + IdP redirect whitelist notes |
| [`Dockerfile`](../Dockerfile) / [`docker-compose.yml`](../docker-compose.yml) | Container deploy (API + Caddy, §4b) |
| [BACKLOG.md](../BACKLOG.md) | Remaining prioritized work |
