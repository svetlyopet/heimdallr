APP_NAME := heimdallr

BIN_DIR := ./bin
REPORTS_DIR := ./reports
WEB_DIR := ./web
WEB_DIST_DIR := $(WEB_DIR)/dist
WEB_DEPS_DIR := $(WEB_DIR)/node_modules

MAIN_PATH := ./cmd/main.go
APP_PATH := $(BIN_DIR)/$(APP_NAME)

LOG_FORMAT ?= text
LOG_LEVEL ?= info

OAPI_CODEGEN := go tool oapi-codegen
OPENAPI_SPEC := api/docs/openapi.yaml

TEST_DB_MANAGED ?= 1
TEST_POSTGRES_URL ?= postgres://heimdallr:heimdallr@127.0.0.1:5433/heimdallr?sslmode=disable
export TEST_POSTGRES_URL

DATABASE_URL ?=

.PHONY: out-reports
out-reports:
	@mkdir -p $(REPORTS_DIR)

.PHONY: out-api
out-api:
	@mkdir -p $(BIN_DIR)

.PHONY: out-web
out-web:
	@mkdir -p $(WEB_DIST_DIR)

.PHONY: test-web-stub
test-web-stub: out-web
	@test -f $(WEB_DIST_DIR)/index.html || printf '%s\n' '<!DOCTYPE html><html><head><title>Heimdallr</title></head><body></body></html>' > $(WEB_DIST_DIR)/index.html

.PHONY: download
download: ## Downloads go dependencies
	@go mod download

.PHONY: tidy
tidy: ## Cleans up go.mod and go.sum
	@go mod tidy

.PHONY: fmt
fmt: ## Formats code with go fmt and goimports
	@go fmt ./...
	@go tool goimports -w .

.PHONY: check-fmt
check-fmt: ## Verifies code is gofmt/goimports clean (CI)
	@test -z "$$(gofmt -l . | grep -v '^vendor/' || true)"
	@test -z "$$(go tool goimports -l . | grep -v '^vendor/' || true)"

.PHONY: setup-hooks
setup-hooks: ## Enable local git pre-commit hooks (.githooks)
	@./scripts/setup-hooks.sh

.PHONY: govulncheck
govulncheck: ## Vulnerability detection using govulncheck
	@go tool govulncheck ./...

coverage: out-reports $(REPORTS_DIR)/report.json ## Displays coverage per func on cli
	@go tool cover -func=$(REPORTS_DIR)/cover.out

html-coverage: out-reports $(REPORTS_DIR)/report.json ## Displays the coverage results in the browser
	@go tool cover -html=$(REPORTS_DIR)/cover.out

test-reports: out-reports $(REPORTS_DIR)/report.json

.PHONY: $(REPORTS_DIR)/report.json
$(REPORTS_DIR)/report.json: out-reports test-web-stub test-db-up
	@go test -count 1 ./... -coverprofile=$(REPORTS_DIR)/cover.out --json | tee "$(@)"

.PHONY: web-install-deps
web-install-deps: ## Install web dependencies
	cd $(WEB_DIR) && npm install

.PHONY: generate-automation-api
generate-automation-api: ## Generate automation API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/automation.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-job-api
generate-job-api: ## Generate job API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/job.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-application-api
generate-application-api: ## Generate application API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/application.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-release-api
generate-release-api: ## Generate release API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/release.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-report-api
generate-report-api: ## Generate report API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/report.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-provider-api
generate-provider-api: ## Generate provider API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/provider.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-analytics-api
generate-analytics-api: ## Generate analytics API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/analytics.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-server-api
generate-server-api: ## Generate server API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/server.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-agent-api
generate-agent-api: ## Generate agent API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/agent.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-auth-api
generate-auth-api: ## Generate auth API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/auth.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-token-api
generate-token-api: ## Generate token API from OpenAPI
	@$(OAPI_CODEGEN) -config api/oapi-codegen/token.cfg.yaml $(OPENAPI_SPEC)

.PHONY: generate-api
generate-api: generate-automation-api generate-job-api generate-application-api generate-release-api generate-report-api generate-provider-api generate-analytics-api generate-server-api generate-agent-api generate-auth-api generate-token-api ## Generate all OpenAPI server code

.PHONY: check-generated
check-generated: generate-api ## Verifies generated OpenAPI code is up to date (CI)
	@git diff --exit-code -- internal/*/api/api.gen.go

.PHONY: generate-postman ## Generates the Postman collection from the OpenAPI spec
generate-postman:
	@python3 ./scripts/postprocess-postman-collection.py /tmp/heimdallr_postman_generated.json api/postman_collection.json

.PHONY: lint-api
lint-api: test-web-stub ## Lints all code with golangci-lint
	@go tool golangci-lint run

.PHONY: test-db-up
test-db-up: ## Start ephemeral Postgres for local tests
	@if [ "$(TEST_DB_MANAGED)" = "1" ]; then \
		docker compose -f docker-compose.test.yml up -d --wait; \
	fi

.PHONY: test-db-down
test-db-down: ## Stop ephemeral Postgres used for local tests
	@if [ "$(TEST_DB_MANAGED)" = "1" ]; then \
		docker compose -f docker-compose.test.yml down -v --remove-orphans; \
	fi

test: test-web-stub test-db-up ## Run unit tests
	@go test -race -shuffle=on -count=1 -v -covermode=atomic ./...

.PHONY: test-integration
test-integration: test-web-stub test-db-up ## Run integration tests
	@go test -tags=integration -race -shuffle=on -v -count=1 ./tests/integration/...

.PHONY: e2e-up
e2e-up: ## Start docker-compose stack and wait for health
	@docker compose down -v --remove-orphans 2>/dev/null || true
	@docker compose up --build -d
	@HEIMDALLR_PASSWORD=e2e-test-password ./scripts/wait-for-health.sh

.PHONY: e2e-up-ci
e2e-up-ci: ## Start pre-built CI image (no rebuild)
	@docker compose -f docker-compose.yml -f docker-compose.ci.yml down -v --remove-orphans 2>/dev/null || true
	@docker compose -f docker-compose.yml -f docker-compose.ci.yml up -d
	@HEIMDALLR_PASSWORD=e2e-test-password ./scripts/wait-for-health.sh

.PHONY: e2e-down
e2e-down: ## Stop docker-compose stack
	@docker compose down -v

.PHONY: e2e-down-ci
e2e-down-ci: ## Stop CI docker-compose stack
	@docker compose -f docker-compose.yml -f docker-compose.ci.yml down -v

.PHONY: e2e-operations-run
e2e-operations-run: ## Run operations E2E scripts (stack must already be up)
	@HEIMDALLR_PASSWORD=e2e-test-password ./tests/e2e/operations/run.sh

.PHONY: e2e-compliance-run
e2e-compliance-run: ## Run compliance E2E tests (stack must already be up)
	@HEIMDALLR_PASSWORD=e2e-test-password go test -tags=e2e -v -count=1 ./tests/e2e/compliance/...

.PHONY: e2e-fleet-run
e2e-fleet-run: ## Run fleet E2E tests (stack must already be up)
	@HEIMDALLR_PASSWORD=e2e-test-password go test -tags=e2e -v -count=1 ./tests/e2e/fleet/...

.PHONY: e2e-operations
e2e-operations: e2e-up e2e-operations-run ## Run operations E2E (Ansible job flow)

.PHONY: e2e-compliance
e2e-compliance: e2e-up e2e-compliance-run ## Run compliance E2E (release/report flow)

.PHONY: e2e-fleet
e2e-fleet: e2e-up e2e-fleet-run ## Run fleet E2E

.PHONY: e2e
e2e: e2e-operations e2e-compliance e2e-fleet e2e-down ## Run all E2E suites

.PHONY: build-web
build-web: out-web ## Build web assets
	@cd $(WEB_DIR) && npm run build

.PHONY: build-api
build-api: out-api generate-api ## Build api release
	@go build -ldflags="-w -s" -o $(APP_PATH) $(MAIN_PATH)

build: build-web build-api ## Build api and web assets

.PHONY: run-debug
run-debug: build-web test-db-up ## Run web app in debug mode
	@GIN_MODE=debug DATABASE_URL=$(TEST_POSTGRES_URL) go run $(MAIN_PATH) -log-format=$(LOG_FORMAT) -log-level=$(LOG_LEVEL)

.PHONY: run-release
run-release: build-web test-db-up ## Run web app in release mode
	@GIN_MODE=release DATABASE_URL=$(TEST_POSTGRES_URL) go run $(MAIN_PATH) -log-format=$(LOG_FORMAT) -log-level=$(LOG_LEVEL)

.PHONY: docker-up
docker-up: ## Start Postgres and Heimdallr via docker-compose
	@docker compose up --build -d

.PHONY: docker-down
docker-down: ## Stop docker-compose services
	@docker compose down

.PHONY: migrate
migrate: ## Migrations run automatically on startup (Postgres via DATABASE_URL)
	@echo "Migrations are applied automatically when the server starts with DATABASE_URL set."

clean-deps:	## Cleans up web dependencies
	@rm -rf $(WEB_DEPS_DIR)

clean-web: ## Cleans up web generated assets
	@rm -rf $(WEB_DIST_DIR)

clean-api: ## Cleans up api generated output
	@rm -rf $(BIN_DIR)

clean-reports: ## Cleans up coverage reports
	@rm -rf $(REPORTS_DIR)

clean: clean-api clean-web clean-deps clean-reports ## Cleans up all produced artifacts

help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@echo ''