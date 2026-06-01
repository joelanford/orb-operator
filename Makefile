IMAGE ?= ghcr.io/joelanford/orb-operator:dev
NAMESPACE ?= orb-operator-system

.PHONY: lint lint-fix test-unit test-integration test-e2e test-all build tidy generate verify
.PHONY: image kind-cluster kind-cluster-delete kind-load deploy undeploy

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

image:
	go tool goreleaser release --snapshot --clean

kind-cluster:
	go tool kind create cluster

kind-cluster-delete:
	go tool kind delete cluster

kind-load:
	go tool kind load docker-image $(IMAGE)

deploy:
	kubectl apply -f deploy/crds/
	go tool jsonnet --ext-str image=$(IMAGE) --ext-str namespace=$(NAMESPACE) deploy/operator.jsonnet | kubectl apply -f -

undeploy:
	go tool jsonnet --ext-str image=$(IMAGE) --ext-str namespace=$(NAMESPACE) deploy/operator.jsonnet | kubectl delete --ignore-not-found -f -
	kubectl delete --ignore-not-found -f deploy/crds/
