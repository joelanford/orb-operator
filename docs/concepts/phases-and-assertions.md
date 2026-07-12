# Phases & Assertions

Phases and assertions are the core mechanisms for safe, ordered rollout of Kubernetes objects.

## Phases

A **phase** is an ordered group of objects that are applied together. Phases within a revision are applied sequentially: the controller does not advance to the next phase until all objects in the current phase satisfy their assertions.

This ordering guarantees that dependencies are ready before dependents are created. For example, a CRD must be `Established` before instances of that CRD can be created.

### Phase structure

```yaml
phases:
  - name: crds          # Phase 1: Install CRDs first
    objects:
      - object: { ... }
        assertions: [ ... ]
  - name: namespace     # Phase 2: Then the namespace
    objects:
      - object: { ... }
        assertions: [ ... ]
  - name: workloads     # Phase 3: Finally the workloads
    objects:
      - object: { ... }
        assertions: [ ... ]
```

### Limits

- A revision may contain **1–20 phases**
- Each phase may contain **1–50 objects**
- Phase names must be valid DNS-1035 labels (lowercase alphanumeric or `-`, starting with a letter)

### Phase status

Each phase reports one of these statuses:

| Status | Meaning |
|--------|---------|
| `Reconciling` | The controller is actively evaluating this phase |
| `Available` | All objects in this phase pass their assertions |
| `Invalid` | The phase failed preflight validation |
| `Unknown` | The phase was not evaluated during the most recent reconcile |
| `Superseded` | All objects have been adopted by a newer revision |
| `TearingDown` | The controller is deleting objects in this phase |
| `TeardownComplete` | All objects in this phase have been deleted |

## Assertions

An **assertion** defines what "ready" means for a specific object. When an object has assertions, it is not considered available until all of them pass. Each object can have up to 16 assertions.

### Assertion types

#### conditionEqual

Checks that a status condition with the given type has the expected status value.

```yaml
assertions:
  - conditionEqual:
      type: Established    # The condition type
      status: "True"       # Expected status: "True", "False", or "Unknown"
```

Best for: CRDs (`Established`), resources with standard conditions (`Ready`, `Available`).

#### fieldsEqual

Checks that two fields on the object have equal values.

```yaml
assertions:
  - fieldsEqual:
      fieldA: .status.replicas         # JSON path to first field
      fieldB: .status.readyReplicas    # JSON path to second field
```

Best for: Deployments (all replicas ready), StatefulSets, or any resource where readiness is expressed as a field comparison.

#### fieldValue

Checks that a single field matches an expected value.

```yaml
assertions:
  - fieldValue:
      fieldPath: .status.phase    # JSON path to the field
      value: Active               # Expected value
```

Best for: Namespaces (`phase: Active`), Pods (`phase: Running`), or any resource with a known ready-state value.

#### celExpression

Evaluates a CEL expression against the object. The managed object is available as `self` in the expression scope. The expression must evaluate to `true` for the assertion to pass.

```yaml
assertions:
  - celExpression:
      expression: "has(self.data) && has(self.data.gate) && self.data.gate == 'open'"
      message: "gate is not open"    # Optional: shown when assertion fails
```

Best for: Custom readiness logic, gating mechanisms, complex conditions that other assertion types can't express.

!!! tip
    CEL expressions have access to the full object. You can check any field, compare multiple values, or use CEL's built-in functions (string operations, list operations, etc.).

### Built-in assertions

When **no explicit assertions** are specified for an object, the controller applies built-in assertions for well-known types:

| Kind | Built-in Assertion |
|------|--------------------|
| CustomResourceDefinition | `conditionEqual: type=Established, status=True` |
| Deployment | `fieldsEqual: .status.replicas == .status.readyReplicas` |

For all other types with no explicit assertions, the object is considered available immediately after successful apply.

## Example: Multi-phase with mixed assertions

```yaml
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
  - name: namespace
    objects:
      - object:
          apiVersion: v1
          kind: Namespace
          metadata:
            name: widget-system
        assertions:
          - fieldValue:
              fieldPath: .status.phase
              value: Active
  - name: controller
    objects:
      - object:
          apiVersion: apps/v1
          kind: Deployment
          metadata:
            name: widget-controller
            namespace: widget-system
          # ...
        assertions:
          - fieldsEqual:
              fieldA: .status.replicas
              fieldB: .status.readyReplicas
```

In this example:

1. The CRD is created first and must be `Established` before proceeding
2. The Namespace is created and must be `Active` before proceeding
3. The Deployment is created and must have all replicas ready
