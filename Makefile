IMAGE ?= ghcr.io/joelanford/orb-operator:dev
NAMESPACE ?= orb-operator-system
KIND_CLUSTER ?= orb-operator

.PHONY: lint lint-fix test-unit test-e2e test-all build tidy generate verify
.PHONY: run

lint:
	go tool golangci-lint run ./...

lint-fix:
	go tool golangci-lint run --fix ./...

ENVTEST_K8S_VERSION := $(shell go list -m -f '{{.Version}}' k8s.io/api | sed 's/^v0\./1./' | cut -d. -f1-2)
KUBEBUILDER_ASSETS := $(shell go tool setup-envtest use $(ENVTEST_K8S_VERSION) --print path 2>/dev/null)

test-unit:
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test $(shell go list ./... | grep -v /test/)

test-e2e: KIND_CLUSTER = orb-operator-e2e
test-e2e: run
	go test ./test/e2e/... -count 1 -v

test-all: test-unit test-e2e

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
