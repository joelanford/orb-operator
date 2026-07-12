# ClusterObjectDeployment

A **ClusterObjectDeployment** (short name: `cod`) is the primary resource you interact with. It declares a set of Kubernetes objects organized into phases and manages their lifecycle through immutable revisions.

## Analogy

A ClusterObjectDeployment is to a ClusterObjectSet what a Deployment is to a ReplicaSet. You update the COD's template, and the controller stamps out a new COS revision.

## Spec

### template

The `template` field defines the content of each revision:

- **`metadata`** — Labels and annotations propagated to each COS revision (max 32 labels, 32 annotations, annotation values up to 256 KiB each)
- **`spec.phases`** — The ordered list of [phases](phases-and-assertions.md) (1–20 phases)
- **`spec.collisionProtection`** — Default [collision protection](collision-protection.md) mode (default: `Prevent`)

### revisionHistoryLimit

Controls how many archived COS resources are retained. Older archived revisions beyond this limit are garbage collected. Defaults to 10. Set to 0 to disable revision history entirely.

```yaml
spec:
  revisionHistoryLimit: 5
```

### progressDeadlineMinutes

Specifies how long the controller waits for a new revision to make progress before reporting `Progressing=False` with reason `ProgressDeadlineExceeded`. Progress is defined as any phase completing successfully. When omitted, no deadline is enforced.

```yaml
spec:
  progressDeadlineMinutes: 10
```

## Status

### Conditions

| Condition | Meaning |
|-----------|---------|
| `Available` | Whether the latest revision's managed objects satisfy their assertions |
| `Progressing` | Whether the latest revision is making forward progress |

### Active revisions

The `activeRevisions` field lists all non-archived COS resources with their conditions. During a revision transition, you'll see both the old and new revision listed here.

## Example

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: my-operator
spec:
  revisionHistoryLimit: 3
  progressDeadlineMinutes: 5
  template:
    metadata:
      labels:
        app.kubernetes.io/name: my-operator
    spec:
      phases:
        - name: crds
          objects:
            - object:
                apiVersion: apiextensions.k8s.io/v1
                kind: CustomResourceDefinition
                metadata:
                  name: widgets.example.com
                # ...
              assertions:
                - conditionEqual:
                    type: Established
                    status: "True"
        - name: controller
          objects:
            - object:
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                  name: my-operator
                  namespace: my-operator-system
                # ...
              assertions:
                - fieldsEqual:
                    fieldA: .status.replicas
                    fieldB: .status.readyReplicas
```

## Naming

COD names are limited to **52 characters**. This is because the controller creates COS resources named `{cod-name}-{revision}`, and Kubernetes resource names are limited to 63 characters. This constraint is enforced by a ValidatingAdmissionPolicy.
