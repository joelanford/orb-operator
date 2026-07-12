# Installation

## Prerequisites

- Kubernetes 1.30+ (required for ValidatingAdmissionPolicy support)
- `kubectl` configured to access your cluster
- Cluster-admin privileges

## Install from release manifest

Apply the latest release manifest directly:

```bash
kubectl apply -f https://github.com/joelanford/orb-operator/releases/latest/download/operator.yaml
```

This installs:

- Three CRDs (ClusterObjectDeployment, ClusterObjectSet, ClusterObjectSlice)
- ValidatingAdmissionPolicies for API integrity
- The `orb-operator-system` namespace
- The operator Deployment, ServiceAccount, and ClusterRoleBinding
- A metrics Service on port 8443

## Verify the installation

Check that the operator is running:

```bash
kubectl -n orb-operator-system rollout status deployment/orb-operator
```

Verify the CRDs are installed:

```bash
kubectl get crd clusterobjectdeployments.orb.operatorframework.io \
              clusterobjectsets.orb.operatorframework.io \
              clusterobjectslices.orb.operatorframework.io
```

## Uninstall

Remove the operator and all its resources:

```bash
kubectl delete -f https://github.com/joelanford/orb-operator/releases/latest/download/operator.yaml
```

!!! warning
    Deleting the CRDs will delete all ClusterObjectDeployment, ClusterObjectSet, and ClusterObjectSlice resources. Objects managed by those resources will be cleaned up by the operator's finalizers before the CRDs are removed.
