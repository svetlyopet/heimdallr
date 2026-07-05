# Heimdallr

Central hub for **compliance artifacts** (SAST, DAST, SBOM, code coverage) and **operational automation** (Ansible, AWX, etc.).

## Architecture

```
Application → Release → Report     (CI compliance track)
Provider → Automation → Job        (operations track)
```

- **Compliance track**: CI pipelines push scan results for a software version (release).
- **Operations track**: Ansible and other automation platforms push job execution results.

## Quick start

### Local (SQLite)

```bash
make run-debug
```

Open http://localhost:8080 and log in with the bootstrapped `root` credentials printed on first startup.

### Postgres (Docker)

```bash
make docker-up
```

This starts Postgres and Heimdallr with `DATABASE_URL=postgres://heimdallr:heimdallr@postgres:5432/heimdallr?sslmode=disable`.

### Configuration

| Variable / flag | Description |
|-----------------|-------------|
| `DATABASE_URL` | Postgres connection string (preferred for production) |
| `-database-path` | SQLite file path when `DATABASE_URL` is unset (default: `heimdallr.db`) |
| `-server-port` | HTTP port (default: `8080`) |

## Authentication

### Username / password (UI & Ansible)

```http
X-Auth-Username: root
X-Auth-Password: <password>
```

### API tokens (CI runners)

Create a token as admin:

```http
POST /api/v1/auth/tokens
Authorization: Bearer <admin-session-not-supported-yet>
X-Auth-Username: root
X-Auth-Password: <password>

{
  "name": "ci-github-actions",
  "scopes": ["application:write", "read"]
}
```

Use the returned token:

```http
Authorization: Bearer <token>
```

**Scopes**: `application:write`, `automation:write`, `read`, `admin`

## CI push flow (compliance)

1. Upsert a release: `POST /api/v1/application/{id}/release?upsert=true`
2. Create report (started): `POST .../release/{release_id}/report`
3. Patch report with results: `PATCH .../report/{report_id}`

See examples:

- [`tests/github-actions-sast-push.yaml`](tests/github-actions-sast-push.yaml)
- [`tests/azure-devops-sbom-push.yaml`](tests/azure-devops-sbom-push.yaml)

## Ansible push flow (operations)

See [`tests/awx-output-job.yaml`](tests/awx-output-job.yaml).

## API documentation

OpenAPI spec: [`api/docs/openapi.yaml`](api/docs/openapi.yaml)

## Development

```bash
make test          # Go unit tests
make lint-api      # golangci-lint
make build         # Web + API binary
```
