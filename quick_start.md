# Camper Vane — Quick Start

Run the cost-aware LLM gateway locally: Go API on `:8080`, React UI on `:5173` (Vite proxies `/api` → backend).

---

## 1. Prerequisites

| Software | Minimum | Notes |
| :--- | :--- | :--- |
| **Go** | 1.22+ | Verified with 1.25+ |
| **Node.js** | 18+ | Verified with 22+ |
| **npm** | 9+ | Bundled with Node |
| **Git** | 2.30+ | |
| **PostgreSQL** *(optional)* | 14+ | Only if using `DATABASE_URL` |
| **Docker** *(optional)* | 24+ | `docker compose` path below; Compose V2 plugin |
| **GitHub CLI (`gh`)** *(optional)* | 2+ | Issue/epic automation script |

---

## 2. Environment variables

Copy [`.env.example`](.env.example) for a production-shaped template (placeholders only). Full matrix and go-live checks: [`docs/ops.md`](docs/ops.md).

### Core

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Go HTTP listen port |
| `DATABASE_PATH` | `camper_vane.db` | SQLite file (ignored when `DATABASE_URL` is set) |
| `DATABASE_URL` | _(empty)_ | PostgreSQL DSN, e.g. `postgres://user:pass@localhost:5432/camper_vane?sslmode=disable` |
| `APP_ENV` | `development` | Set `production` to enforce JWT secret + secure defaults |
| `JWT_SECRET` | built-in dev secret | **Required** non-default value when `APP_ENV=production` |
| `COOKIE_SECURE` | `true` in prod | Set `true` behind HTTPS |
| `COOKIE_SAMESITE` | `lax` | `lax`, `strict`, or `none` (`none` forces Secure; needed for cross-site cookies) |
| `CORS_ALLOWED_ORIGINS` | _(empty)_ | Exact comma-separated origins for credentialed CORS; empty disables CORS; `*` rejected at boot |
| `FRONTEND_URL` | `http://localhost:5173` | Redirect target after real OAuth callback |
| `OAUTH_REDIRECT_URI` | `http://localhost:5173/api/v1/auth/callback` | Must match IdP app config (Vite proxies `/api`) |

### Identity (OAuth)

| Variable | Description |
| :--- | :--- |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth app |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth app |
| `ALLOW_MOCK_AUTH` | `true` / `false`. Default: mock allowed in non-prod when no IdP credentials are set |

### Provider credentials (server-held only)

Users never enter these in the UI.

| Variable | Models |
| :--- | :--- |
| `OPENAI_API_KEY` | `gpt-*` / OpenAI |
| `ANTHROPIC_API_KEY` | `claude-*` |
| `GEMINI_API_KEY` | `gemini-*` |
| `PERPLEXITY_API_KEY` | `sonar*` / `perplexity*` |
| `ALLOW_MOCK_PROVIDERS` | Default `true` outside production. When `false` (or production default), missing keys **fail** instead of silent mock streams |

---

## 3. Install & run

Ship target is **`main`** (tagged [v0.1.0](https://github.com/Senthilsivam41/camper-vane/releases/tag/v0.1.0)). No feature-branch checkout required.

### Clone

```bash
git clone https://github.com/Senthilsivam41/camper-vane.git
cd camper-vane
git checkout main   # or: git checkout v0.1.0
```

### Option A — Local dev (Vite + Go)

#### Backend

Use **three dots** — `go test ./...` runs every package under this module.  
(`go test ./..` is wrong: it points at the parent directory and fails with “no Go files”.)

```bash
go mod tidy
go test ./...
go run ./cmd/server/main.go
```

Server: `http://localhost:8080`.

#### Frontend

```bash
npm --prefix frontend install
npm --prefix frontend run dev
```

UI: `http://localhost:5173` (Vite proxies `/api/*` → `:8080`).

Production UI bundle (for bare-metal / custom reverse proxy):

```bash
npm --prefix frontend run build
```

### Option B — Docker Compose (same-origin UI + API)

Builds the Go API and a Caddy proxy that serves `frontend/dist` and proxies `/api` (SSE-safe). Cookies stay first-party.

```bash
cp .env.example .env   # optional; edit secrets for real OAuth / providers
docker compose up --build
```

Open `http://localhost/` (UI + `/api` on the same origin).  
Details / TLS hostname: [`docs/ops.md`](docs/ops.md) § Container deploy.

### Local happy path (no cloud keys)

1. Start **Option A** (backend + frontend) or **Option B** (Compose).
2. Open the UI → **Continue with local mock auth**.
3. Send a chat prompt → metrics panel shows negotiated model; stream uses mock provider text.
4. **Log out** clears the session cookie.

### Production-like start (local binary)

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
# and/or GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...
export GEMINI_API_KEY=...
export PERPLEXITY_API_KEY=...
go run ./cmd/server/main.go
```

Prefer Compose or a reverse proxy so the SPA and `/api` share one origin (leave `CORS_ALLOWED_ORIGINS` empty).

### PostgreSQL instead of SQLite

```bash
export DATABASE_URL='postgres://user:pass@localhost:5432/camper_vane?sslmode=disable'
go run ./cmd/server/main.go
```

Live store contract test (optional):

```bash
export TEST_DATABASE_URL="$DATABASE_URL"
go test ./internal/db -run Postgres -count=1
```

---

## 4. API cookbook

### Auth status (no side effects)

```bash
curl -s "http://localhost:8080/api/v1/auth/login?intent=status" | jq
```

### Mock login (sets `session_token` cookie)

```bash
curl -i -c cookies.txt -H 'Accept: application/json' \
  "http://localhost:8080/api/v1/auth/callback?code=mock_code&mock_user_id=dev_user_1&format=json"
```

### Real OAuth start (needs IdP credentials)

```bash
curl -s "http://localhost:8080/api/v1/auth/login?provider=google" | jq
# Open auth_url in a browser; callback sets cookie and redirects to FRONTEND_URL
```

### Current user / config

```bash
curl -s -b cookies.txt "http://localhost:8080/api/v1/auth/me" | jq
curl -s -b cookies.txt "http://localhost:8080/api/v1/user/config" | jq
```

### Update preferences

```bash
curl -s -b cookies.txt -X PUT \
  -H "Content-Type: application/json" \
  -d '{
    "daily_token_cap": 75000,
    "routing_strategy": "advanced",
    "preferred_models": ["claude-3-5-sonnet", "gpt-4o", "sonar"]
  }' \
  "http://localhost:8080/api/v1/user/config" | jq
```

### Trailing 24h usage

```bash
curl -s -b cookies.txt "http://localhost:8080/api/v1/user/usage" | jq
```

### Sessions

```bash
curl -s -b cookies.txt "http://localhost:8080/api/v1/sessions" | jq
curl -s -b cookies.txt "http://localhost:8080/api/v1/sessions/demo-session/messages" | jq
```

### Chat stream (SSE)

```bash
curl -N -b cookies.txt -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "demo-session",
    "prompt": "Explain Go channels briefly",
    "model": ""
  }' \
  "http://localhost:8080/api/v1/chat/stream"
```

Expect event sequence: `metrics` → `text`* → `final_usage` (or `error` if provider misconfigured in fail-loud mode).

### Logout

```bash
curl -i -b cookies.txt -X POST "http://localhost:8080/api/v1/auth/logout"
```

---

## 5. OAuth app setup (optional)

1. Create Google and/or GitHub OAuth apps.
2. Set authorized redirect URI to `OAUTH_REDIRECT_URI` (default `http://localhost:5173/api/v1/auth/callback`).
3. Export client ID/secret env vars and restart the Go server.
4. In the UI, use **Sign in with Google** / **Sign in with GitHub**.

---

## 6. Repo automation

Create GitHub epics / stories / milestones:

```bash
python3 scripts/setup_github_issues.py
```

Requires `gh auth login` with `repo` scope.

---

## 7. Related docs

| Doc | Purpose |
| :--- | :--- |
| [README.md](README.md) | Product overview, architecture, story status |
| [docs/ops.md](docs/ops.md) | Production env, cookies, reverse proxy, **container deploy**, go-live checklist |
| [.env.example](.env.example) | Non-secret env template (Compose / production-shaped) |
| [docker-compose.yml](docker-compose.yml) | Same-origin API + Caddy UI |
| [BACKLOG.md](BACKLOG.md) | Prioritized remaining work |
| [Release v0.1.0](https://github.com/Senthilsivam41/camper-vane/releases/tag/v0.1.0) | Current tagged ship |
