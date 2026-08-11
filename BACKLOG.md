# Camper Vane — Prioritized Implementation Backlog

_Last updated: 2026-07-30_  
_Baseline branch: `feature/epic-4-frontend-presentation` (MVP scaffold complete; `main` not yet merged)_

Priority legend:
- **P0** — Blocks real users / production correctness
- **P1** — Required to match README functional requirements
- **P2** — Hardening, scale, and polish

Effort: **S** ≤1 day · **M** 2–3 days · **L** ≈1 week

---

## P0 — Make it real and shippable

| ID | Task | Why | Effort | Depends on | Done when | Status |
|---|---|---|---|---|---|---|
| **P0-1** | Merge epic work into `main` via PR | App code is only on feature branches; `main` is docs-only | S | — | PR merged; CI green on `main` | In progress (PR) |
| **P0-2** | Real Google + GitHub OAuth2 handshake | Auth is mock-only today; AC for Story #1 unmet | M | P0-1 | `/auth/login` redirects to IdP; `/auth/callback` exchanges code; HttpOnly session cookie set; first login provisions SQLite profile | Done + Exchange unit tests |
| **P0-3** | Production session security | JWT secret hardcoded; `Secure=false` | S | P0-2 | `JWT_SECRET` from env; `Secure=true` when HTTPS; reject empty/default secrets in prod | Done + Secure cookie assertions |
| **P0-4** | Decide & implement provider credential model | Conflicts with “Zero Trust / no end-user API keys”; today needs env keys or mocks | M | P0-1 | Documented model (server-held secrets **or** provider OAuth); providers fail loudly if misconfigured (no silent mock in prod) | Done (server-held; fail-loud tests) |
| **P0-5** | Wire live providers end-to-end | Chat only useful with real streams | M | P0-4 | OpenAI / Anthropic / Gemini stream with real keys; token usage recorded; SSE `metrics` → `text` → `final_usage` verified in UI | Done (SSE `error` on misconfig) |
| **P0-6** | Frontend logout | Backend exists; UI never calls it | S | P0-2 | Logout clears cookie, returns to auth screen, `/auth/me` fails | Done |

---

## P1 — Meet core product requirements

| ID | Task | Why | Effort | Depends on | Done when |
|---|---|---|---|---|---|
| **P1-1** | Honor `preferred_models` in router _(done)_ | Route selection ignored prefs | S | P0-5 | Simple/advanced pick from preference order |
| **P1-2** | Daily usage on page load _(done)_ | Gauge empty until stream ends | S | P0-5 | `GET /api/v1/user/usage` feeds metrics panel |
| **P1-3** | Sliding 24-hour token window _(done)_ | Spec says sliding 24h | M | P1-2 | `usage_events` + `GetUsageSince` drive throttle |
| **P1-4** | Real cost delta estimation _(done)_ | Hardcoded cost strings | M | P0-5 | Cost table vs baseline in `metrics` |
| **P1-5** | Session history restore + multi-session _(done)_ | UI single default session | M | P0-5 | List/open/new session + history restore |
| **P1-6** | SSE disconnect auto-retry _(done)_ | Story #8 AC | S | P0-5 | Backoff retries + retry status in UI |
| **P1-7** | Perplexity provider adapter _(done)_ | Multi-provider SSE gap | M | P0-4 | `GetProviderClient` routes Perplexity/Sonar models |
| **P1-8** | Env / ops documentation _(done)_ | Ops runbook needed | S | P0-2, P0-4 | `docs/ops.md` + quick_start cover OAuth, JWT, keys, DB, cookies, proxy |

---

## P2 — Hardening & scale

| ID | Task | Why | Effort | Depends on | Done when |
|---|---|---|---|---|---|
| **P2-1** | PostgreSQL repository implementation _(done)_ | Pluggable persistence goal | L | P0-1 | `PostgresRepo` + `DATABASE_URL` via `NewStoreFromEnv()` |
| **P2-2** | Stronger advanced-mode classifier _(done)_ | Keyword-only was too weak | L | P1-1 | Semantic centroid + multi-signal `semantic_heuristic_v2` + table tests |
| **P2-3** | Frontend CI (build + lint + unit) _(done)_ | Workflow was Go-only | S | P0-1 | `.github/workflows/frontend.yml` runs npm ci / lint / test / build |
| **P2-4** | Auth + chat integration / e2e tests | Unit tests exist; no full flow coverage | M | P0-2, P0-5 | Automated happy path: login → config → stream → usage |
| **P2-5** | CORS / reverse-proxy production config | Needed if FE/BE on different origins | S | P0-3 | Documented allowed origins; cookies work cross-origin when intended |
| **P2-6** | Close README acceptance checkboxes | Tracking hygiene | S | P0–P1 done | All Story #1–#8 ACs verified and checked off in README / GitHub issues |

---

## Suggested execution order

```text
Week 1
  P0-1 Merge to main
  P0-2 Real OAuth
  P0-3 Session security
  P0-6 Logout UI

Week 2
  P0-4 Credential model decision + impl
  P0-5 Live provider E2E
  P1-8 Ops docs
  P1-2 Usage on load
  P1-1 Preferred models in router

Week 3
  P1-6 SSE auto-retry
  P1-5 Multi-session + history
  P1-4 Cost estimation
  P1-3 Sliding 24h window

Week 4+
  P1-7 Perplexity
  P2-3 Frontend CI
  P2-4 E2E tests
  P2-1 Postgres (if scale needed)
  P2-2 Classifier upgrade
  P2-5 / P2-6 polish & close ACs
```

---

## Mapping to README user stories

| Story | Scaffold | Remaining backlog IDs |
|---|---|---|
| #1 OAuth2 Handshake | Mock | P0-2, P0-3 |
| #2 User Preferences API | Done | P1-1 (consume prefs) |
| #3 Pluggable SQLite Repo | Done | P2-1 (Postgres) |
| #4 Multi-provider SSE | Adapters + mock | P0-4, P0-5, P1-7 |
| #5 Simple Mode Budget Router | Done | P1-3 |
| #6 Advanced Mode Classifier | Done (`semantic_heuristic_v2`) | — |
| #7 Metrics Sub-Panel | Done | P1-2 |
| #8 SSE Consumption Hook | Done | P1-6 |

---

## Out of scope (for now)

- Mobile-native clients
- Multi-tenant org/billing
- Fine-grained RBAC beyond authenticated user
- Provider key entry in the UI (explicitly forbidden by product philosophy)
