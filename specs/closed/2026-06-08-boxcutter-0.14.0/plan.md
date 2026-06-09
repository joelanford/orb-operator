# Implementation Plan

1. **Bump the dependency**
   - `GOFLAGS=-mod=mod go get pkg.package-operator.run/boxcutter@v0.14.0`
   - `go mod tidy`

2. **Migrate the API call site**
   - In `internal/controller/cosr_controller.go`:
     - Rename `buildRevisionWithPreviousOwners` → `buildRevisionWithSiblings`
     - Rename the `previousOwners` parameter → `siblings`
     - Replace the `boxcutter.WithPreviousOwners` slice construction with `boxcutter.WithSiblingOwners(siblings)` call
     - Update both callers (lines ~307 and ~481) to use the new function name

3. **Verify**
   - `go build ./...`
   - `make verify`
   - `make test-unit`
   - `make test-integration`
   - `make test-e2e`
