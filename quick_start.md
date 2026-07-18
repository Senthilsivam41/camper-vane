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

The backend Go proxy supports the following optional environment variables:

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for Go HTTP server |
| `DATABASE_PATH` | `camper_vane.db` | Path to embedded SQLite database file |

---

## 3. Installation & Local Execution

### Step 1: Clone Repository & Checkout Branch
```bash
git clone https://github.com/Senthilsivam41/camper-vane.git
cd camper-vane
git checkout feature/epic-1-identity-foundations
```

### Step 2: Backend Setup & Execution (Go Core)

Tidy Go module dependencies:
```bash
go mod tidy
```

Run unit tests:
```bash
go test ./... -v
```

Start backend HTTP server:
```bash
go run ./cmd/server/main.go
```
*Server runs at `http://localhost:8080`.*

### Step 3: Frontend Setup & Execution (React + Vite)

Install node modules:
```bash
npm --prefix frontend install
```

Start Vite dev server:
```bash
npm --prefix frontend run dev
```
*UI runs at `http://localhost:5173`.*

Build production bundle:
```bash
npm --prefix frontend run build
```

---

## 4. API & Authentication Flow Testing

### Authenticate via OAuth Callback:
```bash
curl -i "http://localhost:8080/api/v1/auth/callback?code=mock_code&mock_user_id=dev_user_1"
```
*Response sets `HttpOnly`, `SameSite=Lax` cookie `session_token`.*

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

---

## 5. Repository Automation Scripts

To automatically create GitHub Epics, User Stories, and Milestones in your repo:
```bash
python3 scripts/setup_github_issues.py
```
*(Requires `gh auth login` with `repo` scope)*
