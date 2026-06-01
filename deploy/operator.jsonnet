local image = std.extVar('image');
local namespace = if std.extVar('namespace') != '' then std.extVar('namespace') else 'orb-operator-system';

local crds = [
  std.parseYaml(importstr 'crds/orb.operatorframework.io_clusterobjectsets.yaml')[0],
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
  items: crds + [ns, sa, crb, deploy, svc],
}
