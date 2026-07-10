#!/usr/bin/env bash
set -euo pipefail

COD_NAME="gated-rollout"
PHASES=5
OBJECTS_PER_PHASE=10

gate_assertion='has(self.data) && has(self.data.gate) && self.data.gate == '"'"'open'"'"''

build_cod() {
  local cod
  cod=$(cat <<EOF
apiVersion: orb.operatorframework.io/v1alpha1
kind: ClusterObjectDeployment
metadata:
  name: ${COD_NAME}
spec:
  progressDeadlineMinutes: 1
  template:
    spec:
      phases: []
EOF
)

  for p in $(seq 1 "$PHASES"); do
    local phase_name="phase-${p}"
    cod=$(echo "$cod" | yq e ".spec.template.spec.phases += [{\"name\": \"${phase_name}\", \"objects\": []}]" -)

    for o in $(seq 1 "$OBJECTS_PER_PHASE"); do
      local cm_name="cm-p${p}-o${o}"
      local obj
      obj=$(cat <<OBJEOF
object:
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: ${cm_name}
    namespace: default
assertions:
  - celExpression:
      expression: "${gate_assertion}"
      message: "gate is not open"
OBJEOF
)
      cod=$(echo "$cod" | yq e ".spec.template.spec.phases[$((p-1))].objects += [$(echo "$obj" | yq e -o=json -I=0 '.')]" -)
    done
  done

  echo "$cod"
}

echo "Building COD with ${PHASES} phases x ${OBJECTS_PER_PHASE} gated ConfigMaps..."
cod_yaml=$(build_cod)

echo "Creating COD ${COD_NAME}..."
echo "$cod_yaml" | kubectl apply -f -

echo ""
echo "Opening gates one per second (${PHASES} phases x ${OBJECTS_PER_PHASE} objects = $((PHASES * OBJECTS_PER_PHASE)) total)..."
echo ""

for p in $(seq 1 "$PHASES"); do
  for o in $(seq 1 "$OBJECTS_PER_PHASE"); do
    cm_name="cm-p${p}-o${o}"
    echo -n "[$(date +%H:%M:%S)] Opening gate: phase-${p} / ${cm_name}"
    until kubectl patch configmap "$cm_name" -n default --type merge -p '{"data":{"gate":"open"}}' 2>/dev/null; do
      echo -n "."
      sleep 0.2
    done
    sleep 1.5
  done
done

echo ""
echo "All gates opened. Waiting for COD to become Available..."
kubectl wait --for=condition=Available "clusterobjectdeployment/${COD_NAME}" --timeout=120s
echo "Done!"
