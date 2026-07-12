# Phased Rollout

This guide walks through a realistic phased rollout: deploying a Kubernetes operator with CRDs, RBAC, and a controller Deployment, each in their own phase with appropriate readiness assertions.

## The scenario

You're deploying an operator that manages `Widget` and `Gadget` custom resources. The deployment has natural dependencies:

1. **CRDs first** — The controller needs these to exist before it can watch them
2. **Namespace and RBAC** — The controller needs a namespace and permissions
3. **Controller Deployment** — Only after everything else is ready

## Define the ClusterObjectDeployment

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: sample-group
spec:
  template:
    spec:
      phases:
        - name: crds
          objects:
            - object:
                apiVersion: apiextensions.k8s.io/v1
                kind: CustomResourceDefinition
                metadata:
                  name: widgets.example.com
                spec:
                  group: example.com
                  names:
                    kind: Widget
                    listKind: WidgetList
                    plural: widgets
                    singular: widget
                  scope: Namespaced
                  versions:
                    - name: v1
                      served: true
                      storage: true
                      schema:
                        openAPIV3Schema:
                          type: object
                          properties:
                            spec:
                              type: object
                              properties:
                                size:
                                  type: string
              assertions:
                - conditionEqual:
                    type: Established
                    status: "True"
            - object:
                apiVersion: apiextensions.k8s.io/v1
                kind: CustomResourceDefinition
                metadata:
                  name: gadgets.example.com
                spec:
                  group: example.com
                  names:
                    kind: Gadget
                    listKind: GadgetList
                    plural: gadgets
                    singular: gadget
                  scope: Namespaced
                  versions:
                    - name: v1
                      served: true
                      storage: true
                      schema:
                        openAPIV3Schema:
                          type: object
                          properties:
                            spec:
                              type: object
                              properties:
                                color:
                                  type: string
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
                  name: sample-system
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
                  name: sample-controller
                  namespace: sample-system
            - object:
                apiVersion: rbac.authorization.k8s.io/v1
                kind: ClusterRole
                metadata:
                  name: sample-controller
                rules:
                  - apiGroups: ["example.com"]
                    resources: ["widgets", "gadgets"]
                    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
            - object:
                apiVersion: rbac.authorization.k8s.io/v1
                kind: ClusterRoleBinding
                metadata:
                  name: sample-controller
                roleRef:
                  apiGroup: rbac.authorization.k8s.io
                  kind: ClusterRole
                  name: sample-controller
                subjects:
                  - kind: ServiceAccount
                    name: sample-controller
                    namespace: sample-system

        - name: deployment
          objects:
            - object:
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                  name: sample-controller
                  namespace: sample-system
                spec:
                  replicas: 1
                  selector:
                    matchLabels:
                      app: sample-controller
                  template:
                    metadata:
                      labels:
                        app: sample-controller
                    spec:
                      serviceAccountName: sample-controller
                      containers:
                        - name: controller
                          image: registry.k8s.io/pause:3.10
                          ports:
                            - containerPort: 8080
              assertions:
                - fieldsEqual:
                    fieldA: .status.replicas
                    fieldB: .status.readyReplicas
```

## Apply and observe

```bash
kubectl apply -f sample-cod.yaml
kubectl get cod sample-group -w
```

Watch the phases complete in order:

```bash
kubectl get cos sample-group-1 -o jsonpath='{range .status.observedPhases[*]}{.name}{"\t"}{.status}{"\n"}{end}'
```

```
crds        Available
namespace   Available
rbac        Available
deployment  Available
```

## Upgrade

Update any field in the template (e.g., change the container image or add a new CRD) and re-apply. The operator creates revision 2, applies it phase by phase, transfers shared objects, and archives revision 1.

## Why phases matter

Without phases, all objects would be applied simultaneously. This can cause failures:

- A Deployment referencing a ServiceAccount in a namespace that doesn't exist yet
- A controller Pod starting before its CRDs are established
- Webhook configurations activating before the webhook server is ready

Phases enforce the dependency order and verify readiness at each step.
