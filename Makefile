APP_NAME := heimdallr
MAIN_PATH := ./cmd/main.go
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
LOG_FORMAT ?= text
LOG_LEVEL ?= info
WEB_DIR := ./web
PUBLIC_DIR := ./public

.PHONY: out-api
out-api:
	@mkdir -p $(BIN_DIR)

.PHONY: out-web
out-web:
	@mkdir -p $(PUBLIC_DIR)

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

.PHONY: coverage
coverage: $(OUT_DIR)/report.json ## Displays coverage per func on cli
	@go tool cover -func=$(OUT_DIR)/cover.out

.PHONY: html-coverage
html-coverage: $(OUT_DIR)/report.json ## Displays the coverage results in the browser
	@go tool cover -html=$(OUT_DIR)/cover.out

.PHONY: web-install-deps
web-install-deps: ## Install web dependencies
	cd $(WEB_DIR) && npm install

.PHONY: lint-api
lint-api: fmt download ## Lints all code with golangci-lint
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run

test: ## Run unit tests
	@go test -v -covermode=atomic ./...

.PHONY: build-web
build-web: out-web web-install-deps ## Build web assets
	@cd $(WEB_DIR) && npm run build

.PHONY: build-api
build-api: out-api ## Build api release
	@go build -ldflags="-w -s" -o $(BIN_PATH) $(MAIN_PATH)

build: build-web build-api ## Build api and web assets

run-release: ## Run web app in release mode
	@GIN_MODE=release go run $(MAIN_PATH) -log-format=$(LOG_FORMAT) -log-level=$(LOG_LEVEL)

run-debug: ## Run web app in debug mode
	@GIN_MODE=debug go run $(MAIN_PATH) -log-format=$(LOG_FORMAT) -log-level=$(LOG_LEVEL)

clean-web: ## Cleans up web generated assets
	@rm -rf $(PUBLIC_DIR)
	@rm -rf $(WEB_DIR)/node_modules

clean-api: ## Cleans up api generated output
	@rm -rf $(BIN_DIR)

clean: clean-api clean-web ## Cleans up output and release files

help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''