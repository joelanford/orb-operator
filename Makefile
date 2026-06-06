IMAGE ?= ghcr.io/joelanford/orb-operator:dev
NAMESPACE ?= orb-operator-system
KIND_CLUSTER ?= orb-operator

.PHONY: lint lint-fix test-unit test-integration test-e2e test-all build tidy generate verify
.PHONY: run

lint:
	go tool golangci-lint run ./...

lint-fix:
	go tool golangci-lint run --fix ./...

test-unit:
	go test $(shell go list ./... | grep -v /test/)

test-integration:
	go test ./test/integration/...

test-e2e: KIND_CLUSTER = orb-operator-e2e
test-e2e: run
	go test ./test/e2e/... -count 1 -v

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

run: generate
	go tool goreleaser release --snapshot --clean
	go tool kind delete cluster --name $(KIND_CLUSTER) || true
	go tool kind create cluster --name $(KIND_CLUSTER)
	go tool kind load docker-image $(IMAGE)-$$(go env GOARCH) --name $(KIND_CLUSTER)
	go tool jsonnet --ext-str image=$(IMAGE)-$$(go env GOARCH) --ext-str namespace=$(NAMESPACE) deploy/operator.jsonnet | kubectl apply -f -
	kubectl -n $(NAMESPACE) rollout status deployment/orb-operator --timeout=60s
