---
status: done
---
# Implement ClusterObjectSlice

## Summary

ClusterObjectSlice is a cluster-scoped content resource that holds raw Kubernetes object manifests externally from a ClusterObjectSet. It exists to decouple large object payloads from the COS, avoiding etcd's 1.5 MiB object size limit for large bundles. The COS retains full control over phase structure, ordering, assertions, and collision protection — slices are pure content stores.

## Design

### ClusterObjectSlice API

ClusterObjectSlice is a pure content store with no status, so `objects` and `ObjectMap` live at the root level (like `data` on ConfigMap/Secret), not under a `spec` wrapper:

```go
// ClusterObjectSlice is a cluster-scoped resource that holds Kubernetes
// object manifests for use by ClusterObjectSet phases via objectRef.
// It has no spec or status — it is a pure content store analogous to
// ConfigMap or Secret.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=cosl
// +kubebuilder:validation:XValidation:rule="self.objects == oldSelf.objects",message="objects is immutable"
type ClusterObjectSlice struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // objects is the list of Kubernetes object manifests stored in this slice.
    // Each entry is keyed by its Kubernetes identity (apiVersion, kind, name,
    // namespace). The list must contain between 1 and 256 entries. Duplicate
    // keys are rejected at admission time.
    // +kubebuilder:validation:MinItems=1
    // +kubebuilder:validation:MaxItems=256
    // +listType=map
    // +listMapKey=apiVersion
    // +listMapKey=kind
    // +listMapKey=name
    // +listMapKey=namespace
    // +required
    Objects   []SliceObject        `json:"objects"`

    // ObjectMap is the in-memory lookup representation of Objects, keyed by
    // ObjectKey. It is populated by the informer cache transform and is never
    // serialized to the wire. Values are raw bytes (possibly gzip-compressed)
    // matching the original Content from the corresponding SliceObject.
    ObjectMap map[ObjectKey][]byte `json:"-"`
}

// ObjectKey uniquely identifies a Kubernetes object by its API identity.
// It is embedded inline in SliceObject and ObjectRef to provide a single
// source of truth for identity fields and their validation.
type ObjectKey struct {
    // apiVersion is the API version of the object (e.g. "v1",
    // "apps/v1", "apiextensions.k8s.io/v1"). The format is an optional
    // DNS-1123 subdomain group followed by "/" and a DNS-1035 version.
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=253
    // +kubebuilder:validation:XValidation:rule=<apiVersionRule>,message="must be a valid API version: optional dns-subdomain group, '/', dns-1035 version"
    // +required
    APIVersion string `json:"apiVersion"`

    // kind is the kind of the object (e.g. "ConfigMap", "Deployment").
    // Must be a DNS-1035 label with mixed case allowed.
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=63
    // +kubebuilder:validation:XValidation:rule=<kindRule>,message="must be a DNS-1035 label (mixed case allowed)"
    // +required
    Kind       string `json:"kind"`

    // name is the metadata.name of the object. Must be a valid DNS-1123
    // subdomain (lowercase alphanumeric, '-', or '.').
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=253
    // +kubebuilder:validation:XValidation:rule=<dns1123SubdomainRule>,message="must be a valid DNS-1123 subdomain"
    // +required
    Name       string `json:"name"`

    // namespace is the metadata.namespace of the object. Defaults to empty
    // string for cluster-scoped resources. Must be a valid DNS-1123 label
    // when non-empty.
    // +kubebuilder:validation:MaxLength=63
    // +kubebuilder:default=""
    // +kubebuilder:validation:XValidation:rule=<dns1123LabelOrEmptyRule>,message="must be empty or a valid DNS-1123 label"
    // +optional
    Namespace  string `json:"namespace,omitempty"`
}

// SliceObject holds a single Kubernetes object manifest with explicit identity
// fields for keyed lookup and admission-time uniqueness validation.
type SliceObject struct {
    ObjectKey `json:",inline"`

    // content is the Kubernetes object manifest as raw JSON, or as
    // gzip-compressed JSON. The format is auto-detected by checking for
    // the gzip magic number (0x1f 0x8b) in the first two bytes.
    // Decompression happens lazily when the object is referenced by a
    // ClusterObjectSet. The []byte type is automatically base64-encoded
    // during JSON serialization by the Kubernetes API machinery — no
    // user-space encoding is needed.
    // +kubebuilder:validation:MinLength=1
    // +required
    Content    []byte `json:"content"`
}
```

Regex patterns (from `k8s.io/apimachinery/pkg/util/validation`):

| Rule placeholder | Canonical regex | Source |
|---|---|---|
| `<dns1123LabelRule>` | `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` | `IsDNS1123Label` |
| `<dns1123SubdomainRule>` | `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` | `IsDNS1123Subdomain` |
| `<dns1035LabelRule>` | `^[a-z]([-a-z0-9]*[a-z0-9])?$` | `IsDNS1035Label` |
| `<kindRule>` | `^[a-zA-Z]([-a-zA-Z0-9]*[a-zA-Z0-9])?$` | `IsDNS1035Label(toLower(kind))` — mixed case |
| `<apiVersionRule>` | `^(<dns1123SubdomainRule>/)?<dns1035LabelRule>$` | group is DNS-1123 subdomain; version is DNS-1035 label |
| `<dns1123LabelOrEmptyRule>` | `self == '' \|\| self.matches(<dns1123LabelRule>)` | empty for cluster-scoped |
```

Each entry has explicit identity fields (the key) and a `content` field holding either raw JSON or gzip-compressed JSON of the Kubernetes manifest. The key fields match ObjectRef exactly, enabling direct lookup without deserializing content.

The `listType=map` with composite `listMapKey` enforces uniqueness of (apiVersion, kind, name, namespace) at admission time — duplicate keys are rejected by the API server.

Content is auto-detected: if the first two bytes match the gzip magic number (`0x1f 0x8b`), the content is decompressed; otherwise it is treated as raw JSON. The `[]byte` type is automatically base64-encoded/decoded during JSON serialization/deserialization by the Kubernetes API machinery — no user-space encoding is needed. Callers work with raw bytes in Go; the wire format is an implementation detail. Compression allows large manifests (e.g., CRDs) to fit within etcd's 1.5 MiB object size limit for the slice itself.

No assertions, collision protection, or phase structure — those belong on the COS.

### Informer transform

The COS controller registers a cache transform that reindexes `Objects` into `ObjectMap` and nils `Objects` to avoid double-storing. The transform does not decompress — content stays as-is (raw JSON or gzip). Decompression happens lazily in the reconciler, only for objects actually referenced by a COS.

```go
mgr, err := ctrl.NewManager(cfg, ctrl.Options{
    Cache: cache.Options{
        ByObject: map[client.Object]cache.ByObject{
            &orbv1alpha1.ClusterObjectSlice{}: {
                Transform: transformClusterObjectSlice,
            },
        },
    },
})

func transformClusterObjectSlice(obj interface{}) (interface{}, error) {
    slice := obj.(*orbv1alpha1.ClusterObjectSlice)
    objectMap := make(map[orbv1alpha1.ObjectKey][]byte, len(slice.Objects))
    for _, so := range slice.Objects {
        objectMap[orbv1alpha1.ObjectKey{
            APIVersion: so.APIVersion,
            Kind:       so.Kind,
            Name:       so.Name,
            Namespace:  so.Namespace,
        }] = so.Content
    }
    slice.ObjectMap = objectMap
    slice.Objects = nil // wire format not needed; cache is read-only
    return slice, nil
}
```

### Object resolution in the reconciler

The reconciler fetches slices from the cache (`ObjectMap` already populated by transform) and resolves objectRefs with direct O(1) map lookups:

```go
func (r *COSReconciler) resolvePhaseObjects(
    ctx context.Context, cos *orbv1alpha1.ClusterObjectSet,
) ([]resolvedPhase, string, error) {
    var hashInputs [][]byte
    var phases []resolvedPhase

    for _, p := range cos.Spec.Phases {
        rp := resolvedPhase{name: p.Name}

        for _, po := range p.Objects {
            var raw []byte

            switch {
            case po.ObjectRef != nil:
                ref := po.ObjectRef
                slice := &orbv1alpha1.ClusterObjectSlice{}
                if err := r.client.Get(ctx, client.ObjectKey{Name: ref.SliceName}, slice); err != nil {
                    return nil, "", fmt.Errorf(
                        "phase %q: fetching slice %q: %w", p.Name, ref.SliceName, err)
                }
                content, ok := slice.ObjectMap[ref.ObjectKey]
                if !ok {
                    return nil, "", fmt.Errorf(
                        "phase %q: object %s %s/%s not found in slice %q",
                        p.Name, ref.Kind, ref.Namespace, ref.Name, ref.SliceName)
                }
                // Lazy decompress: only for objects actually referenced
                if len(content) >= 2 && content[0] == 0x1f && content[1] == 0x8b {
                    var err error
                    raw, err = decompressGzip(content)
                    if err != nil {
                        return nil, "", fmt.Errorf(
                            "phase %q: decompress %s %s/%s from slice %q: %w",
                            p.Name, ref.Kind, ref.Namespace, ref.Name, ref.SliceName, err)
                    }
                } else {
                    raw = content
                }

            default:
                raw = po.Object.Raw
            }

            obj := &unstructured.Unstructured{}
            if err := json.Unmarshal(raw, &obj.Object); err != nil {
                return nil, "", fmt.Errorf("phase %q: unmarshal: %w", p.Name, err)
            }
            rp.objects = append(rp.objects, resolvedObject{obj: obj, phaseObject: po})
            hashInputs = append(hashInputs, raw)
        }
        phases = append(phases, rp)
    }

    h := sha256.New()
    for _, input := range hashInputs {
        h.Write(input)
    }
    return phases, hex.EncodeToString(h.Sum(nil)), nil
}
```

### PhaseObject: mutually exclusive object vs objectRef

`PhaseObject` gains an `objectRef` field that is mutually exclusive with the existing `object` field. Exactly one must be set:

```go
// PhaseObject wraps a single Kubernetes object with optional collision
// protection and availability assertions. The object is specified either
// inline (via object) or by reference to a ClusterObjectSlice entry (via
// objectRef). Exactly one must be set.
//
// +kubebuilder:validation:ExactlyOneOf=object;objectRef
type PhaseObject struct {
    // object is the inline Kubernetes resource manifest to create or update.
    // Mutually exclusive with objectRef.
    // +optional
    Object              runtime.RawExtension   `json:"object,omitempty"`

    // objectRef identifies a Kubernetes object stored in a ClusterObjectSlice.
    // The COS controller resolves the reference at reconcile time by looking
    // up the named slice and matching the object by its identity fields.
    // Mutually exclusive with object.
    // +optional
    ObjectRef           *ObjectRef             `json:"objectRef,omitempty"`

    // collisionProtection overrides the phase-level collision protection
    // setting for this specific object. When omitted, the phase-level setting
    // applies. This field applies identically whether the object is inline
    // or referenced via objectRef.
    // +optional
    CollisionProtection *CollisionProtection   `json:"collisionProtection,omitempty"`

    // assertions define conditions that must be met before this object is
    // considered available. This field applies identically whether the object
    // is inline or referenced via objectRef.
    // +kubebuilder:validation:MaxItems=16
    // +optional
    Assertions          []Assertion            `json:"assertions,omitempty"`
}
```

`ObjectRef` identifies a single object within a ClusterObjectSlice by its natural Kubernetes identity:

```go
// ObjectRef identifies a single object within a ClusterObjectSlice by its
// natural Kubernetes identity. The embedded ObjectKey fields (apiVersion,
// kind, name, namespace) must match a SliceObject entry in the named slice.
type ObjectRef struct {
    // sliceName is the metadata.name of the ClusterObjectSlice resource
    // containing the referenced object.
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=253
    // +kubebuilder:validation:XValidation:rule=<dns1123SubdomainRule>,message="must be a valid DNS-1123 subdomain"
    // +required
    SliceName  string `json:"sliceName"`

    ObjectKey `json:",inline"`
}
```

The COS controller resolves the ref by fetching the named ClusterObjectSlice and finding the object whose apiVersion, kind, name, and namespace match.

### COS controller changes

Two methods need updates:

1. **`managedObjectsForCOS`** — currently iterates only over inline `PhaseObject.Object` entries to discover GVKs for the managed cache. Must also resolve `objectRef` entries by fetching the referenced slice and extracting the object.

2. **`buildRevisionWithSiblings`** — currently calls `objectFromRawExtension(o.Object)` for every phase object. Must branch: if `o.Object` is set, use it directly (as today); if `o.ObjectRef` is set, fetch the ClusterObjectSlice, find the matching object, and use it.

Both methods need a way to fetch ClusterObjectSlice resources, so the reconciler needs a client/reader for slices. The controller should watch ClusterObjectSlice resources so that COS reconciliation re-triggers when a referenced slice is created (a COS may be created before the slices it references exist).

### Error handling and conditions

Slice resolution failures surface through the existing `Available` condition with `Reason=InvalidRevision`. The condition status depends on whether a previous resolution succeeded:

| Scenario | resolvedContentHash | Available | Reason |
|---|---|---|---|
| First resolution fails (slice not found, object missing) | empty | `False` | `InvalidRevision` |
| Previously resolved, now slice deleted | set | `Unknown` | `InvalidRevision` |
| Previously resolved, hash mismatch (delete+recreate) | set | `Unknown` | `InvalidRevision` |

The rationale: when `resolvedContentHash` is already set, the COS previously reconciled its objects successfully. A subsequent resolution failure means we can't re-verify cluster state, but the managed objects may still be healthy — hence `Unknown` rather than `False`. When `resolvedContentHash` is empty, no objects have ever been reconciled, so the revision is genuinely invalid (`False`).

In all cases the controller short-circuits before reaching the boxcutter engine. The condition message describes the specific failure.

### Ownership model

Per ADR-0001: "The caller creates ClusterObjectSlices. The COD and COS controllers only resolve refs — they never create, own, or manage ClusterObjectSlices." The COS controller reads slices but does not set owner references on them.

### COD controller

No changes needed. The COD template uses `ClusterObjectDeploymentTemplateSpec` which inlines `phases: []Phase`. Since Phase contains PhaseObjects, and PhaseObjects now support objectRef, the COD template naturally supports objectRefs via the existing JSON round-trip in `buildCOSFromTemplate`.

### Immutability and content integrity

**ClusterObjectSlice immutability** — a CRD-level CEL rule (`self.objects == oldSelf.objects`) rejects any update to the objects list after creation. Content cannot be modified in place.

**Resolved content hash** — the COS controller detects content substitution (delete+recreate with different content) using a hash-and-lock mechanism in COS status:

1. Every reconcile begins by resolving all objects across all phases (inline objects are used directly; objectRef entries are resolved from ClusterObjectSlices). If resolution fails (slice not found, object not found in slice), reconcile short-circuits with an error condition in status.

2. On the first successful resolution, the controller computes a hash of all resolved objects and stores it in `status.resolvedContentHash`. This field is set-once and immutable:

   ```go
   // +kubebuilder:validation:XValidation:rule="!has(oldSelf.resolvedContentHash) || (has(self.resolvedContentHash) && self.resolvedContentHash == oldSelf.resolvedContentHash)",message="resolvedContentHash is immutable once set"
   ```

3. On subsequent reconciles, the controller resolves objects again and verifies the hash matches `status.resolvedContentHash`. A mismatch (from a delete+recreate of a slice with different content) short-circuits with an error condition.

4. Only after successful resolution and hash verification does the controller proceed with reconcile or teardown.

This covers all integrity cases without external admission infrastructure:
- In-place modification of a slice: blocked by CRD CEL (`self.objects == oldSelf.objects`)
- Delete+recreate with different content: detected by hash mismatch
- Delete without recreate: resolution fails, error condition
- Slices exist and content matches: proceed normally

**PhaseObject objectRef immutability** — objectRef fields are immutable after COS creation, same as `object`, enforced by the existing `self.phases == oldSelf.phases` CEL validation on `ClusterObjectSetSpec`.
