# Large Bundles

When your set of managed objects is too large to fit in a single ClusterObjectSet resource (~1.5 MiB etcd limit), use ClusterObjectSlice resources to store the object content externally.

## How it works

1. Create one or more ClusterObjectSlice resources containing your object manifests
2. In your COS phases, use `objectRef` instead of `object` to reference objects in the slices
3. The COS controller resolves the references at reconcile time

## Step 1: Create ClusterObjectSlice resources

Split your objects across multiple slices. Each slice can hold up to 256 objects.

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectSlice
metadata:
  name: my-app-crds
objects:
  - apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    name: widgets.example.com
    content: |
      {
        "apiVersion": "apiextensions.k8s.io/v1",
        "kind": "CustomResourceDefinition",
        "metadata": {"name": "widgets.example.com"},
        "spec": {
          "group": "example.com",
          "names": {"kind": "Widget", "plural": "widgets"},
          "scope": "Namespaced",
          "versions": [{"name": "v1", "served": true, "storage": true,
            "schema": {"openAPIV3Schema": {"type": "object"}}}]
        }
      }
---
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectSlice
metadata:
  name: my-app-workloads
objects:
  - apiVersion: apps/v1
    kind: Deployment
    name: my-app
    namespace: my-app
    content: |
      {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {"name": "my-app", "namespace": "my-app"},
        "spec": {
          "replicas": 1,
          "selector": {"matchLabels": {"app": "my-app"}},
          "template": {
            "metadata": {"labels": {"app": "my-app"}},
            "spec": {"containers": [{"name": "app", "image": "my-app:v1"}]}
          }
        }
      }
```

!!! tip
    The `content` field also accepts gzip-compressed JSON for even more compact storage. The format is auto-detected by the gzip magic number.

## Step 2: Reference from COS phases

Use `objectRef` to point to objects in the slices:

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectSet
metadata:
  name: my-app-1
spec:
  group: my-app
  revision: 1
  lifecycleState: Active
  phases:
    - name: crds
      objects:
        - objectRef:
            sliceName: my-app-crds
            apiVersion: apiextensions.k8s.io/v1
            kind: CustomResourceDefinition
            name: widgets.example.com
          assertions:
            - conditionEqual:
                type: Established
                status: "True"
    - name: workloads
      objects:
        - objectRef:
            sliceName: my-app-workloads
            apiVersion: apps/v1
            kind: Deployment
            name: my-app
            namespace: my-app
          assertions:
            - fieldsEqual:
                fieldA: .status.replicas
                fieldB: .status.readyReplicas
```

## Mixing inline and referenced objects

You can freely mix `object` (inline) and `objectRef` (slice reference) within the same phase:

```yaml
phases:
  - name: setup
    objects:
      - object:                     # Inline: small object
          apiVersion: v1
          kind: Namespace
          metadata:
            name: my-app
      - objectRef:                  # Referenced: large CRD from a slice
          sliceName: my-app-crds
          apiVersion: apiextensions.k8s.io/v1
          kind: CustomResourceDefinition
          name: widgets.example.com
```

## Content integrity

The COS controller computes a SHA-256 hash of all resolved content on first resolution and stores it in `status.resolvedContentHash`. If a ClusterObjectSlice is deleted and recreated with different content, the hash mismatch is detected and the COS reports an error. This prevents accidental content substitution.

## Updating content

Since ClusterObjectSlice objects are immutable, updating content requires:

1. Create a new ClusterObjectSlice with the updated content
2. Create a new COS revision (or update the COD template) that references the new slice
3. The old slice can be cleaned up after the old revision is archived and garbage collected
