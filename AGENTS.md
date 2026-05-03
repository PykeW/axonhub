# AGENTS.md

## Project Name
AxonHub - an all-in-one AI gateway and development platform.

## Overview
AxonHub is a multi-package repository built around an AI request gateway. The Go backend accepts OpenAI-, Anthropic-, Gemini-, and other compatible requests, then routes, transforms, traces, and authorizes them across multiple providers. It also manages users, projects, API keys, shared channels, observability, configuration, and deployment concerns.

The repository also contains a standalone React + TypeScript frontend that serves as the management console. The UI uses TanStack Router and TanStack Query, with feature-oriented modules for dashboards, relay/share-use flows, project pages, and admin tooling. A separate `llm/` Go module holds provider transformation utilities and related helpers.

## Technology Stack
- Language/Runtime: Go 1.26.0, Node.js, TypeScript
- Framework(s): Gin, Ent ORM, gqlgen, Uber FX, React 19, TanStack Router, TanStack Query, Zustand, Tailwind CSS, Vite, Playwright
- Key Dependencies: viper, zap, uuid, pgx, MySQL driver, SQLite, i18next, Radix UI, DnD Kit, React Hook Form, Zod, Framer Motion, OpenTelemetry
- Build Tools: Make, Air, GoReleaser, ESLint, Prettier, Knip, Husky, pnpm

## Project Structure
```text
.
|-- cmd/                  # Backend entry points and generators
|-- conf/                 # Runtime configuration loading and defaults
|-- deploy/               # Helm chart and service scripts
|-- docs/                 # Diagrams and Chinese documentation
|-- examples/             # Sample payloads and usage examples
|-- frontend/             # Standalone React + TypeScript application
|-- integration_test/     # Cross-provider integration coverage
|-- internal/             # Backend services, auth, metrics, tracing, server, Ent models
|-- llm/                  # Separate Go module for LLM transformation helpers
|-- scripts/              # E2E, migration, lint, sync, and utility scripts
|-- Makefile              # Root build/test/generate workflow
|-- README.md             # Main project introduction and usage overview
|-- config.example.yml    # Sample runtime configuration
`-- LICENSE               # Mixed licensing details
```

### Notable backend areas
- `cmd/axonhub/` - backend entry point and CLI wiring
- `cmd/schema/` - configuration schema generator
- `internal/server/` - HTTP server, routes, API handlers, GraphQL, static asset serving
- `internal/server/biz/` - business logic and orchestration services
- `internal/ent/` - Ent schemas, generated entities, and database access
- `internal/log/`, `internal/metrics/`, `internal/tracing/` - observability support
- `internal/authz/`, `internal/scopes/` - authorization and permission rules
- `internal/contexts/` - request/thread/trace context helpers
- `internal/pkg/` - shared utilities and helpers

### Notable frontend areas
- `frontend/src/main.tsx` - application bootstrap and provider setup
- `frontend/src/routes/` - TanStack Router route modules
- `frontend/src/features/` - feature-oriented UI modules
- `frontend/src/components/` - shared layout and UI components
- `frontend/src/gql/` - GraphQL client and generated query layer
- `frontend/src/stores/` - Zustand state management
- `frontend/src/locales/` - i18n resources
- `frontend/src/lib/`, `frontend/src/utils/` - shared utilities and domain helpers

## Key Features
- AI gateway that translates between client SDKs and provider-specific APIs
- OpenAI, Anthropic, and Gemini compatible request flows
- Provider routing, fallback, and load balancing
- Request tracing, request previews, and observability
- RBAC, projects, API keys, and share/use access control
- Multi-database support: SQLite, PostgreSQL, MySQL, TiDB, and hosted variants
- Web management console and deployment packaging for local and Kubernetes environments

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
- `make generate` - generate GraphQL and Ent code
- `make generate-openapi` - generate OpenAPI-related code
- `make generate-schema` - regenerate `config.schema.json`
- `make build-backend` - build `./cmd/axonhub`
- `make build-frontend` - build the frontend and copy assets into `internal/server/static/dist`
- `make build` - build both frontend and backend
- `make test-backend-all` - run `go test ./...` at the repo root and `cd llm && go test ./...`
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
- Regenerate code with `make generate` after Ent or GraphQL schema changes.
- Regenerate configuration schema with `make generate-schema` when config definitions change.
- Run `make test-backend-all` before backend-related changes.
- Run `make e2e-test` for end-to-end verification when UI/API flows change.
- Keep frontend changes aligned with `frontend/src/features`, `frontend/src/routes`, and `frontend/src/gql` conventions.

## Configuration
- Primary runtime configuration is loaded by `conf/conf.go` from `config.yml`, `./conf`, `/etc/axonhub/`, or `$HOME/.config/axonhub/`, with environment variable overrides using the `AXONHUB_` prefix.
- Example runtime settings are in `config.example.yml`.
- Backend defaults include SQLite, port `8090`, and JSON logging.
- Important environment variables include `AXONHUB_SERVER_PORT`, `AXONHUB_DB_DIALECT`, `AXONHUB_DB_DSN`, `AXONHUB_LOG_LEVEL`, `AXONHUB_CACHE_MODE`, `AXONHUB_METRICS_ENABLED`, and `AXONHUB_GC_CRON`.
- Frontend environment examples are in `frontend/.env.example`; `VITE_API_URL` and `VITE_PORT` control the dev proxy and dev server port.
- Playwright test defaults are set in `frontend/playwright.config.ts` via `AXONHUB_ADMIN_EMAIL`, `AXONHUB_ADMIN_PASSWORD`, and `AXONHUB_API_URL`.
- Deployment-related configuration also appears in `.air.toml`, `.goreleaser.yml`, `deploy/helm/`, and `render.yaml`.

## Architecture
AxonHub follows a layered architecture centered on request transformation. The backend starts in `cmd/axonhub/main.go`, loads config from `conf/`, and wires services with Uber FX. HTTP routing and API handling live under `internal/server/`, while database models and migrations are managed by Ent in `internal/ent/`. Observability is handled through logging, metrics, and tracing packages under `internal/log`, `internal/metrics`, and `internal/tracing`.

The frontend is a separate Vite app under `frontend/` that uses TanStack Router for routing, TanStack Query for server state, Zustand for local state, and a feature-based folder structure. A nested `llm/` module contains provider transformation utilities and should be treated as its own Go module for builds and tests.

## Contributing
- Keep backend and frontend changes aligned when adding or modifying provider, channel, or relay behavior.
- Regenerate generated artifacts after schema changes instead of editing generated files by hand.
- Prefer small, focused changes that preserve the request transformation pipeline.
- Follow the existing docs and module boundaries; `llm/` is a separate Go module.
- Review `docs/zh/development/development.md` for detailed development guidance.

## License
The repository uses mixed licensing: the root project is Apache-2.0, while `llm/` is LGPL-3.0. See `LICENSE` for the full licensing overview and any file-specific exceptions.
