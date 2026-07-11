# camper-vane
Camper Vane is a flexible, plug-and-play middleware layer and interactive chat application designed to keep your AI infrastructure running efficiently within budget boundaries.
## Functional & Technical Requirements:

## 1. System Overview & Core Philosophy

The application is an intelligent, cost-aware LLM gateway and interactive UI designed to optimize token efficiency and orchestrate multi-model execution. By intercepting user queries behind a secure Go proxy, the platform dynamically evaluates query complexity and operational token constraints to downgrade or upgrade downstream model execution (Gemini, Claude, ChatGPT, Perplexity).

### Core Architecture Goals:
* **Contract-First Development:** Rigid, predictable boundaries between the React frontend and Go backend.
* **Zero Trust Key Management:** Entirely driven via secure OAuth2 authentication; no manual provider API keys stored or provided by the end-user.
* **Pluggable Persistence:** Abstracted storage access mapping to SQLite for local lightweight configuration, with explicit interface patterns allowing an instantaneous swap to heavy SQL engines (PostgreSQL) when horizontal scale is required.

---

## 2. Technical Stack Specification

| Layer | Technology | Justification |
| :--- | :--- | :--- |
| **Frontend** | React (TypeScript) + Vite | High-performance state management, efficient component lifecycle for rapid re-renders during high-frequency token streams, clean component isolation. |
| **Backend** | Go (Golang) | Native high-concurrency primitives (goroutines/channels), low memory overhead for continuous proxying, high-speed text parsing, excellent raw throughput. |
| **Database** | Pluggable SQLite | Zero-configuration embedded store, easy snapshot backups, easily swapped via standard Go interfaces. |
| **Communication** | SSE (Server-Sent Events) | Native unidirectional HTTP streaming, lightweight alternative to WebSockets, perfect for incremental LLM text delta updates and metadata emission. |

---

## 3. Detailed Functional Requirements

### 3.1 Seamless Authentication & User Provisioning
* **Requirement:** The system must omit manual input text boxes for downstream AI provider API tokens. 
* **Mechanism:** Integration with standard OAuth2 identity providers (e.g., Google, GitHub). Upon successful handshake, the backend establishes an independent secure, HTTP-only cookie-based session token (JWT).
* **Profile Provisioning:** First-time authentication dynamically creates default routing profiles within the pluggable persistence layer.

### 3.2 Dual-Tier Optimization Engine (The Routing Core)
The backend proxy executes routing evaluations based on user preferences toggled within the UI:

#### Tier 1: Simple Mode (Volumetric Throttle)
* Tracks cumulative user token expenditure metrics across a sliding 24-hour window.
* Compares current utilization against a user-defined **Daily Token Cap**.
* If current utilization cross thresholds (e.g., $>85\%$ of cap), subsequent prompts are automatically forced onto ultra-low-cost fast models (e.g., *Gemini 1.5 Flash*, *GPT-4o-Mini*) regardless of architectural complexity.

#### Tier 2: Advanced Mode (Semantic & Session Analytics)
* **Context Hydration:** The backend automatically queries the pluggable store to extract historical session context up to $N$ iterations.
* **Complexity Classification:** The proxy analyzes the combination of the outbound prompt and historical text complexity.
* **Routing Logic:**
  * Complex architectural design, comprehensive bug isolation, multi-variable logic $\rightarrow$ Elevated to premium tiers (*Claude 3.5 Sonnet*, *GPT-4o*).
  * Repetitive data text formatting, simple syntax requests, casual definitions $\rightarrow$ Transparently routed to lightweight execution tiers to maximize cost savings.

### 3.3 Dynamic Chat Interface & Real-time Sub-Panel
* **Requirement:** The React client must present a standard layout optimized for immediate developer insights.
* **Location:** Directly underneath the prompt entry text box sits a dedicated, persistent **Optimization & Metrics Area**.
* **Visual Data Points Required:**
  * **Active Model Badge:** Visually shifts colors and labels to signal exactly which downstream model the proxy has negotiated for the active response turn.
  * **Live Tracking Bar:** A horizontal gauge illustrating historical daily token consumption boundaries.
  * **Optimization Insight Readout:** Clear textual rationale compiled by the routing engine explaining *why* a dynamic upgrade/downgrade event occurred, including estimated financial metrics saved per transaction.

---

## 4. Architectural Interfaces & Data Contracts

### 4.1 Pluggable Database Interface (Go Pattern)
To isolate persistence choices, the Go application strictly consumes storage interactions via explicit structural interfaces:

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

### 4.2 Server-Sent Events (SSE) Wire Protocol
All interactive client execution will flow over an established unidirectional stream via `/api/v1/chat/stream`. The proxy emits discrete structural event wrappers:

1. **Event: `metrics`** (Fired instantly when proxy analysis concludes)
   ```json
   {
     "selected_model": "gemini-1.5-flash",
     "routing_rationale": "Session history signals repetitive content extraction. Downshifted to maximize allocation efficiency.",
     "estimated_cost_delta": "-$0.0021"
   }
   ```
2. **Event: `text`** (Repeated continuously as chunks drop from downstream provider)
   ```json
   {
     "text_delta": "func NewConnectionPool..."
   }
   ```
3. **Event: `final_usage`** (Emitted once the downstream proxy connection cleanly terminates)
   ```json
   {
     "input_tokens_consumed": 142,
     "output_tokens_consumed": 512,
     "updated_daily_total": 45120
   }
   ```

---

## 5. GitHub User Story Framework

Below are the structured, production-grade User Stories ready to be imported into your GitHub Issues tracking board.

### Epic 1: Identity & Profile Foundations (Authentication)

#### User Story #1: OAuth2 Handshake Integration
* **As a** Registered User
* **I want to** authenticate using unified identity providers (Google/GitHub) without inputting separate model API keys
* **So that** I can access a pre-configured routing platform instantly and securely.
* **Acceptance Criteria:**
  * [ ] `/api/v1/auth/callback` handles token exchange safely.
  * [ ] Session token is stored strictly via an `HttpOnly`, `Secure` cookie.
  * [ ] First-time login automatically spins up a baseline SQLite profile entry with a default daily token cap.

#### User Story #2: User Preferences Management API
* **As a** Developer using the platform
* **I want to** adjust my routing rules, model order preferences, and volume limit ceilings via an explicit JSON endpoint
* **So that** the proxy engine knows exactly how to apply optimization decisions to my session.
* **Acceptance Criteria:**
  * [ ] Implements a clean `PUT /api/v1/user/config` endpoint.
  * [ ] Request validation rejects negative token values or missing execution strategies.
  * [ ] Front-end UI saves configuration changes instantly with clear visual confirmations.

---

### Epic 2: Proxy Layer & Persistence (Go Core)

#### User Story #3: Pluggable Core Repository Setup (SQLite Baseline)
* **As a** System Maintainer
* **I want to** define strict Go database access interfaces and instantiate them using an embedded SQLite target
* **So that** the application boots with zero configurations while allowing seamless migrations to PostgreSQL later.
* **Acceptance Criteria:**
  * [ ] Implements `UserRepository` and `SessionRepository` structs passing explicit mock evaluations.
  * [ ] Database connection driver initializes cleanly from a single local environmental path variable.
  * [ ] Schema tracking supports incremental user metric additions without dropping active chat contexts.

#### User Story #4: Downstream Multi-Provider SSE Proxying
* **As an** Active User chatting with the app
* **I want to** receive immediate word-by-word text streaming from the targeted downstream provider
* **So that** I do not suffer latency bottlenecks while the system calculates metrics.
* **Acceptance Criteria:**
  * [ ] Go handler translates incoming JSON requests into an isolated downstream client call.
  * [ ] Implements standard streaming parsing loops for Anthropic, Google, and OpenAI text response envelopes.
  * [ ] Encapsulates ongoing data inside structured `event: text` envelopes delivered seamlessly to the web front-end.

---

### Epic 3: Intelligence & Optimization (Routing Engine)

#### User Story #5: Simple Mode Budget-Aware Router
* **As a** Cost-Conscious User
* **I want the backend proxy to** monitor my current daily token expenditures and automatically force low-cost models if I approach my limits
* **So that** I never exceed my allocated cloud budgets unexpectedly.
* **Acceptance Criteria:**
  * [ ] Every incoming prompt executes a fast database verification call reading current daily usage metrics.
  * [ ] Automatically bypasses advanced classification filters if current usage maps $\ge 85\%$ of the absolute user ceiling.
  * [ ] Injects a specialized budget notification flag inside the outbound `metrics` event.

#### User Story #6: Advanced Mode Semantic & Contextual Classifier
* **As a** Developer handling complex tasks
* **I want the system to** inspect the current question alongside the last five lines of historical discussion context
* **So that** it automatically targets high-performance logic engines for heavy assignments and cost-effective engines for basic edits.
* **Acceptance Criteria:**
  * [ ] Implements a Go analytics module parsing token density and structural syntax markers.
  * [ ] Successfully targets high-capability models if coding patterns, architectural expressions, or complex system keywords are observed.
  * [ ] Dynamically updates session state flags to handle changing topics transparently.

---

### Epic 4: Frontend Presentation (React Interface)

#### User Story #7: Metric Sub-Panel Visual Component Design
* **As an** Analytical User
* **I want to** monitor the active model choice and ongoing data optimizations immediately below the text entry panel
* **So that** I gain complete transparency over how the proxy interprets my prompt complexity.
* **Acceptance Criteria:**
  * [ ] Component displays clear indicators rendering active models (e.g., customized badge colors for Claude, Gemini, GPT).
  * [ ] Displays an accurate visual status bar tracking current volumetric consumption.
  * [ ] Renders the descriptive optimization text parsed from the inbound SSE `metrics` event smoothly.

#### User Story #8: Unified SSE Event Consumption Hook
* **As a** Frontend Engineer
* **I want to** utilize a single custom hook or controller capable of digesting structured Server-Sent Events split by their custom header states (`metrics`, `text`, `final_usage`)
* **So that** state management stays highly responsive during active stream delivery.
* **Acceptance Criteria:**
  * [ ] Custom handler reads events sequentially without dropping concurrent characters.
  * [ ] State machine explicitly branches layout reactions upon receiving `metrics` and `final_usage` structural boundaries.
  * [ ] Gracefully catches backend disconnection errors and triggers automated user-friendly retry states.
llm_router_requirements.md
Displaying llm_router_requirements.md.
