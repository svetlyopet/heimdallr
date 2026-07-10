//go:build tools
// +build tools

package tools

import (
	// https://golangci-lint.run
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	// https://github.com/oapi-codegen/oapi-codegen
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	// https://go.dev/blog/vuln
	_ "golang.org/x/vuln/cmd/govulncheck"
	// https://pkg.go.dev/golang.org/x/tools/cmd/goimports
	_ "golang.org/x/tools/cmd/goimports"
)
