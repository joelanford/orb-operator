# Troubleshooting

## Common issues

### COD stuck at "Unavailable"

Check which phase is blocking:

```bash
kubectl get cos <cos-name> -o jsonpath='{range .status.observedPhases[*]}{.name}{"\t"}{.status}{"\n"}{end}'
```

Look for phases with status `Reconciling` or `Invalid`, then check their incomplete objects:

```bash
kubectl get cos <cos-name> -o jsonpath='{range .status.observedPhases[?(@.status!="Available")]}{.name}:{"\n"}{range .incompleteObjects[*]}  {.kind}/{.name}: {.messages}{"\n"}{end}{end}'
```

### ProgressDeadlineExceeded

The rollout hasn't made progress within the configured `progressDeadlineMinutes`. Common causes:

- An assertion that can never pass (e.g., checking a condition that the object doesn't report)
- An object that fails to create (e.g., referencing a namespace that doesn't exist yet — check phase ordering)
- A Deployment that can't reach ready replicas (e.g., image pull errors, resource limits)

Check the COS status for specific error messages:

```bash
kubectl describe cos <cos-name>
```

### Collision errors

A phase object reports a collision when it tries to manage an object that already exists and the collision protection mode doesn't allow adoption.

**Resolution options:**

1. **Delete the existing object** and let the revision create it
2. **Change collision protection** to `IfNoController` or `None` on the specific object, phase, or revision
3. **Remove the object from your phases** if it should be managed by something else

### COS name rejected on creation

COS names must match `{group}-{revision}`. For example, a COS with `group: my-app` and `revision: 3` must be named `my-app-3`. This is enforced by a ValidatingAdmissionPolicy.

### COD name rejected on creation

COD names are limited to 52 characters so that derived COS names fit within the 63-character Kubernetes limit.

### Content hash mismatch

If a ClusterObjectSlice referenced by a COS is deleted and recreated with different content, the COS detects the hash mismatch and reports an error. This is a safety mechanism against content substitution.

**Resolution:** Create a new COS revision that references the new slice content.

### Objects not cleaned up after deletion

When deleting a COS, the operator runs a reverse-order teardown of managed objects via finalizers. If teardown is stuck:

1. Check for finalizers on the COS: `kubectl get cos <name> -o jsonpath='{.metadata.finalizers}'`
2. Check for objects that can't be deleted (e.g., namespaces with running pods, resources with their own finalizers)

### Orphan finalizer ordering error

The `orphan` finalizer cannot be removed while `orb.operatorframework.io/cos-finalizer` is still present. Remove the COS finalizer first, then the orphan finalizer.

## Inspecting state

### List all revisions for a group

```bash
kubectl get cos -l orb.operatorframework.io/template-hash
```

### Check operator logs

```bash
kubectl logs -n orb-operator-system deployment/orb-operator
```

### Watch all resources

```bash
kubectl get cod,cos,cosl -w
```

### Check phase progression in detail

```bash
kubectl get cos <name> -o json | jq '.status.observedPhases[] | {name, status, error, incomplete: [.incompleteObjects[]? | {kind, name, messages}]}'
```
