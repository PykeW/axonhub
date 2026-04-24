# AGENTS.md

## Project Name
AxonHub - an all-in-one AI gateway and development platform.

## Overview
AxonHub is a multi-app repository that combines a Go backend, a standalone React + TypeScript frontend, and a nested Go module for LLM transformation utilities. Its main purpose is to let clients keep using familiar SDKs and API formats while transparently routing requests to different AI providers. The backend handles configuration, routing, authentication, observability, and provider adaptation, while the frontend provides the management console and user-facing UI.

The project is designed around request transformation and provider compatibility. It supports OpenAI-, Anthropic-, and Gemini-compatible flows, model mapping, load balancing, tracing, API key management, and deployment-oriented configuration for local development and production environments.

## Technology Stack
- Language/Runtime: Go 1.26.0, Node.js with pnpm, TypeScript
- Framework(s): Gin, Ent ORM, gqlgen, Uber FX, React 19, TanStack Router, TanStack Query, Zustand, Tailwind CSS, Vite, Playwright
- Key Dependencies: viper, zap, uuid, pgx, mysql driver, sqlite, i18next, Radix UI, DnD Kit, React Hook Form, Zod, Framer Motion, OpenTelemetry
- Build Tools: Make, Air, GoReleaser, ESLint, Prettier, Knip, Husky

## Project Structure
```text
.
|-- cmd/axonhub/          # Backend entry point and CLI commands
|-- cmd/schema/           # Configuration schema generator
|-- conf/                 # YAML + environment configuration loading
|-- deploy/               # Install/start/stop scripts and Helm assets
|-- docs/                 # Chinese documentation
|-- examples/             # Example API payloads and samples
|-- frontend/             # Standalone React + TypeScript app
|-- integration_test/     # Cross-provider integration tests
|-- internal/             # Backend implementation, Ent models, services, metrics, auth, tracing
|-- llm/                  # Separate Go module for LLM transformers and helpers
|-- scripts/              # E2E, lint, sync, migration, and utility scripts
|-- README.md             # Chinese project introduction and documentation entry point
|-- Makefile              # Root build/test/generate workflow
|-- render.yaml           # Render deployment config
`-- config.example.yml    # Sample runtime configuration
```

### Notable backend areas
- `internal/server/` - HTTP server, routes, API handlers, GraphQL, and static assets
- `internal/server/biz/` - Core business logic and services
- `internal/ent/` - Ent schema, generated entities, and database access
- `internal/log/`, `internal/metrics/`, `internal/tracing/` - observability and runtime support
- `internal/authz/` and `internal/scopes/` - authorization and permission rules
- `internal/contexts/` - request/thread/trace context helpers
- `internal/pkg/` - shared utilities

### Notable frontend areas
- `frontend/src/main.tsx` - app bootstrap
- `frontend/src/routes/` - TanStack Router route modules
- `frontend/src/features/` - feature-oriented UI organization
- `frontend/src/components/` - shared UI and utility components
- `frontend/src/gql/` - GraphQL client integration
- `frontend/src/stores/` - Zustand state management
- `frontend/src/locales/` - i18n resources
- `frontend/src/lib/` and `frontend/src/utils/` - shared helpers and domain utilities

## Key Features
- AI gateway that translates between client SDKs and provider-specific APIs
- OpenAI, Anthropic, and Gemini compatible request flows
- Provider routing, failover, and load balancing
- Request tracing and observability
- RBAC and API key/profile management
- Multi-database support: SQLite, PostgreSQL, MySQL, TiDB, and related hosted variants
- Web management console and deployment-friendly packaging

## Getting Started

### Prerequisites
- Go 1.26.0 or newer
- Node.js 18+ with pnpm
- Git

### Installation
```bash
git clone https://github.com/looplj/axonhub.git
cd axonhub

# Backend dependencies are managed by Go modules at the repo root
# Frontend dependencies are managed separately
cd frontend
pnpm install
```

### Usage
```bash
# Backend development with hot reload
air

# Or build the backend binary
make build-backend
./axonhub

# Frontend development server
cd frontend
pnpm dev

# Full project build (backend + frontend)
make build

# Run root backend tests and llm module tests
make test-backend-all

# Run E2E tests
make e2e-test
```

## Development

### Available Scripts
Root automation is defined in `Makefile`:
- `make generate` - run GraphQL/Ent generation from `internal/server/gql`
- `make generate-openapi` - run OpenAPI generation from `internal/server/gql/openapi`
- `make generate-schema` - regenerate `config.schema.json` from `cmd/schema`
- `make build-backend` - build `./cmd/axonhub`
- `make build-frontend` - build the frontend and copy assets into `internal/server/static/dist`
- `make build` - build both frontend and backend
- `make test-backend-all` - run `go test ./...` at the root and `cd llm && go test ./...`
- `make e2e-test` - run the E2E suite via `scripts/e2e/e2e-test.sh`
- `make lint` - run repository lint checks
- `make lint-privacy` - enforce privacy.DecisionContext usage rules
- `make sync-faq` / `make sync-models` - update synced content from scripts
- `make filter-logs` - analyze load-balancing logs

Frontend scripts live in `frontend/package.json`:
- `pnpm dev`
- `pnpm build`
- `pnpm lint`
- `pnpm lint:fix`
- `pnpm format:check`
- `pnpm format`
- `pnpm knip`
- `pnpm test:e2e` and related Playwright variants

### Development Workflow
- Use `air` or `make build-backend` for backend iteration.
- Use `cd frontend && pnpm dev` for the frontend dev server.
- Regenerate code with `make generate` after schema changes.
- Regenerate configuration schema with `make generate-schema` when config definitions change.
- Run `make test-backend-all` before backend-related changes.
- Run `make e2e-test` for end-to-end verification when UI/API flows change.
- Keep frontend code aligned with `frontend/src/features`, `frontend/src/routes`, and `frontend/src/gql` conventions.

## Configuration
- Primary runtime configuration is loaded by `conf/conf.go` from `config.yml`, `./conf`, `/etc/axonhub/`, or `$HOME/.config/axonhub/`, with environment variable overrides using the `AXONHUB_` prefix.
- Example runtime settings are in `config.example.yml`.
- Backend defaults include SQLite, port `8090`, and JSON logging.
- Important environment variables include `AXONHUB_SERVER_PORT`, `AXONHUB_DB_DIALECT`, `AXONHUB_DB_DSN`, `AXONHUB_LOG_LEVEL`, `AXONHUB_CACHE_MODE`, `AXONHUB_METRICS_ENABLED`, and `AXONHUB_GC_CRON`.
- Frontend environment examples are in `frontend/.env.example`; `VITE_API_URL` and `VITE_PORT` control the dev proxy and server port.
- Playwright test defaults are set in `frontend/playwright.config.ts` via `AXONHUB_ADMIN_EMAIL`, `AXONHUB_ADMIN_PASSWORD`, and `AXONHUB_API_URL`.
- Deployment-related configuration also appears in `render.yaml`, `.air.toml`, and `.goreleaser.yml`.

## Architecture
AxonHub follows a layered architecture centered on request transformation. The backend starts in `cmd/axonhub/main.go`, loads config from `conf/`, and wires services with Uber FX. HTTP routing and API handling live under `internal/server/`, while database models and migrations are managed by Ent in `internal/ent/`. Observability is handled through logging, metrics, and tracing packages under `internal/log`, `internal/metrics`, and `internal/tracing`.

The frontend is a separate Vite application under `frontend/` that uses TanStack Router for routing, TanStack Query for server state, Zustand for local state, and a feature-based folder structure. A nested `llm/` module contains provider transformation utilities and is treated as its own Go module, so Go commands for that code should be run from `llm/`.

## Contributing
- Keep backend and frontend changes aligned when adding or modifying provider/channel behavior.
- Regenerate generated artifacts after schema changes instead of editing generated files by hand.
- Prefer small, focused changes that preserve the request transformation pipeline.
- Follow the existing docs and module boundaries; `llm/` is a separate Go module.
- Review `docs/zh/development/development.md` for detailed development guidance.

## License
The repository uses mixed licensing: the root project is Apache-2.0, while `llm/` is LGPL-3.0. See `LICENSE` for the full licensing overview and any file-specific exceptions.