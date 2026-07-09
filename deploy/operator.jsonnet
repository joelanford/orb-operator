local image = std.extVar('image');
local namespace = if std.extVar('namespace') != '' then std.extVar('namespace') else 'orb-operator-system';

local crds = [
  std.parseYaml(importstr 'crds/orb.operatorframework.io_clusterobjectdeployments.yaml')[0],
  std.parseYaml(importstr 'crds/orb.operatorframework.io_clusterobjectsetrevisions.yaml')[0],
  std.parseYaml(importstr 'crds/orb.operatorframework.io_clusterobjectslices.yaml')[0],
];

local ns = {
  apiVersion: 'v1',
  kind: 'Namespace',
  metadata: { name: namespace },
};

local sa = {
  apiVersion: 'v1',
  kind: 'ServiceAccount',
  metadata: {
    name: 'orb-operator',
    namespace: namespace,
  },
};

local crb = {
  apiVersion: 'rbac.authorization.k8s.io/v1',
  kind: 'ClusterRoleBinding',
  metadata: { name: 'orb-operator' },
  roleRef: {
    apiGroup: 'rbac.authorization.k8s.io',
    kind: 'ClusterRole',
    name: 'cluster-admin',
  },
  subjects: [{
    kind: 'ServiceAccount',
    name: sa.metadata.name,
    namespace: namespace,
  }],
};

local deploy = {
  apiVersion: 'apps/v1',
  kind: 'Deployment',
  metadata: {
    name: 'orb-operator',
    namespace: namespace,
  },
  spec: {
    replicas: 1,
    selector: { matchLabels: { app: 'orb-operator' } },
    template: {
      metadata: { labels: { app: 'orb-operator' } },
      spec: {
        serviceAccountName: sa.metadata.name,
        securityContext: {
          runAsNonRoot: true,
          runAsUser: 65532,
          runAsGroup: 65532,
        },
        containers: [{
          name: 'operator',
          image: image,
          ports: [{
            containerPort: 8443,
            name: 'metrics',
            protocol: 'TCP',
          }],
          securityContext: {
            allowPrivilegeEscalation: false,
            capabilities: { drop: ['ALL'] },
          },
        }],
      },
    },
  },
};

local vapCOSRName = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicy',
  metadata: { name: 'cosr-name-must-match-group-revision' },
  spec: {
    matchConstraints: {
      resourceRules: [{
        apiGroups: ['orb.operatorframework.io'],
        apiVersions: ['v1alpha1'],
        resources: ['clusterobjectsetrevisions'],
        operations: ['CREATE'],
      }],
    },
    validations: [{
      expression: "object.metadata.name == object.spec.group + '-' + string(object.spec.revision)",
      message: 'name must be {group}-{revision}',
    }],
  },
};

local vapCOSRNameBinding = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicyBinding',
  metadata: { name: 'cosr-name-must-match-group-revision' },
  spec: {
    policyName: vapCOSRName.metadata.name,
    validationActions: ['Deny'],
  },
};

local vapCOSROrphanFinalizer = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicy',
  metadata: { name: 'cosr-orphan-finalizer-ordering' },
  spec: {
    failurePolicy: 'Fail',
    matchConstraints: {
      resourceRules: [{
        apiGroups: ['orb.operatorframework.io'],
        apiVersions: ['v1alpha1'],
        resources: ['clusterobjectsetrevisions'],
        operations: ['UPDATE'],
      }],
    },
    validations: [{
      expression: |||
        !(
          oldObject.metadata.?finalizers.orValue([]).exists(f, f == 'orphan') &&
          !object.metadata.?finalizers.orValue([]).exists(f, f == 'orphan') &&
          object.metadata.?finalizers.orValue([]).exists(f, f == 'orb.operatorframework.io/cosr-finalizer')
        )
      |||,
      message: 'cannot remove orphan finalizer while cosr-finalizer is still present',
    }],
  },
};

local vapCOSROrphanFinalizerBinding = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicyBinding',
  metadata: { name: 'cosr-orphan-finalizer-ordering' },
  spec: {
    policyName: vapCOSROrphanFinalizer.metadata.name,
    validationActions: ['Deny'],
  },
};

local vapCODNameLength = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicy',
  metadata: { name: 'cod-name-max-length' },
  spec: {
    matchConstraints: {
      resourceRules: [{
        apiGroups: ['orb.operatorframework.io'],
        apiVersions: ['v1alpha1'],
        resources: ['clusterobjectdeployments'],
        operations: ['CREATE'],
      }],
    },
    validations: [{
      expression: "size(object.metadata.name) <= 52",
      message: 'name must be at most 52 characters',
    }],
  },
};

local vapCODNameLengthBinding = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicyBinding',
  metadata: { name: 'cod-name-max-length' },
  spec: {
    policyName: vapCODNameLength.metadata.name,
    validationActions: ['Deny'],
  },
};

local svc = {
  apiVersion: 'v1',
  kind: 'Service',
  metadata: {
    name: 'orb-operator-metrics',
    namespace: namespace,
  },
  spec: {
    selector: deploy.spec.selector.matchLabels,
    ports: [{
      port: 8443,
      targetPort: 'metrics',
      name: 'metrics',
      protocol: 'TCP',
    }],
  },
};

{
  apiVersion: 'v1',
  kind: 'List',
  items: crds + [vapCOSRName, vapCOSRNameBinding, vapCOSROrphanFinalizer, vapCOSROrphanFinalizerBinding, vapCODNameLength, vapCODNameLengthBinding, ns, sa, crb, deploy, svc],
}
