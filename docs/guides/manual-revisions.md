# Manual Revisions

While ClusterObjectDeployments automate revision management, you can create and manage ClusterObjectSet resources directly for full control over the revision lifecycle.

## When to use manual revisions

- You have your own controller or pipeline that handles resolution and versioning
- You need custom archival logic that differs from the COD controller's behavior
- You want to test the COS layer independently

## Creating a revision chain

### Step 1: Create revision 1

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

Wait for it to become available:

```bash
kubectl wait --for=condition=Available cos/my-app-1 --timeout=120s
```

### Step 2: Create revision 2

Create a new COS with the same `group`, a higher `revision`, and updated content:

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectSet
metadata:
  name: my-app-2
spec:
  group: my-app
  revision: 2
  lifecycleState: Active
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
              replicas: 2    # Changed: scaled up
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
                      image: my-app:v2    # Changed: new image
          assertions:
            - fieldsEqual:
                fieldA: .status.replicas
                fieldB: .status.readyReplicas
```

The COS controller automatically:

1. Transfers ownership of shared objects (Namespace) from revision 1 to revision 2
2. Applies the updated Deployment spec
3. Reports per-phase availability

### Step 3: Archive the old revision

Once revision 2 is available, archive revision 1:

```bash
kubectl patch cos my-app-1 --type merge -p '{"spec":{"lifecycleState":"Archived"}}'
```

This triggers reverse-order teardown of objects still owned by revision 1. Since all objects in this example exist in both revisions, no objects are deleted — they've already been transferred to revision 2.

## Naming rules

COS names must follow the pattern `{group}-{revision}`. For example:

| Group | Revision | Required Name |
|-------|----------|---------------|
| `my-app` | `1` | `my-app-1` |
| `my-app` | `2` | `my-app-2` |
| `sample-group` | `42` | `sample-group-42` |

This naming constraint is enforced by a ValidatingAdmissionPolicy at creation time.

## Orphan deletion

If you want to delete a COS without deleting its managed objects, add the `orphan` finalizer before creating it:

```yaml
metadata:
  name: my-app-1
  finalizers:
    - orphan
```

When you delete the COS, the operator removes its owner references from managed objects but leaves the objects in place. This is useful for handing off management to another system.

!!! note
    The `orphan` finalizer must be removed **before** the `orb.operatorframework.io/cos-finalizer`. Attempting to remove them in the wrong order is rejected by a ValidatingAdmissionPolicy.
