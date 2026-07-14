---
status: pending
---
# COSL Object Count and Printer Column

## Summary

Add an `objectCount` field and OBJECTS printer column to
ClusterObjectSlice (COSL), so `kubectl get cosl` shows how many objects
each slice contains at a glance.

Because COSL is immutable after creation, the count is set once by a
MutatingAdmissionPolicy — no controller logic is needed.

## Design

### COSL Columns

```
NAME              OBJECTS   AGE
my-ext-abc-1-0    42        5m
```

New field on `ClusterObjectSlice`:
- `objectCount int32` — number of entries in the `objects` list.

### MutatingAdmissionPolicy

A MutatingAdmissionPolicy sets `objectCount` on create:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicy
metadata:
  name: cosl-object-count
spec:
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
          [JSONPatch{op: "add", path: "/objectCount",
           value: string(object.objects.size())}]
```

This is deployed alongside the operator manifests. Since COSL is
immutable, the policy only needs to run on CREATE.
