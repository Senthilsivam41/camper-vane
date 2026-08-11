# Camper Vane

Cost-aware LLM gateway with a React chat UI. A Go proxy authenticates users (OAuth2 / JWT cookie), routes prompts by budget + complexity, and streams responses over SSE from OpenAI, Anthropic, Gemini, or Perplexity.

Provider API keys stay on the server. End users never paste them into the UI.

## Status

| Epic | Focus | Status |
| :--- | :--- | :--- |
| **1** Identity & profiles | Google/GitHub OAuth, JWT `HttpOnly` cookie, user config API + settings UI | Done (mock auth when IdP unset) |
| **2** Proxy & persistence | SQLite / PostgreSQL store, multi-provider SSE (incl. Perplexity) | Done |
| **3** Routing engine | Sliding 24h budget throttle, preferred models, cost deltas, richer classifier | Done |
| **4** Frontend | Chat UI, metrics panel, sessions, usage on load, SSE auto-retry, logout | Done |

Remaining work is tracked in [`BACKLOG.md`](BACKLOG.md). Local runbook: [`quick_start.md`](quick_start.md). Production ops: [`docs/ops.md`](docs/ops.md).

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
                          ├─ user config
                          ├─ router (simple / advanced)
                          ├─ proxy (OpenAI / Anthropic / Gemini / Perplexity / mock)
                          └─ store (SQLite | Postgres | memory tests)
```

### Auth model
- **Identity:** Google or GitHub OAuth when client credentials are configured.
- **Local/dev:** mock auth when IdP credentials are unset (`ALLOW_MOCK_AUTH` defaults on in development).
- **Session:** JWT in `session_token` cookie (`HttpOnly`; `Secure` in production).
- **Provider keys:** server env only (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `PERPLEXITY_API_KEY`). Missing keys mock in development; fail loud when `ALLOW_MOCK_PROVIDERS=false` or `APP_ENV=production`.

### Persistence
`db.NewStoreFromEnv()` selects:
- **SQLite** via `DATABASE_PATH` (default `camper_vane.db`)
- **PostgreSQL** via `DATABASE_URL`

Interfaces: `UserRepository`, `SessionRepository`, combined as `Store`. Schema migrations are versioned.

### Routing
- **Sliding 24h budget:** trailing usage from `usage_events`; if ≥ 85% of cap → force low-cost preferred model + `budget_throttled`.
- **Simple mode:** always prefer low-cost entries from `preferred_models`.
- **Advanced mode:** hydrate last N session messages, multi-signal complexity score, pick premium/low-cost from preferences, emit real cost delta vs baseline.

### SSE events (`POST /api/v1/chat/stream`)

| Event | Purpose |
| :--- | :--- |
| `metrics` | Selected model, rationale, cost delta, budget flag |
| `text` | Streaming `text_delta` chunks |
| `final_usage` | Input/output tokens + updated daily total |
| `error` | Provider/config failure (no silent mock in prod) |

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

## Project layout

```text
cmd/server/          HTTP entrypoint
internal/api/        Auth, user, chat handlers
internal/auth/       JWT, OAuth, session cookies
internal/db/         SQLite, Postgres, Memory stores + migrations
internal/proxy/      Provider adapters + credentials policy
internal/router/     Budget + complexity routing
frontend/            React UI (chat, metrics, settings)
BACKLOG.md           Prioritized remaining work
quick_start.md       Install / env / curl cookbook
```

## Quick start

```bash
git clone https://github.com/Senthilsivam41/camper-vane.git
cd camper-vane
git checkout feature/epic-4-frontend-presentation

go mod tidy && go test ./...
go run ./cmd/server/main.go

npm --prefix frontend install
npm --prefix frontend run dev
```

Open `http://localhost:5173`. Use **Continue with local mock auth** when OAuth client IDs are not set.

Full env tables, production flags, Postgres DSN, and curl examples: **[quick_start.md](quick_start.md)**.

## Functional requirements (source of truth)

### Philosophy
- Contract-first FE/BE boundaries
- Zero-trust toward end users for provider keys (server-held secrets)
- Pluggable persistence (SQLite → PostgreSQL)

### Dual-tier optimization
1. **Simple / volumetric:** daily token cap; throttle at ≥85%
2. **Advanced:** session context + complexity classification → premium vs lightweight models

### UI metrics panel
Active model badge, daily usage gauge, routing rationale / estimated cost delta under the prompt box.

### Repository contracts

```go
type UserRepository interface {
    GetUserConfig(ctx context.Context, userID string) (*UserConfig, error)
    UpdateUserConfig(ctx context.Context, config *UserConfig) error
    GetDailyUsage(ctx context.Context, userID string, date time.Time) (int64, error)
    IncrementDailyUsage(ctx context.Context, userID string, date time.Time, tokens int64) error
}

type SessionRepository interface {
    GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error)
    AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error
}
```

## User stories

### Epic 1: Identity & Profile Foundations
- **#1 OAuth2 handshake** — `[x]` callback exchange, HttpOnly cookie (`Secure` in prod), profile provision (mock path for local)
- **#2 User preferences API** — `[x]` `PUT /api/v1/user/config` + settings UI

### Epic 2: Proxy Layer & Persistence
- **#3 Pluggable store** — `[x]` SQLite + Postgres + Memory; env-driven `NewStoreFromEnv()`
- **#4 Multi-provider SSE** — `[x]` OpenAI, Anthropic, Gemini, Perplexity + structured SSE events

### Epic 3: Intelligence & Optimization
- **#5 Simple budget router** — `[x]` trailing-24h usage, ≥85% throttle, `budget_throttled`, preferred low-cost models
- **#6 Advanced classifier** — `[x]` `semantic_heuristic_v2` (domain centroids + multi-signal) + history hydration + cost delta + preferred models

### Epic 4: Frontend Presentation
- **#7 Metrics sub-panel** — `[x]` model badge, 24h usage bar (loads on open), rationale, cost delta
- **#8 SSE hook** — `[x]` `useChatSSE` handles `metrics` / `text` / `final_usage` / `error` with auto-retry; multi-session restore

## License

See repository for license terms.
