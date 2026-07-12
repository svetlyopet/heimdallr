# Heimdallr

Heimdallr gives engineering and operations teams one place to see software
compliance results, automation runs, and server inventory. Existing CI pipelines
and automation tools push their results through the API; Heimdallr organizes
them into searchable records and dashboard summaries.

## What you can track

- **Software compliance** — group SAST, DAST, SBOM, code coverage, and custom
  reports by application and release.
- **Automation activity** — record jobs from Ansible, AWX, or another automation
  system, including status, output, and estimated cost savings.
- **Servers and agents** — maintain an inventory of hosts and the security or
  monitoring agents installed on them.
- **Cross-system context** — associate servers with releases and automation jobs
  to understand where software ran and what changed it.
- **Operational summaries** — view compliance success rates and automation
  outcomes from the dashboard.
- **Controlled API access** — use admin and reader accounts for people, and
  scoped tokens for CI or automation clients.

Heimdallr collects and presents results; it does not run scanners or automation
jobs itself.

## How it works

```mermaid
flowchart LR
  subgraph sources [Your toolchain]
    CI["CI pipelines<br/>SAST · DAST · SBOM · coverage"]
    Automation["Automation systems<br/>Ansible · AWX · scripts"]
    Operators["Operators<br/>server and agent inventory"]
  end

  API["Heimdallr API<br/>Bearer authentication"]

  subgraph heimdallr [Heimdallr]
    Compliance["Compliance<br/>applications · releases · reports"]
    Operations["Operations<br/>providers · automations · jobs"]
    Fleet["Fleet<br/>servers · agents"]
    Analytics["Dashboard<br/>compliance and job summaries"]
  end

  Storage[("SQLite or PostgreSQL")]
  UI["Web UI"]

  CI --> API
  Automation --> API
  Operators --> API
  API --> Compliance
  API --> Operations
  API --> Fleet
  Compliance --> Analytics
  Operations --> Analytics
  Fleet -. context .-> Compliance
  Fleet -. context .-> Operations
  Compliance --> Storage
  Operations --> Storage
  Fleet --> Storage
  Analytics --> UI
  Storage --> UI
```

The data model follows three connected tracks:

```mermaid
flowchart LR
  Application --> Release --> Report
  Provider --> Automation --> Job
  Server <--> Agent
  Server -. deployed release .-> Release
  Server -. automation history .-> Job
```

## Get started with Docker

Docker Compose starts Heimdallr and PostgreSQL:

```bash
make docker-up
```

Open [http://localhost:8080](http://localhost:8080) and sign in with:

- Username: `root`
- Password: `e2e-test-password`

These credentials are intended for local use. Set
`HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD` to a strong password for any persistent
deployment.

Stop the stack with:

```bash
make docker-down
```

For a source-based setup with SQLite, frontend development, or test commands,
see [CONTRIBUTING.md](CONTRIBUTING.md).

## Common workflows

### Collect release evidence

Create an application once, then have each pipeline:

1. Upsert the release for its version or commit.
2. Create a report when the scan starts.
3. Update the report with its final status, metadata, and output.

Heimdallr accepts `sast`, `dast`, `sbom`, `code_coverage`, and `custom` reports.
Ready-to-adapt examples are available for
[GitHub Actions](tests/github-actions-sast-push.yaml) and
[Azure DevOps](tests/azure-devops-sbom-push.yaml).

### Record automation jobs

Register a provider and automation, then report each job as it starts and
finishes. The job record keeps the result and output alongside its automation
and related servers. See the
[Ansible/AWX example](tests/awx-output-job.yaml).

### Maintain fleet inventory

Register servers with host, operating system, hypervisor, location, and custom
metadata. Agents can be created independently and attached to one or more
servers, making it possible to find unassigned agents and inspect each host's
tooling.

## Web interface

The web UI is included with the API and provides:

- a dashboard for compliance and automation results;
- application, release, and report views;
- provider, automation, and job views;
- server and agent inventory;
- user administration for admins.

API clients can access the same data using Bearer authentication. Long-lived
tokens can be limited to `application:write`, `automation:write`, `read`, or
`admin` scopes.

## API

The [OpenAPI specification](api/docs/openapi.yaml) is the source of truth for
routes, request bodies, responses, and authentication requirements. Apart from
the health check and login endpoint, all routes require a Bearer token.

Example API and integration material:

- [Postman collection](api/postman_collection.json)
- [GitHub Actions SAST push](tests/github-actions-sast-push.yaml)
- [Azure DevOps SBOM push](tests/azure-devops-sbom-push.yaml)
- [Ansible/AWX job reporting](tests/awx-output-job.yaml)

## Configuration

- `DATABASE_URL` — PostgreSQL connection string; omit it to use SQLite.
- `HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD` — initial `root` password (minimum 12
  characters). If unset, a generated password is written to the startup log.
- `-database-path` — SQLite database path; defaults to `heimdallr.db`.
- `-server-name` and `-server-port` — bind address; defaults to
  `localhost:8080`.
- `-log-format` — `text` or `json`.
- `-log-level` — `debug`, `info`, `warn`, or `error`.

Database migrations run automatically when the application starts.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup, generated API code,
development guidelines, and the checks required before opening a pull request.

## License

Heimdallr is available under the [MIT License](LICENSE).
