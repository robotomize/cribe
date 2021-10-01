//go:build tools
// +build tools

// Package tools includes the list of tools used in the project.
package tools

// $ go generate -tags tools tools/tools.go
// make bootstrap
import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "golang.org/x/tools/cmd/goimports"
)
