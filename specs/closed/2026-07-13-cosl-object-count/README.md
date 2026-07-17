---
status: done
---
# COSL Object Count and Printer Column

## Summary

Add a `count` field and OBJECTS printer column to ClusterObjectSlice
(COSL), so `kubectl get cosl` shows how many objects each slice
contains at a glance.

Because COSL is immutable after creation, the count is set once by a
MutatingAdmissionPolicy (MAP) - no controller logic is needed.

## Design

### COSL Columns

```
NAME              OBJECTS   AGE
my-ext-abc-1-0    42        5m
```

New field on `ClusterObjectSlice`:
- `count int32` (required) - number of entries in the `objects` list.
  The MAP sets this automatically on CREATE, so callers may omit it.

### CEL Validation

A type-level XValidation rule on `ClusterObjectSlice` guarantees the
invariant:

```
+kubebuilder:validation:XValidation:rule="self.count == self.objects.size()",message="count must equal the number of objects"
```

The MAP sets `count` automatically on CREATE so callers don't have to,
but this rule ensures no request (including direct API writes that
bypass the MAP) can violate the invariant.

### MutatingAdmissionPolicy

A MAP sets `count` on CREATE. A matching MutatingAdmissionPolicyBinding
(MAPB) activates it. Both follow the same naming and pairing pattern as
the existing VAP/VAPB definitions.

**MAP** (`cosl-set-count`):
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicy
metadata:
  name: cosl-set-count
spec:
  reinvocationPolicy: IfNeeded
  matchConstraints:
    resourceRules:
      - apiGroups: ["orb.operatorframework.io"]
        apiVersions: ["v1alpha1"]
        resources: ["clusterobjectslices"]
        operations: ["CREATE"]
  mutations:
    - patchType: JSONPatch
      jsonPatch:
        expression: >-
          [JSONPatch{op: "add", path: "/count",
           value: object.objects.size()}]
```

**MAPB** (`cosl-set-count`):
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicyBinding
metadata:
  name: cosl-set-count
spec:
  policyName: cosl-set-count
```

Both are defined in `deploy/lib/api.libsonnet` alongside the existing
VAP/VAPB definitions and included in the `generate()` output array.
Since COSL is immutable, the policy only needs to run on CREATE.
