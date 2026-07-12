# ClusterObjectSlice

A **ClusterObjectSlice** (short name: `cosl`) is a pure content store for Kubernetes object manifests. It has no spec or status — it holds objects that are referenced from ClusterObjectSet phases via `objectRef`.

## When to use

Use ClusterObjectSlice when your bundle of objects is too large to fit inline in a single ClusterObjectSet. Kubernetes has a ~1.5 MiB etcd object size limit, and large sets of CRDs, RBAC rules, or other resources can easily exceed this. ClusterObjectSlice lets you split the content across multiple resources.

See the [Large Bundles](../guides/large-bundles.md) guide for a walkthrough.

## Structure

A ClusterObjectSlice contains a single field:

### objects

A list of 1–256 object entries. Each entry has:

| Field | Description |
|-------|-------------|
| `apiVersion` | API version of the object (e.g., `v1`, `apps/v1`) |
| `kind` | Kind of the object (e.g., `ConfigMap`, `Deployment`) |
| `name` | Name of the object |
| `namespace` | Namespace (empty for cluster-scoped resources) |
| `content` | Raw JSON or gzip-compressed JSON of the full object manifest |

The `content` field accepts either plain JSON or gzip-compressed JSON (auto-detected by the gzip magic number `0x1f 0x8b`). The `[]byte` type is automatically base64-encoded during JSON serialization — no manual encoding is needed.

## Immutability

The `objects` field is **immutable after creation**. To update content, create a new ClusterObjectSlice and update the COS phases to reference it.

## Referencing from a COS phase

In a ClusterObjectSet phase, use `objectRef` instead of `object` to reference content stored in a slice:

```yaml
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
```

The `sliceName` identifies the ClusterObjectSlice, and the remaining fields (`apiVersion`, `kind`, `name`, `namespace`) identify the specific object within that slice.

## Content hash verification

When a COS first resolves its objectRef references, it computes a SHA-256 hash of all resolved content and stores it in `status.resolvedContentHash`. On subsequent reconciles, if the hash no longer matches (because a slice was deleted and recreated with different content), the COS reports an error. This prevents accidental or malicious content substitution.

## Example

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectSlice
metadata:
  name: my-app-crds
objects:
  - apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    name: widgets.example.com
    content: >-
      eyJhcGlWZXJzaW9uIjoiYXBpZXh0ZW5zaW9ucy5rOHMuaW8vdjEiLC...
```
