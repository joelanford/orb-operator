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

local mapCOSLCount = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'MutatingAdmissionPolicy',
  metadata: { name: 'cosl-set-count' },
  spec: {
    matchConstraints: {
      resourceRules: [{
        apiGroups: ['orb.operatorframework.io'],
        apiVersions: ['v1alpha1'],
        resources: ['clusterobjectslices'],
        operations: ['CREATE'],
      }],
    },
    reinvocationPolicy: 'IfNeeded',
    mutations: [{
      patchType: 'JSONPatch',
      jsonPatch: {
        expression: '[JSONPatch{op: "add", path: "/count", value: object.objects.size()}]',
      },
    }],
  },
};

local mapCOSLCountBinding = {
  apiVersion: 'admissionregistration.k8s.io/v1',
  kind: 'MutatingAdmissionPolicyBinding',
  metadata: { name: 'cosl-set-count' },
  spec: {
    policyName: mapCOSLCount.metadata.name,
  },
};

local crds = [
  std.parseYaml(importstr '../crds/orb.operatorframework.io_clusterobjectdeployments.yaml')[0],
  std.parseYaml(importstr '../crds/orb.operatorframework.io_clusterobjectsets.yaml')[0],
  std.parseYaml(importstr '../crds/orb.operatorframework.io_clusterobjectslices.yaml')[0],
];

{
  generate()::
    [vapCOSName, vapCOSNameBinding, vapCOSOrphanFinalizer, vapCOSOrphanFinalizerBinding, vapCODNameLength, vapCODNameLengthBinding, mapCOSLCount, mapCOSLCountBinding] + crds,
}
