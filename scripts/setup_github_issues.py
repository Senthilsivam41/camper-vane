#!/usr/bin/env python3
import os
import subprocess
import json

REPO = "Senthilsivam41/camper-vane"

# Env setup: pop GITHUB_TOKEN so gh uses keyring auth
env = os.environ.copy()
env.pop("GITHUB_TOKEN", None)

def run_gh(args):
    cmd = ["gh"] + args
    res = subprocess.run(cmd, capture_output=True, text=True, env=env)
    if res.returncode != 0:
        print(f"Error running {' '.join(cmd)}:\n{res.stderr}")
    return res.stdout.strip()

print("--- Creating Labels ---")
labels = [
    ("epic", "3E4B9B", "Epic tracking issue"),
    ("feature", "0E8A16", "Feature / User story issue"),
    ("authentication", "D4C5F9", "Identity & Auth"),
    ("go-core", "00ADD8", "Go Proxy & Persistence"),
    ("routing-engine", "FBCA04", "Dual-tier optimizer"),
    ("react-frontend", "61DAFB", "UI components & hooks"),
]

for name, color, desc in labels:
    run_gh(["label", "create", name, "--color", color, "--description", desc, "--repo", REPO, "--force"])

print("--- Creating Milestones ---")
milestones = [
    "Epic 1: Identity & Profile Foundations",
    "Epic 2: Proxy Layer & Persistence",
    "Epic 3: Intelligence & Optimization",
    "Epic 4: Frontend Presentation",
]

for m in milestones:
    run_gh(["api", f"repos/{REPO}/milestones", "-f", f"title={m}"])

print("--- Creating Epic Issues ---")
epics = [
    {
        "title": "[EPIC] Identity & Profile Foundations (Authentication)",
        "milestone": "Epic 1: Identity & Profile Foundations",
        "labels": "epic,authentication,go-core",
        "body": """# Epic 1: Identity & Profile Foundations (Authentication)

## Summary
Establish seamless OAuth2 authentication (Google/GitHub), HttpOnly JWT session cookies, and dynamic SQLite profile provisioning upon first login.

## Features / User Stories
- [ ] #1 OAuth2 Handshake Integration
- [ ] #2 User Preferences Management API
"""
    },
    {
        "title": "[EPIC] Proxy Layer & Persistence (Go Core)",
        "milestone": "Epic 2: Proxy Layer & Persistence",
        "labels": "epic,go-core",
        "body": """# Epic 2: Proxy Layer & Persistence (Go Core)

## Summary
Implement Go backend interfaces for pluggable storage (SQLite baseline) and downstream multi-provider SSE proxying.

## Features / User Stories
- [ ] #3 Pluggable Core Repository Setup (SQLite Baseline)
- [ ] #4 Downstream Multi-Provider SSE Proxying
"""
    },
    {
        "title": "[EPIC] Intelligence & Optimization (Routing Engine)",
        "milestone": "Epic 3: Intelligence & Optimization",
        "labels": "epic,routing-engine,go-core",
        "body": """# Epic 3: Intelligence & Optimization (Routing Engine)

## Summary
Build dynamic prompt routing engine with Simple Mode (volumetric daily token throttle) and Advanced Mode (semantic context & complexity classifier).

## Features / User Stories
- [ ] #5 Simple Mode Budget-Aware Router
- [ ] #6 Advanced Mode Semantic & Contextual Classifier
"""
    },
    {
        "title": "[EPIC] Frontend Presentation (React Interface)",
        "milestone": "Epic 4: Frontend Presentation",
        "labels": "epic,react-frontend",
        "body": """# Epic 4: Frontend Presentation (React Interface)

## Summary
Develop interactive React UI with metric sub-panel for real-time model badge & token usage visual feedback, and SSE event streaming hook.

## Features / User Stories
- [ ] #7 Metric Sub-Panel Visual Component Design
- [ ] #8 Unified SSE Event Consumption Hook
"""
    }
]

epic_issue_urls = []
for epic in epics:
    out = run_gh([
        "issue", "create",
        "--repo", REPO,
        "--title", epic["title"],
        "--milestone", epic["milestone"],
        "--label", epic["labels"],
        "--body", epic["body"]
    ])
    print(f"Created Epic: {out}")
    epic_issue_urls.append(out)

print("--- Creating User Story Feature Issues ---")
stories = [
    {
        "title": "[FEAT] OAuth2 Handshake Integration",
        "milestone": "Epic 1: Identity & Profile Foundations",
        "labels": "feature,authentication,go-core",
        "epic_idx": 0,
        "body": """### User Story
* **As a** Registered User
* **I want to** authenticate using unified identity providers (Google/GitHub) without inputting separate model API keys
* **So that** I can access a pre-configured routing platform instantly and securely.

### Acceptance Criteria
- [ ] `/api/v1/auth/callback` handles token exchange safely.
- [ ] Session token is stored strictly via an `HttpOnly`, `Secure` cookie.
- [ ] First-time login automatically spins up a baseline SQLite profile entry with a default daily token cap.
"""
    },
    {
        "title": "[FEAT] User Preferences Management API",
        "milestone": "Epic 1: Identity & Profile Foundations",
        "labels": "feature,authentication,go-core",
        "epic_idx": 0,
        "body": """### User Story
* **As a** Developer using the platform
* **I want to** adjust my routing rules, model order preferences, and volume limit ceilings via an explicit JSON endpoint
* **So that** the proxy engine knows exactly how to apply optimization decisions to my session.

### Acceptance Criteria
- [ ] Implements a clean `PUT /api/v1/user/config` endpoint.
- [ ] Request validation rejects negative token values or missing execution strategies.
- [ ] Front-end UI saves configuration changes instantly with clear visual confirmations.
"""
    },
    {
        "title": "[FEAT] Pluggable Core Repository Setup (SQLite Baseline)",
        "milestone": "Epic 2: Proxy Layer & Persistence",
        "labels": "feature,go-core",
        "epic_idx": 1,
        "body": """### User Story
* **As a** System Maintainer
* **I want to** define strict Go database access interfaces and instantiate them using an embedded SQLite target
* **So that** the application boots with zero configurations while allowing seamless migrations to PostgreSQL later.

### Acceptance Criteria
- [ ] Implements `UserRepository` and `SessionRepository` structs passing explicit mock evaluations.
- [ ] Database connection driver initializes cleanly from a single local environmental path variable.
- [ ] Schema tracking supports incremental user metric additions without dropping active chat contexts.
"""
    },
    {
        "title": "[FEAT] Downstream Multi-Provider SSE Proxying",
        "milestone": "Epic 2: Proxy Layer & Persistence",
        "labels": "feature,go-core,routing-engine",
        "epic_idx": 1,
        "body": """### User Story
* **As an** Active User chatting with the app
* **I want to** receive immediate word-by-word text streaming from the targeted downstream provider
* **So that** I do not suffer latency bottlenecks while the system calculates metrics.

### Acceptance Criteria
- [ ] Go handler translates incoming JSON requests into an isolated downstream client call.
- [ ] Implements standard streaming parsing loops for Anthropic, Google, and OpenAI text response envelopes.
- [ ] Encapsulates ongoing data inside structured `event: text` envelopes delivered seamlessly to the web front-end.
"""
    },
    {
        "title": "[FEAT] Simple Mode Budget-Aware Router",
        "milestone": "Epic 3: Intelligence & Optimization",
        "labels": "feature,routing-engine,go-core",
        "epic_idx": 2,
        "body": """### User Story
* **As a** Cost-Conscious User
* **I want the backend proxy to** monitor my current daily token expenditures and automatically force low-cost models if I approach my limits
* **So that** I never exceed my allocated cloud budgets unexpectedly.

### Acceptance Criteria
- [ ] Every incoming prompt executes a fast database verification call reading current daily usage metrics.
- [ ] Automatically bypasses advanced classification filters if current usage maps >= 85% of the absolute user ceiling.
- [ ] Injects a specialized budget notification flag inside the outbound `metrics` event.
"""
    },
    {
        "title": "[FEAT] Advanced Mode Semantic & Contextual Classifier",
        "milestone": "Epic 3: Intelligence & Optimization",
        "labels": "feature,routing-engine,go-core",
        "epic_idx": 2,
        "body": """### User Story
* **As a** Developer handling complex tasks
* **I want the system to** inspect the current question alongside the last five lines of historical discussion context
* **So that** it automatically targets high-performance logic engines for heavy assignments and cost-effective engines for basic edits.

### Acceptance Criteria
- [ ] Implements a Go analytics module parsing token density and structural syntax markers.
- [ ] Successfully targets high-capability models if coding patterns, architectural expressions, or complex system keywords are observed.
- [ ] Dynamically updates session state flags to handle changing topics transparently.
"""
    },
    {
        "title": "[FEAT] Metric Sub-Panel Visual Component Design",
        "milestone": "Epic 4: Frontend Presentation",
        "labels": "feature,react-frontend",
        "epic_idx": 3,
        "body": """### User Story
* **As an** Analytical User
* **I want to** monitor the active model choice and ongoing data optimizations immediately below the text entry panel
* **So that** I gain complete transparency over how the proxy interprets my prompt complexity.

### Acceptance Criteria
- [ ] Component displays clear indicators rendering active models (e.g., customized badge colors for Claude, Gemini, GPT).
- [ ] Displays an accurate visual status bar tracking current volumetric consumption.
- [ ] Renders the descriptive optimization text parsed from the inbound SSE `metrics` event smoothly.
"""
    },
    {
        "title": "[FEAT] Unified SSE Event Consumption Hook",
        "milestone": "Epic 4: Frontend Presentation",
        "labels": "feature,react-frontend",
        "epic_idx": 3,
        "body": """### User Story
* **As a** Frontend Engineer
* **I want to** utilize a single custom hook or controller capable of digesting structured Server-Sent Events split by their custom header states (`metrics`, `text`, `final_usage`)
* **So that** state management stays highly responsive during active stream delivery.

### Acceptance Criteria
- [ ] Custom handler reads events sequentially without dropping concurrent characters.
- [ ] State machine explicitly branches layout reactions upon receiving `metrics` and `final_usage` structural boundaries.
- [ ] Gracefully catches backend disconnection errors and triggers automated user-friendly retry states.
"""
    }
]

for s in stories:
    epic_url = epic_issue_urls[s["epic_idx"]]
    full_body = s["body"] + f"\n\nPart of Epic: {epic_url}"
    out = run_gh([
        "issue", "create",
        "--repo", REPO,
        "--title", s["title"],
        "--milestone", s["milestone"],
        "--label", s["labels"],
        "--body", full_body
    ])
    print(f"Created Story: {out}")

print("--- Done ---")
