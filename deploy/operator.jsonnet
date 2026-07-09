local image = std.extVar('image');
local namespace = if std.extVar('namespace') != '' then std.extVar('namespace') else 'orb-operator-system';
local profiles = std.extVar('profiles');
local hasProfile(p) = std.member(profiles, p);

local crds = [
  std.parseYaml(importstr 'crds/orb.operatorframework.io_clusterobjectdeployments.yaml')[0],
  std.parseYaml(importstr 'crds/orb.operatorframework.io_clusterobjectsets.yaml')[0],
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

local applyE2eProfile(deploy) = deploy {
  spec+: {
    template+: {
      spec+: {
        terminationGracePeriodSeconds: 120,
        volumes+: [{
          name: 'coverage',
          emptyDir: {},
        }],
        containers: [
          c {
            imagePullPolicy: 'Never',
            env+: [{ name: 'GOCOVERDIR', value: '/coverage' }],
            volumeMounts+: [{ name: 'coverage', mountPath: '/coverage' }],
          }
          for c in deploy.spec.template.spec.containers
        ],
      },
    },
  },
};

local baseDeploy = {
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

local deploy = if hasProfile('e2e') then applyE2eProfile(baseDeploy) else baseDeploy;

local vapCOSName = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicy',
  metadata: { name: 'cos-name-must-match-group-revision' },
  spec: {
    matchConstraints: {
      resourceRules: [{
        apiGroups: ['orb.operatorframework.io'],
        apiVersions: ['v1alpha1'],
        resources: ['clusterobjectsets'],
        operations: ['CREATE'],
      }],
    },
    validations: [{
      expression: "object.metadata.name == object.spec.group + '-' + string(object.spec.revision)",
      message: 'name must be {group}-{revision}',
    }],
  },
};

local vapCOSNameBinding = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicyBinding',
  metadata: { name: 'cos-name-must-match-group-revision' },
  spec: {
    policyName: vapCOSName.metadata.name,
    validationActions: ['Deny'],
  },
};

local vapCOSOrphanFinalizer = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicy',
  metadata: { name: 'cos-orphan-finalizer-ordering' },
  spec: {
    failurePolicy: 'Fail',
    matchConstraints: {
      resourceRules: [{
        apiGroups: ['orb.operatorframework.io'],
        apiVersions: ['v1alpha1'],
        resources: ['clusterobjectsets'],
        operations: ['UPDATE'],
      }],
    },
    validations: [{
      expression: |||
        !(
          oldObject.metadata.?finalizers.orValue([]).exists(f, f == 'orphan') &&
          !object.metadata.?finalizers.orValue([]).exists(f, f == 'orphan') &&
          object.metadata.?finalizers.orValue([]).exists(f, f == 'orb.operatorframework.io/cos-finalizer')
        )
      |||,
      message: 'cannot remove orphan finalizer while cos-finalizer is still present',
    }],
  },
};

local vapCOSOrphanFinalizerBinding = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'ValidatingAdmissionPolicyBinding',
  metadata: { name: 'cos-orphan-finalizer-ordering' },
  spec: {
    policyName: vapCOSOrphanFinalizer.metadata.name,
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
  items: crds + [vapCOSName, vapCOSNameBinding, vapCOSOrphanFinalizer, vapCOSOrphanFinalizerBinding, vapCODNameLength, vapCODNameLengthBinding, ns, sa, crb, deploy, svc],
}
