# Gated Rollout with CEL

This guide demonstrates how to use CEL expression assertions to create manual gates in your rollout. Objects are created but not considered "available" until an external condition is met — giving you fine-grained control over rollout progression.

## How it works

1. Objects are created in each phase
2. Each object has a CEL assertion that checks for an external signal (e.g., a ConfigMap field)
3. The phase doesn't complete until the signal is set
4. You control the pace by updating the signal objects

## Example: Gated ConfigMaps

Create a COD where each ConfigMap in each phase has a "gate" that must be opened:

```yaml
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: gated-rollout
spec:
  progressDeadlineMinutes: 10
  template:
    spec:
      phases:
        - name: phase-1
          objects:
            - object:
                apiVersion: v1
                kind: ConfigMap
                metadata:
                  name: gate-1a
                  namespace: default
              assertions:
                - celExpression:
                    expression: >-
                      has(self.data) && has(self.data.gate) && self.data.gate == 'open'
                    message: "gate is not open"
            - object:
                apiVersion: v1
                kind: ConfigMap
                metadata:
                  name: gate-1b
                  namespace: default
              assertions:
                - celExpression:
                    expression: >-
                      has(self.data) && has(self.data.gate) && self.data.gate == 'open'
                    message: "gate is not open"
        - name: phase-2
          objects:
            - object:
                apiVersion: v1
                kind: ConfigMap
                metadata:
                  name: gate-2a
                  namespace: default
              assertions:
                - celExpression:
                    expression: >-
                      has(self.data) && has(self.data.gate) && self.data.gate == 'open'
                    message: "gate is not open"
```

## Apply and observe

```bash
kubectl apply -f gated-rollout.yaml
```

The COD will show as `Unavailable` because the gates are closed:

```bash
kubectl get cod gated-rollout
```

```
NAME             AVAILABILITY   PROGRESSING                      AGE
gated-rollout    Unavailable    NewClusterObjectSetProgressing   5s
```

Check the incomplete objects to see which gates are waiting:

```bash
kubectl get cos gated-rollout-1 -o jsonpath='{range .status.observedPhases[*]}{.name}: {.status}{"\n"}{end}'
```

```
phase-1: Reconciling
phase-2: Unknown
```

## Open the gates

Open gates one at a time by patching the ConfigMaps:

```bash
kubectl patch configmap gate-1a -n default --type merge -p '{"data":{"gate":"open"}}'
kubectl patch configmap gate-1b -n default --type merge -p '{"data":{"gate":"open"}}'
```

Once all phase-1 gates are open, phase-2 begins:

```bash
kubectl patch configmap gate-2a -n default --type merge -p '{"data":{"gate":"open"}}'
```

## Use cases

- **Canary deployments** — Gate each phase on external health checks or metrics
- **Approval workflows** — Require human sign-off before proceeding to the next phase
- **Dependency readiness** — Wait for an external system to signal readiness
- **Staged migrations** — Roll out changes to one component at a time with manual verification between stages

## Automated gating script

The project includes a script that demonstrates automated gate opening. It creates a COD with 5 phases of 10 gated ConfigMaps each, then opens gates one per second:

```bash
./examples/gated-rollout.sh
```

This produces output like:

```
Building COD with 5 phases x 10 gated ConfigMaps...
Creating COD gated-rollout...
Opening gates one per second (50 total)...

[14:30:01] Opening gate: phase-1 / cm-p1-o1
[14:30:02] Opening gate: phase-1 / cm-p1-o2
...
[14:31:15] Opening gate: phase-5 / cm-p5-o10

All gates opened. Waiting for COD to become Available...
Done!
```
