# Contributing to Heimdallr

Thanks for helping improve Heimdallr. This guide covers the shortest path to a
working development environment and the checks expected before a pull request.

## Prerequisites

For backend and full-stack development:

- Go 1.26.5
- Node.js 22 and npm
- Git and Make

Install these only when you need the related workflow:

- Docker with Compose v2 for PostgreSQL and end-to-end tests
- Ansible for the operations end-to-end suite
- [gitleaks](https://github.com/gitleaks/gitleaks) for the pre-commit hook
- `curl` for local health checks

Go tools such as `goimports`, `oapi-codegen`, `golangci-lint`, and
`govulncheck` are pinned in `go.mod`. You do not need to install them globally.

## Set up the repository

```bash
git clone https://github.com/svetlyopet/heimdallr.git
cd heimdallr
make download
make web-install-deps
make setup-hooks
```

`make setup-hooks` is optional but recommended. It enables the repository's
pre-commit hook, which formats and lints Go changes and runs a gitleaks scan.
Disable it with:

```bash
git config --unset core.hooksPath
```

## Run locally

### SQLite

SQLite is the quickest option and does not require an external database:

```bash
make run-debug
```

The command builds the web application and starts Heimdallr at
[http://localhost:8080](http://localhost:8080). Data is stored in
`heimdallr.db`, which is ignored by Git.

On the first start, Heimdallr creates the `root` user. Set a predictable local
password with:

```bash
HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD=local-development-password make run-debug
```

If the variable is unset, the generated password is printed in the startup log.

### Frontend development

Run the API and Vite dev server in separate terminals:

```bash
# Terminal 1
make run-debug

# Terminal 2
cd web
npm run dev
```

Open [http://127.0.0.1:5173](http://127.0.0.1:5173). Vite proxies `/api`
requests to the API on port 8080.

### PostgreSQL with Docker

```bash
make docker-up
```

This starts PostgreSQL and a production-style Heimdallr container at
[http://localhost:8080](http://localhost:8080). Stop the stack with
`make docker-down`.

The Compose credentials are for local development only:
`root` / `e2e-test-password`.

## Repository map

```text
cmd/                    Application entry point
internal/               Go domains, HTTP middleware, database, and RBAC
web/                    Vue 3 application embedded in the Go binary
api/docs/               OpenAPI specification
api/oapi-codegen/       Per-domain generator configuration
tests/integration/      In-process API integration tests
tests/e2e/              Live Docker-based workflow tests
tests/flows/            HTTP flows shared by integration and E2E tests
scripts/                Development and CI helper scripts
```

Most domains under `internal/` contain their generated API contract plus
hand-written handler, service, and repository code.

## Development guidelines

### Go changes

- Run `make fmt`; it applies both `gofmt` and `goimports`.
- Keep changes within the relevant domain package where possible.
- Add or update tests with behavior changes.
- Use the tools pinned by `go.mod` instead of unrelated global versions.
- Run `make tidy` after changing Go dependencies and commit both `go.mod` and
  `go.sum`.

### API changes

[`api/docs/openapi.yaml`](api/docs/openapi.yaml) is the API source of truth.
Do not edit `internal/*/api/api.gen.go` by hand.

For an API change:

1. Update the OpenAPI specification.
2. Run `make generate-api`.
3. Update the relevant handler, service, repository, and tests.
4. Commit the generated `api.gen.go` changes.
5. Run `make check-generated`.

Keep existing routes and response shapes compatible unless the change
intentionally introduces a breaking API change.

### Database changes

PostgreSQL migrations live in `internal/database/migrations/` and run
automatically at startup. Local SQLite uses GORM auto-migration, so schema
changes must work in both paths. Exercise PostgreSQL through Docker when a
change affects models, constraints, or queries.

### Web changes

Frontend source is under `web/src/`. Use `npm run dev` for live development and
`npm run build` (or `make build-web`) to verify a production build.

The Go binary embeds `web/dist/`. Go-only tests use `make test-web-stub` to
create the minimum required asset, so a full frontend build is not needed for
every backend change.

## Testing

Run checks that match the area you changed. The normal backend baseline is:

```bash
make check-fmt
make lint-api
make test
make test-integration
make govulncheck
```

Also run:

- `make check-generated` after OpenAPI or generated-handler changes.
- `make build-web` after frontend changes.
- `make e2e-compliance` after application, release, or report flow changes.
- `make e2e-operations` after provider, automation, or job flow changes.
- `make e2e-fleet` after server or agent flow changes.
- `make e2e` when a change crosses several domains.

The E2E targets start a fresh Docker stack and remove its volumes. The
operations suite additionally requires Ansible.

For coverage reports:

```bash
make test-reports
make coverage
# or: make html-coverage
```

## Before opening a pull request

1. Keep the change focused and explain its user-visible effect.
2. Add tests for new behavior and regressions.
3. Confirm formatting, lint, unit tests, and relevant integration/E2E suites
   pass.
4. Run `make tidy` and ensure it leaves no unexpected `go.mod` or `go.sum`
   changes.
5. Regenerate committed API code when the OpenAPI specification changes.
6. Do not commit credentials, local databases, generated reports, or build
   output.

CI repeats module, formatting, generated-code, lint, unit, integration,
vulnerability, Docker E2E, and OWASP ZAP checks. A pull request should be ready
for those checks before review.

## Useful commands

Run `make help` for the full list. Common targets:

```bash
make run-debug          # Build the UI and run the API with SQLite
make build              # Build the web app and Go binary
make fmt                # Format Go code and imports
make test               # Unit tests with race detection and shuffled order
make test-integration   # In-process API tests
make lint-api           # golangci-lint
make generate-api       # Regenerate handlers from OpenAPI
make govulncheck        # Scan Go dependencies and call paths
make clean              # Remove local build output and dependencies
```
