#!/usr/bin/env bash
set -euo pipefail

# Flushes coverage data from a running coverage-instrumented operator pod
# and copies it to a local directory.
#
# Usage: ./hack/collect-e2e-coverage.sh <kind-cluster> <namespace> <output-dir>

KIND_CLUSTER=${1:?usage: collect-e2e-coverage.sh <kind-cluster> <namespace> <output-dir>}
NAMESPACE=${2:?usage: collect-e2e-coverage.sh <kind-cluster> <namespace> <output-dir>}
OUTPUT_DIR=${3:?usage: collect-e2e-coverage.sh <kind-cluster> <namespace> <output-dir>}

POD_UID=$(kubectl -n "$NAMESPACE" get pod -l app=orb-operator -o jsonpath='{.items[0].metadata.uid}')
COV_DIR=/var/lib/kubelet/pods/$POD_UID/volumes/kubernetes.io~empty-dir/coverage

docker exec "$KIND_CLUSTER"-control-plane pkill -USR1 -f '/orb-operator$'

for i in $(seq 1 150); do
    if docker exec "$KIND_CLUSTER"-control-plane sh -c "ls $COV_DIR/covcounters.* >/dev/null 2>&1"; then
        break
    fi
    if [ "$i" -eq 150 ]; then
        echo "ERROR: timed out waiting for covcounters files — coverage flush failed" >&2
        exit 1
    fi
    sleep 0.1
done

rm -rf "$OUTPUT_DIR" && mkdir -p "$OUTPUT_DIR"
docker cp "$KIND_CLUSTER"-control-plane:"$COV_DIR"/. "$OUTPUT_DIR"/
go tool covdata textfmt -i="$OUTPUT_DIR" -o="$(dirname "$OUTPUT_DIR")/coverage.out"
