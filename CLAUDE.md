# CLAUDE.md - Development & Architectural Guidelines

This repository implements a high-performance **Feature Flag & Multivariate Configuration System** tailored for Design Systems. It features zero-CLS synchronous hydration, deterministic Murmur3 user bucketing, real-time SSE updates, and an in-memory fallback ruleset store.

---

## 🛠️ Essential Commands

### Makefile Shortcuts (Recommended)
```bash
# Start all local dev services (Backend :8080, Admin :3000, Client :3001)
make dev

# Stop running processes and restart dev environment
make restart

# Kill processes listening on ports 8080, 3000, and 3001
make kill

# Run Go backend unit & API integration tests
make test

# Install all Go and Node.js dependencies
make install
```

### Direct Component Commands
```bash
# Go Backend (Port 8080)
cd services/flag-service && go run main.go

# Admin Control Console (Port 3000)
cd apps/admin-portal && npm run dev

# Client Demo Application (Port 3001)
cd apps/client-demo-app && npm run dev
```

---

## 🧪 Test-Driven Development (TDD) Guidelines

1. **Always Follow TDD**: Write or update tests alongside or prior to code implementation.
2. **Go Testing Framework**: Use `github.com/stretchr/testify` (`assert`, `require`) for all Go unit and API integration tests.
3. **HTTP API Isolation**: Test Gin REST handlers using `net/http/httptest.NewRecorder()` to verify status codes, headers, and JSON responses without requiring external network dependencies.
4. **Parity Enforcement**: Maintain exact MurmurHash3 evaluation parity between Go backend engine (`internal/evaluator/engine.go`) and TypeScript SDK (`FeatureFlagContext.tsx`).

---

## 📁 Repository Structure

```
feature-flag-poc/
├── apps/
│   ├── admin-portal/        # React/Vite Admin Control Console (Port 3000)
│   └── client-demo-app/     # React/Vite Design System Demo App & SDK (Port 3001)
├── services/
│   └── flag-service/        # High-performance Go Backend Service (Port 8080)
│       ├── internal/
│       │   ├── api/         # Gin REST Handlers & SSE Streamer
│       │   ├── cache/       # Redis Ruleset Cache
│       │   ├── domain/      # Core Domain Models & Prototypes
│       │   └── evaluator/   # MurmurHash3 Bucketing & Flag Evaluator Engine
│       └── main.go          # HTTP Server & CORS Middleware
├── CLAUDE.md                # Repository Guidelines
└── feature_flag_system_design.md  # Comprehensive Architecture Design Doc
```
