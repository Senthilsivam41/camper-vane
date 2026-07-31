# Camper Vane - Quick Start Guide

Camper Vane is an intelligent, cost-aware LLM gateway and interactive web dashboard.

---

## 1. Prerequisites

Ensure the following tools are installed on your system:

| Software | Minimum Version | Verified Version |
| :--- | :--- | :--- |
| **Go** | 1.22+ | 1.25+ |
| **Node.js** | 18.0+ | 22.0+ |
| **npm** | 9.0+ | 11.0+ |
| **Git** | 2.30+ | System default |
| **GitHub CLI (`gh`)** *(Optional)* | 2.0+ | Required only for automated issue/epic creation |

---

## 2. Environment Variables

### Core

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for Go HTTP server |
| `DATABASE_PATH` | `camper_vane.db` | Path to embedded SQLite database file |
| `APP_ENV` | `development` | Use `production` to enforce secrets and secure cookies |
| `JWT_SECRET` | dev default | Required non-default value when `APP_ENV=production` |
| `COOKIE_SECURE` | `true` in prod | Set `true` behind HTTPS |
| `FRONTEND_URL` | `http://localhost:5173` | SPA URL used after OAuth redirect |
| `OAUTH_REDIRECT_URI` | `http://localhost:5173/api/v1/auth/callback` | Must match IdP app settings (Vite proxies `/api`) |

### Identity (OAuth)

| Variable | Description |
| :--- | :--- |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth app credentials |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth app credentials |
| `ALLOW_MOCK_AUTH` | `true`/`false`. Default: mock allowed in non-prod when no IdP credentials are set |

### Provider credentials (server-held only)

Users never enter provider API keys in the UI. Configure secrets on the server:

| Variable | Description |
| :--- | :--- |
| `OPENAI_API_KEY` | OpenAI / GPT models |
| `ANTHROPIC_API_KEY` | Claude models |
| `GEMINI_API_KEY` | Gemini models |
| `ALLOW_MOCK_PROVIDERS` | Default `true` outside production. In production, missing keys fail loudly (no silent mock) |

---

## 3. Installation & Local Execution

### Step 1: Clone Repository
```bash
git clone https://github.com/Senthilsivam41/camper-vane.git
cd camper-vane
git checkout feature/epic-4-frontend-presentation
```

### Step 2: Backend Setup & Execution (Go Core)

```bash
go mod tidy
go test ./... -v
go run ./cmd/server/main.go
```
*Server runs at `http://localhost:8080`.*

Optional production-like start:
```bash
export APP_ENV=production
export JWT_SECRET='replace-with-long-random-secret'
export COOKIE_SECURE=true
export ALLOW_MOCK_AUTH=false
export ALLOW_MOCK_PROVIDERS=false
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...
export GEMINI_API_KEY=...
go run ./cmd/server/main.go
```

### Step 3: Frontend Setup & Execution (React + Vite)

```bash
npm --prefix frontend install
npm --prefix frontend run dev
```
*UI runs at `http://localhost:5173` (API proxied to `:8080`).*

```bash
npm --prefix frontend run build
```

---

## 4. API & Authentication Flow Testing

### Mock auth (local, when IdP credentials are unset):
```bash
curl -i -H 'Accept: application/json' \
  "http://localhost:8080/api/v1/auth/callback?code=mock_code&mock_user_id=dev_user_1&format=json"
```
*Response sets `HttpOnly` cookie `session_token` (`Secure` depends on `COOKIE_SECURE` / `APP_ENV`).*

### Start OAuth login (returns IdP URL when credentials are configured):
```bash
curl -i "http://localhost:8080/api/v1/auth/login?provider=google"
```

### Get User Config:
```bash
curl -i -b "session_token=<JWT_TOKEN_FROM_COOKIE>" "http://localhost:8080/api/v1/user/config"
```

### Update User Preferences:
```bash
curl -i -X PUT \
  -b "session_token=<JWT_TOKEN_FROM_COOKIE>" \
  -H "Content-Type: application/json" \
  -d '{
    "daily_token_cap": 75000,
    "routing_strategy": "advanced",
    "preferred_models": ["claude-3-5-sonnet", "gpt-4o"]
  }' \
  "http://localhost:8080/api/v1/user/config"
```

### Logout:
```bash
curl -i -X POST -b "session_token=<JWT_TOKEN_FROM_COOKIE>" \
  "http://localhost:8080/api/v1/auth/logout"
```

---

## 5. Repository Automation Scripts

To automatically create GitHub Epics, User Stories, and Milestones in your repo:
```bash
python3 scripts/setup_github_issues.py
```
*(Requires `gh auth login` with `repo` scope)*
