# Camper Vane

Cost-aware LLM gateway with a React chat UI. A Go proxy authenticates users (OAuth2 / JWT cookie), routes prompts by budget + complexity, and streams responses over SSE from OpenAI, Anthropic, Gemini, or Perplexity.

Provider API keys stay on the server. End users never paste them into the UI.

## Status

| Epic | Focus | Status |
| :--- | :--- | :--- |
| **1** Identity & profiles | Google/GitHub OAuth, JWT `HttpOnly` cookie, user config API + settings UI | Done (mock auth when IdP unset) |
| **2** Proxy & persistence | SQLite / PostgreSQL store, multi-provider SSE (incl. Perplexity) | Done |
| **3** Routing engine | Sliding 24h budget throttle, preferred models, cost deltas, semantic classifier | Done |
| **4** Frontend | Chat UI, metrics panel, sessions, usage on load, SSE auto-retry, logout | Done |

Remaining work is tracked in [`BACKLOG.md`](BACKLOG.md). Local runbook: [`quick_start.md`](quick_start.md). Production ops: [`docs/ops.md`](docs/ops.md). Deploy samples: [`deploy/`](deploy/).

## Stack

| Layer | Technology |
| :--- | :--- |
| Frontend | React + TypeScript + Vite (`frontend/`) |
| Backend | Go (`cmd/server`, `internal/`) |
| Persistence | SQLite by default; PostgreSQL when `DATABASE_URL` is set |
| Streaming | SSE on `POST /api/v1/chat/stream` |

## Architecture

```text
Browser (Vite :5173)
  └─ /api/* proxied ──► Go server (:8080)
                          ├─ auth (OAuth2 + JWT cookie)
                          ├─ user config / usage / sessions
                          ├─ router (simple / advanced)
                          ├─ proxy (OpenAI / Anthropic / Gemini / Perplexity / mock)
                          └─ store (SQLite | Postgres | memory tests)
```

### Auth model
- **Identity:** Google or GitHub OAuth when client credentials are configured.
- **Local/dev:** mock auth when IdP credentials are unset (`ALLOW_MOCK_AUTH` defaults on in development).
- **Session:** JWT in `session_token` cookie (`HttpOnly`; `Secure` in production; `SameSite` via `COOKIE_SAMESITE`).
- **Provider keys:** server env only. Missing keys mock in development; fail loud when `ALLOW_MOCK_PROVIDERS=false` or `APP_ENV=production`.

### Persistence
`db.NewStoreFromEnv()` selects SQLite (`DATABASE_PATH`) or PostgreSQL (`DATABASE_URL`).

### Routing
- Trailing 24h budget throttle at ≥85% of cap
- Simple mode → low-cost preferred models
- Advanced mode → `semantic_heuristic_v2` + preferred premium/low-cost + cost delta

### SSE events (`POST /api/v1/chat/stream`)

| Event | Purpose |
| :--- | :--- |
| `metrics` | Selected model, rationale, cost delta, budget flag |
| `text` | Streaming `text_delta` chunks |
| `final_usage` | Input/output tokens + updated trailing-24h total |
| `error` | Provider/config failure |

## API surface

| Method | Path | Auth | Notes |
| :--- | :--- | :--- | :--- |
| GET | `/api/v1/auth/login?provider=google\|github` | No | Returns IdP URL or mock URL |
| GET | `/api/v1/auth/login?intent=status` | No | Available providers (no side effects) |
| GET/POST | `/api/v1/auth/callback` | No | Code exchange; sets cookie |
| GET | `/api/v1/auth/me` | Yes | Current user config |
| POST | `/api/v1/auth/logout` | No | Clears session cookie |
| GET/PUT | `/api/v1/user/config` | Yes | Daily cap, strategy, preferred models |
| GET | `/api/v1/user/usage` | Yes | Trailing 24h token usage vs cap |
| GET | `/api/v1/sessions` | Yes | List user sessions |
| GET | `/api/v1/sessions/{id}/messages` | Yes | Restore session history |
| POST | `/api/v1/chat/stream` | Yes | SSE chat stream |

## Quick start

```bash
git clone https://github.com/Senthilsivam41/camper-vane.git
cd camper-vane
git checkout main   # or: git checkout v0.1.0

go mod tidy && go test ./...
go run ./cmd/server/main.go

npm --prefix frontend install
npm --prefix frontend run dev
```

Open `http://localhost:5173`. Use **Continue with local mock auth** when OAuth client IDs are not set.

Same-origin Docker path: `docker compose up --build` → `http://localhost/`.

See **[quick_start.md](quick_start.md)** and **[docs/ops.md](docs/ops.md)**.

## User stories & acceptance criteria

### Epic 1: Identity & Profile Foundations

#### #1 OAuth2 Handshake
- [x] `/api/v1/auth/callback` handles token exchange (real IdP or mock)
- [x] Session token stored via `HttpOnly` cookie (`Secure` in production)
- [x] First-time login provisions default profile + daily token cap

#### #2 User Preferences Management API
- [x] `PUT /api/v1/user/config` endpoint
- [x] Validation rejects negative caps / invalid strategies
- [x] Frontend settings UI saves with confirmation

### Epic 2: Proxy Layer & Persistence

#### #3 Pluggable Core Repository
- [x] `UserRepository` / `SessionRepository` (+ `Store`) with SQLite + Memory contract tests
- [x] Env-driven init (`DATABASE_PATH` / `DATABASE_URL`)
- [x] Versioned schema migrations without dropping chat context
- [x] PostgreSQL implementation via `NewStoreFromEnv()`

#### #4 Downstream Multi-Provider SSE Proxying
- [x] JSON request → isolated provider client call
- [x] Streaming parsers for OpenAI, Anthropic, Gemini, Perplexity
- [x] Structured `event: text` (plus `metrics` / `final_usage` / `error`) to the frontend

### Epic 3: Intelligence & Optimization

#### #5 Simple Mode Budget-Aware Router
- [x] Every prompt checks trailing-24h usage
- [x] ≥85% of cap forces low-cost preferred model
- [x] `budget_throttled` flag in `metrics` event

#### #6 Advanced Mode Semantic & Contextual Classifier
- [x] `semantic_heuristic_v2` analytics module (centroids + structural signals)
- [x] High-complexity prompts route to premium preferred models
- [x] Session history hydration influences scoring / topic continuity

### Epic 4: Frontend Presentation

#### #7 Metric Sub-Panel
- [x] Active model badge (provider-colored)
- [x] Trailing-24h usage gauge (loads on open via `/user/usage`)
- [x] Optimization rationale + cost delta from SSE `metrics`

#### #8 Unified SSE Event Consumption Hook
- [x] Sequential event parsing without dropped deltas
- [x] Branches on `metrics` / `text` / `final_usage` / `error`
- [x] Auto-retry with backoff on transient disconnects

## License

See repository for license terms.
