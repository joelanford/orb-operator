# Collision Protection

Collision protection controls what happens when a revision tries to manage an object that already exists on the cluster. This setting determines whether the operator adopts existing objects or reports a collision error.

## Modes

### Prevent (default)

Only manages objects that the revision itself created. If an object already exists (created by another controller, a previous manual apply, or any other source), the revision reports a collision and does not modify it.

```yaml
collisionProtection: Prevent
```

Use when: You want strict ownership — each revision creates its own objects and never interferes with objects from other sources.

### IfNoController

Adopts and updates objects that exist but have no controller owner reference. Objects that are already owned by another controller cause a collision.

```yaml
collisionProtection: IfNoController
```

Use when: You want to adopt unmanaged resources (e.g., objects created manually or by a script) but don't want to take ownership away from another controller.

### None

Adopts and updates objects unconditionally, even if they are already owned by another controller.

```yaml
collisionProtection: None
```

!!! warning
    Use with caution. Multiple controllers managing the same object may cause unnecessary API server and etcd load as they compete to reconcile it.

Use when: You need to take over management of objects currently owned by another controller (e.g., during a migration).

## Override hierarchy

Collision protection can be set at three levels. More specific settings override less specific ones:

```
Object-level  >  Phase-level  >  Spec-level (default)
```

### Spec-level default

Set in the COD template or COS spec. Applies to all phases and objects unless overridden.

```yaml
spec:
  template:
    spec:
      collisionProtection: Prevent
      phases: [...]
```

### Phase-level override

Overrides the spec-level default for all objects within a specific phase.

```yaml
phases:
  - name: adopt-existing
    collisionProtection: IfNoController
    objects: [...]
```

### Object-level override

Overrides both spec-level and phase-level settings for a single object.

```yaml
phases:
  - name: mixed
    objects:
      - object: { ... }
        collisionProtection: None    # Only this object uses None
      - object: { ... }
        # Inherits phase-level or spec-level setting
```

## Example: Migrating ownership

When migrating management of objects from one system to another, you can use `IfNoController` or `None` to adopt existing objects:

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: adopted-app
spec:
  template:
    spec:
      collisionProtection: IfNoController
      phases:
        - name: existing-resources
          objects:
            - object:
                apiVersion: v1
                kind: Namespace
                metadata:
                  name: legacy-app
              assertions:
                - fieldValue:
                    fieldPath: .status.phase
                    value: Active
```

After the revision becomes available, the operator manages these objects — any external modifications will be reverted by drift detection.
