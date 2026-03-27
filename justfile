
# Commands for mcp-acronym-lookup
default:
  @just --list
# Build mcp-acronym-lookup with Go
build:
  go build ./...

# Run tests for mcp-acronym-lookup with Go
test:
  go clean -testcache
  go test ./...