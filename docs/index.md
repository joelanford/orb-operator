# orb-operator

**Phased Kubernetes object management with safe revision transitions.**

orb-operator is a Kubernetes operator that applies and manages sets of Kubernetes resources with:

- **Phased rollout** — resources are applied in dependency order with readiness gates between phases
- **Safe revision transitions** — ownership transfers from old to new revisions without gaps or conflicts
- **Immutable revision records** — each revision is an auditable snapshot of what was applied
- **Large bundle support** — external content storage via ClusterObjectSlice for bundles exceeding etcd's size limit
- **Drift detection** — completed phases are continuously re-reconciled to detect and correct configuration drift

## How it works

orb-operator provides three custom resources:

| Resource | Short Name | Purpose |
|----------|-----------|---------|
| [ClusterObjectDeployment](concepts/cluster-object-deployment.md) | `cod` | Declares what to deploy — like a Deployment for arbitrary objects |
| [ClusterObjectSet](concepts/cluster-object-set.md) | `cos` | An immutable revision snapshot — like a ReplicaSet |
| [ClusterObjectSlice](concepts/cluster-object-slice.md) | `cosl` | External content storage for large bundles |

You define the objects you want to manage in **phases**. Each phase contains a set of Kubernetes objects and optional **assertions** that must pass before the next phase begins. When you update a ClusterObjectDeployment, the operator creates a new revision and safely transitions ownership of shared objects from the old revision to the new one.

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
                  replicas: 2
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

## Next steps

- [Install orb-operator](getting-started/installation.md) on your cluster
- Follow the [Quick Start](getting-started/quickstart.md) tutorial
- Learn about [core concepts](concepts/overview.md)
