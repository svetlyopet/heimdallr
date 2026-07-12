# Heimdallr

Central hub for **compliance artifacts** (SAST, DAST, SBOM, code coverage), **operational automation** (Ansible, AWX, etc.), and **server inventory with agent compliance** (hosts and security/monitoring agents).

## Architecture

```mermaid
flowchart LR
  subgraph compliance [ComplianceTrack]
    App[Application] --> Rel[Release] --> Rep[Report]
  end
  subgraph operations [OperationsTrack]
    Prov[Provider] --> Auto[Automation] --> Job[Job]
  end
  subgraph inventory [InventoryTrack]
    Srv[Server] --> Agt[Agent]
    Srv --> Job
    Srv --> Rel
  end
```

- **Compliance track**: CI pipelines push scan results for a software version (release).
- **Operations track**: Ansible and other automation platforms push job execution results.
- **Inventory track**: Register servers and agents (orphan agents, attach via `agent_ids` or inline `agents` on create/update); associate servers with jobs and releases.

API surface is documented in [`api/docs/openapi.yaml`](api/docs/openapi.yaml) (tags: Application, Release, Report, Provider, Automation, Job, Server, Agent, Analytics, Auth, Token).

## Quick start

### Local (SQLite)

```bash
make run-debug
```

Open http://localhost:8080. On first startup, when no `root` user exists, credentials are logged (generated password or `HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD` if set).

### Postgres (Docker)

```bash
make docker-up
```

This starts Postgres and Heimdallr with `DATABASE_URL=postgres://heimdallr:heimdallr@postgres:5432/heimdallr?sslmode=disable`. Log in with `root` / `e2e-test-password` (set via `HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD` in compose).

```bash
make docker-down   # stop services
```

### Web frontend dev

```bash
make web-install-deps
cd web && npm run dev   # http://127.0.0.1:5173
```

Use `make run-debug` for the API with built assets, or point the Vite dev server at the API as needed.

### Configuration

| Variable / flag | Description |
|-----------------|-------------|
| `DATABASE_URL` | Postgres connection string (preferred for production) |
| `HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD` | Fixed `root` password on first bootstrap (min 12 chars); if unset, a random password is generated and logged |
| `-database-path` | SQLite file path when `DATABASE_URL` is unset (default: `heimdallr.db`) |
| `-server-port` | HTTP port (default: `8080`) |
| `-server-name` | Bind host (default: `localhost`) |
| `-log-format` | `text` or `json` |
| `-log-level` | `debug`, `info`, `warn`, or `error` |

Migrations apply automatically on startup when using Postgres (`make migrate` documents this behavior).

## Authentication

All API routes (except `/api/health` and `POST /api/v1/auth/login`) require a Bearer token.

### Login (UI, scripts, Ansible)

Exchange username and password for a session token:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "root",
  "password": "<password>"
}
```

Response:

```json
{
  "data": {
    "token": "<bearer-token>"
  }
}
```

Use the token on subsequent requests:

```http
Authorization: Bearer <token>
```

### API tokens (CI runners)

Create a long-lived token as admin (using a Bearer token from login):

```http
POST /api/v1/auth/tokens
Authorization: Bearer <admin-token>

{
  "name": "ci-github-actions",
  "scopes": ["application:write", "read"]
}
```

**Scopes**: `application:write`, `automation:write`, `read`, `admin`

## API flows

### Compliance

1. Upsert a release: `POST /api/v1/application/{id}/release?upsert=true`
2. Create report (started): `POST .../release/{release_id}/report`
3. Patch report with results: `PATCH .../report/{report_id}`

See examples:

- [`tests/github-actions-sast-push.yaml`](tests/github-actions-sast-push.yaml)
- [`tests/azure-devops-sbom-push.yaml`](tests/azure-devops-sbom-push.yaml)

### Operations

See [`tests/awx-output-job.yaml`](tests/awx-output-job.yaml).

### Server / agent

1. `POST /api/v1/agent` — create an orphan agent (no server)
2. `POST /api/v1/server` — register a host with `agent_ids` and/or inline `agents`
3. `PUT /api/v1/server/{id}` — attach additional orphan agents
4. Nested routes: `GET` / `DELETE /api/v1/server/{id}/agent/{agent_id}`; global list/create: `GET` / `POST /api/v1/agent`; filter unassigned: `GET /api/v1/agent?unassigned=true`

Go E2E: [`tests/e2e/fleet/`](tests/e2e/fleet/).

## Web UI

Vue 3 SPA served with the API. Main areas:

- Dashboard
- Applications, releases, and reports (compliance)
- Servers and agents (inventory)
- Providers, automations, and jobs (operations)
- Users (admin)

## API documentation

OpenAPI spec: [`api/docs/openapi.yaml`](api/docs/openapi.yaml)

Automation, job, application, release, report, provider, and analytics HTTP handlers are generated from the spec with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) (strict Gin server). Regenerate after spec changes:

```bash
make generate-api
```

## Development

### Git hooks

Enable the repo pre-commit hook (format, lint, secrets scan):

```bash
make setup-hooks
```

On commit, when staged `.go` files are present, the hook runs `make fmt` and `make lint-api`; every commit also runs a `gitleaks` secrets scan. Requires [gitleaks](https://github.com/gitleaks/gitleaks) on your PATH. To disable: `git config --unset core.hooksPath`.

```bash
make generate-api         # Regenerate OpenAPI handlers (automation, job, application, release, report, provider, analytics)
make test                 # Go unit tests
make test-integration     # In-process API integration tests
make lint-api             # golangci-lint
make check-fmt            # verify gofmt/goimports (CI)
make check-generated      # verify OpenAPI codegen is up to date (CI)
make govulncheck          # vulnerability scan (also in CI)
make build                # Web + API binary
make e2e                  # All E2E suites (operations + compliance + fleet)
make e2e-operations       # Ansible job lifecycle
make e2e-compliance       # Release/report push
make e2e-fleet            # Server–agent attach/detach flow
make help                 # Full target list
```

E2E prerequisites: Docker and Ansible (operations flow only). Go E2E tests live under [`tests/e2e/`](tests/e2e/) with shared flows in [`tests/flows/`](tests/flows/); external CI and Ansible templates under [`tests/`](tests/).

## CI

GitHub Actions (push/PR to `main`/`master`) runs unit tests, lint/format/generated checks, SAST (govulncheck + gosec), integration tests, then parallel Docker E2E jobs (operations, compliance, fleet) and authenticated OWASP ZAP DAST.

## Project layout

```
cmd/           API entrypoint
internal/      Domain packages (application, server, agent, job, …)
web/           Vue 3 + Vite frontend
api/docs/      OpenAPI spec and oapi-codegen configs
tests/         Integration tests, E2E scripts, CI/Ansible examples
```

**Stack**: Go 1.26, Gin, GORM, Postgres/SQLite, Vue 3, Vite, samber/do DI.
