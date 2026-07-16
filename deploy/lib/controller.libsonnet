{
  generate(image, namespace, profiles)::
    local hasProfile(p) = std.member(profiles, p);

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
                env+: [
                  { name: 'GOCOVERDIR', value: '/coverage' },
                  { name: 'ORB_DEADLINE_DURATION_UNIT_OVERRIDE', value: '1ms' },
                ],
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

    [ns, sa, crb, deploy, svc],
}
