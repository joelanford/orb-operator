# Mission

orb-operator is a Kubernetes operator that manages phased, safe rollout of extension objects (CRDs, Deployments, RBAC, webhooks, etc.) for OLM v1. It implements the orchestration and content layers from ADR-0001: ClusterObjectDeployment, ClusterObjectSet, and ClusterObjectSlice.

## Goals

- **Phased rollout with readiness gates** — apply resources in dependency order across phases, waiting for per-object assertions to pass before proceeding.
- **Safe revision transitions** — transfer object ownership between revisions within a group without gaps or conflicts, keeping both revisions active until the new one succeeds.
- **Immutable revision records** — each ClusterObjectSet is an auditable, point-in-time snapshot of what was applied.
- **Large bundle support** — ClusterObjectSlice decouples object manifests from the COS, avoiding etcd size limits.
- **Independent usability** — the orchestration layer (COD/COS) is usable without the resolution layer. Users and controllers can create ClusterObjectSets directly.

## Non-Goals

- **Catalog resolution** — ClusterExtension and ClusterCatalog are owned by the OLM v1 resolution layer, not this operator.
- **Multi-tenancy / namespace-scoped variants** — all resources are cluster-scoped. Namespace-scoped equivalents are out of scope.

## Design Principles

- **ADR-driven** — follow the ADR-0001 architecture faithfully. COD/COS/ClusterObjectSlice separation, phased rollout, and collision protection work as designed.
- **Composable layers** — each layer (resolution, orchestration, content) must be independently useful and testable. The orchestration layer has no knowledge of catalogs or packages.
- **Standard controller patterns** — use idiomatic controller-runtime patterns: reconcile loops, owner references, status conditions, finalizers. No custom frameworks.
- **Template metadata is the extension point** — caller-specific metadata flows through `template.metadata` on the COD, not through new spec fields.

## Development Practices

- **Testing** — primary testing via godog BDD e2e scenarios against a kind cluster; unit tests with testify for pure logic.
- **Linting and formatting** — golangci-lint and gofumpt are mandatory. `go vet` runs as part of the lint pipeline.
- **CI gating** — all PRs must pass tests, lint, and build before merge. GitHub Actions enforces this.
- **ADR compliance** — changes to the API or controller behavior must be consistent with ADR-0001. If a change requires deviating from the ADR, update the ADR first.
