# orb-operator

Phased Kubernetes object management with safe revision transitions.

orb-operator applies and manages sets of Kubernetes resources with phased rollout,
readiness assertions, immutable revision tracking, and drift detection. It is designed
as the object management layer for [OLM v1](https://github.com/operator-framework),
but works standalone for any use case that needs structured, auditable object lifecycle
management.

**[Documentation](https://joelanford.github.io/orb-operator)**

## Install

```bash
kubectl apply -f https://github.com/joelanford/orb-operator/releases/latest/download/operator.yaml
```

Requires Kubernetes 1.30+ (ValidatingAdmissionPolicy support).

## Quick example

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: my-app
spec:
  template:
    spec:
      phases:
        - name: namespace
          objects:
            - object:
                apiVersion: v1
                kind: Namespace
                metadata:
                  name: my-app
              assertions:
                - fieldValue:
                    fieldPath: .status.phase
                    value: Active
        - name: workloads
          objects:
            - object:
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                  name: my-app
                  namespace: my-app
                spec:
                  replicas: 1
                  selector:
                    matchLabels:
                      app: my-app
                  template:
                    metadata:
                      labels:
                        app: my-app
                    spec:
                      containers:
                        - name: app
                          image: my-app:v1
              assertions:
                - fieldsEqual:
                    fieldA: .status.replicas
                    fieldB: .status.readyReplicas
```

## Contributing

### Prerequisites

- Go 1.24+
- Docker (for building images and running e2e tests)

### Clone and build

```bash
git clone https://github.com/joelanford/orb-operator.git
cd orb-operator
make build
```

### Run tests

```bash
make test-unit                # Unit tests (uses envtest)
make test-e2e                 # E2E tests (creates a kind cluster)
make test-all                 # Both, with merged coverage
```

### Run locally

Deploy to a local kind cluster:

```bash
make run
```

This builds the image, creates a kind cluster, loads the image, and deploys the operator.

### Other targets

| Target | Description |
|--------|-------------|
| `make lint` | Run golangci-lint |
| `make lint-fix` | Run golangci-lint with `--fix` |
| `make generate` | Regenerate CRDs, deepcopy, and apply configurations |
| `make verify` | Lint + verify generated files are up to date + build |
| `make tidy` | Run `go mod tidy` |

## License

Apache License 2.0
