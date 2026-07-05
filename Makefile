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

.PHONY: out-reports
out-reports:
	@mkdir -p $(REPORTS_DIR)

.PHONY: out-api
out-api:
	@mkdir -p $(BIN_DIR)

.PHONY: out-web
out-web:
	@mkdir -p $(WEB_DIST_DIR)

.PHONY: download
download: ## Downloads go dependencies
	@go mod download

.PHONY: tidy
tidy: ## Cleans up go.mod and go.sum
	@go mod tidy

.PHONY: fmt
fmt: ## Formats code with go fmt and goimports
	@go fmt ./...
	@go run golang.org/x/tools/cmd/goimports@latest -w .

.PHONY: govulncheck
govulncheck: ## Vulnerability detection using govulncheck
	@go run golang.org/x/vuln/cmd/govulncheck ./...

coverage: out-reports $(REPORTS_DIR)/report.json ## Displays coverage per func on cli
	@go tool cover -func=$(REPORTS_DIR)/cover.out

html-coverage: out-reports $(REPORTS_DIR)/report.json ## Displays the coverage results in the browser
	@go tool cover -html=$(REPORTS_DIR)/cover.out

test-reports: out-reports $(REPORTS_DIR)/report.json

.PHONY: $(REPORTS_DIR)/report.json
$(REPORTS_DIR)/report.json: out-reports
	@go test -count 1 ./... -coverprofile=$(REPORTS_DIR)/cover.out --json | tee "$(@)"

.PHONY: web-install-deps
web-install-deps: ## Install web dependencies
	cd $(WEB_DIR) && npm install

.PHONY: lint-api
lint-api: fmt download ## Lints all code with golangci-lint
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run

test: ## Run unit tests
	@go test -v -covermode=atomic ./...

.PHONY: build-web
build-web: out-web ## Build web assets
	@cd $(WEB_DIR) && npm run build

.PHONY: build-api
build-api: out-api ## Build api release
	@go build -ldflags="-w -s" -o $(APP_PATH) $(MAIN_PATH)

build: build-web build-api ## Build api and web assets

.PHONY: run-debug
run-debug: build-web ## Run web app in debug mode
	@GIN_MODE=debug go run $(MAIN_PATH) -log-format=$(LOG_FORMAT) -log-level=$(LOG_LEVEL)

.PHONY: run-release
run-release: build-web ## Run web app in release mode
	@GIN_MODE=release go run $(MAIN_PATH) -log-format=$(LOG_FORMAT) -log-level=$(LOG_LEVEL)

DATABASE_URL ?=

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
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''