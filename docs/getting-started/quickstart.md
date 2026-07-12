# Quick Start

This tutorial walks through deploying a simple application using orb-operator, then upgrading it to a new revision.

## Prerequisites

- orb-operator [installed](installation.md) on your cluster
- `kubectl` access with permissions to create cluster-scoped resources

## Step 1: Create a ClusterObjectDeployment

Create a file called `my-app.yaml`:

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: my-app
spec:
  template:
    spec:
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
        - name: rbac
          objects:
            - object:
                apiVersion: v1
                kind: ServiceAccount
                metadata:
                  name: my-app
                  namespace: my-app
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
                      serviceAccountName: my-app
                      containers:
                        - name: app
                          image: registry.k8s.io/pause:3.10
              assertions:
                - fieldsEqual:
                    fieldA: .status.replicas
                    fieldB: .status.readyReplicas
```

Apply it:

```bash
kubectl apply -f my-app.yaml
```

## Step 2: Watch the rollout

The operator creates objects phase by phase. Watch the progress:

```bash
kubectl get cod my-app -w
```

You'll see the `Availability` and `Progressing` columns update as each phase completes:

```
NAME     AVAILABILITY   PROGRESSING                      AGE
my-app   Unavailable    NewClusterObjectSetProgressing   0s
my-app   Available      NewClusterObjectSetProgressed    3s
```

## Step 3: Inspect the revision

The operator created a ClusterObjectSet (revision) automatically:

```bash
kubectl get cos
```

```
NAME       GROUP    REVISION   AVAILABLE   LIFECYCLE   AGE
my-app-1   my-app   1          True        Active      30s
```

Check the phase-level status:

```bash
kubectl get cos my-app-1 -o jsonpath='{range .status.observedPhases[*]}{.name}{"\t"}{.status}{"\n"}{end}'
```

```
namespace	Available
rbac	Available
workloads	Available
```

## Step 4: Upgrade to a new revision

Edit the deployment to change the replica count:

```bash
kubectl patch cod my-app --type merge -p '
  {"spec":{"template":{"spec":{"phases":[
    {"name":"namespace","objects":[{"object":{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"my-app"}},"assertions":[{"fieldValue":{"fieldPath":".status.phase","value":"Active"}}]}]},
    {"name":"rbac","objects":[{"object":{"apiVersion":"v1","kind":"ServiceAccount","metadata":{"name":"my-app","namespace":"my-app"}}}]},
    {"name":"workloads","objects":[{"object":{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"my-app","namespace":"my-app"},"spec":{"replicas":2,"selector":{"matchLabels":{"app":"my-app"}},"template":{"metadata":{"labels":{"app":"my-app"}},"spec":{"serviceAccountName":"my-app","containers":[{"name":"app","image":"registry.k8s.io/pause:3.10"}]}}}},"assertions":[{"fieldsEqual":{"fieldA":".status.replicas","fieldB":".status.readyReplicas"}}]}]}
  ]}}}}'
```

Or simply edit `my-app.yaml` to set `replicas: 2` and re-apply:

```bash
kubectl apply -f my-app.yaml
```

Watch the new revision roll out:

```bash
kubectl get cos -w
```

```
NAME       GROUP    REVISION   AVAILABLE   LIFECYCLE   AGE
my-app-1   my-app   1          True        Active      2m
my-app-2   my-app   2          False       Active      0s
my-app-2   my-app   2          True        Active      3s
my-app-1   my-app   1          True        Archived    3s
```

The old revision (`my-app-1`) is automatically archived once the new one becomes available. Objects shared between the two revisions (Namespace, ServiceAccount) were seamlessly transferred to the new revision without being recreated.

## Step 5: Clean up

```bash
kubectl delete cod my-app
```

This triggers a reverse-order teardown: the Deployment is removed first, then RBAC, then the Namespace.

## Next steps

- Learn how [phases and assertions](../concepts/phases-and-assertions.md) work
- Explore [collision protection](../concepts/collision-protection.md) modes
- Try a [gated rollout](../guides/gated-rollout.md) with CEL expressions
