.PHONY: lint lint-fix test-unit test-integration test-e2e test-all build tidy generate verify

lint:
	go tool golangci-lint run ./...

lint-fix:
	go tool golangci-lint run --fix ./...

test-unit:
	go test $(shell go list ./... | grep -v /test/)

test-integration:
	go test ./test/integration/...

test-e2e:
	go test ./test/e2e/...

test-all: test-unit test-integration test-e2e

build:
	go build ./...

tidy:
	go mod tidy

generate:
	go generate ./...

verify: lint
	./hack/diff.sh generate
	go tool goreleaser check
	go build ./...
