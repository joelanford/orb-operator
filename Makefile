IMAGE ?= ghcr.io/joelanford/orb-operator:dev
NAMESPACE ?= orb-operator-system
KIND_CLUSTER ?= orb-operator
PROFILES ?= []
GO_BUILD_FLAGS ?=

.PHONY: lint lint-fix test-unit test-e2e test-coverage test-all build tidy generate verify
.PHONY: run

lint:
	go tool golangci-lint run ./...

lint-fix:
	go tool golangci-lint run --fix ./...

ENVTEST_K8S_VERSION := $(shell go list -m -f '{{.Version}}' k8s.io/api | sed 's/^v0\./1./' | cut -d. -f1-2)
KUBEBUILDER_ASSETS := $(shell go tool setup-envtest use $(ENVTEST_K8S_VERSION) --print path 2>/dev/null)

test-unit:
	rm -rf _output/unit/covdata && mkdir -p _output/unit/covdata
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test -cover -covermode=atomic $(shell go list ./... | grep -v /test/) -args -test.gocoverdir=$(CURDIR)/_output/unit/covdata

test-e2e: KIND_CLUSTER = orb-operator-e2e
test-e2e: PROFILES = ["e2e"]
test-e2e: GO_BUILD_FLAGS = -cover -tags=cover -covermode=atomic
test-e2e: run
	go test ./test/e2e/... -count 1 -v
	./hack/collect-e2e-coverage.sh $(KIND_CLUSTER) $(NAMESPACE) _output/e2e/covdata

test-coverage: test-unit test-e2e
	rm -rf _output/merged/covdata && mkdir -p _output/merged/covdata
	go tool covdata merge -i=_output/unit/covdata,_output/e2e/covdata -o=_output/merged/covdata
	go tool covdata textfmt -i=_output/merged/covdata -o=_output/merged/coverage.out
	go tool cover -func=_output/merged/coverage.out

test-all: test-coverage

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
	GO_BUILD_FLAGS="$(GO_BUILD_FLAGS)" go tool goreleaser release --snapshot --clean
	go tool kind delete cluster --name $(KIND_CLUSTER) || true
	go tool kind create cluster --name $(KIND_CLUSTER)
	go tool kind load docker-image $(IMAGE)-$$(go env GOARCH) --name $(KIND_CLUSTER)
	go tool jsonnet --ext-str image=$(IMAGE)-$$(go env GOARCH) --ext-str namespace=$(NAMESPACE) --ext-code profiles='$(PROFILES)' deploy/main.jsonnet | kubectl apply -f -
	kubectl -n $(NAMESPACE) rollout status deployment/orb-operator --timeout=60s
