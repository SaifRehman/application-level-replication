# 03 — Install YugabyteDB on both clusters

## Add the chart

```sh
helm repo add yugabytedb https://charts.yugabyte.com
helm repo update
helm search repo yugabytedb/yugabyte --versions | head -3
```

## Install on each cluster

The values files are scaled for **single-node** clusters: 1 master, 1 tserver,
RF=1. They live at `platform/yugabyte/values-cluster-{1,2}.yaml` and
include the broadcast addresses (`server_broadcast_addresses`) needed
for cross-cluster xCluster — see step 06 for why.

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig helm install yugabyte yugabytedb/yugabyte \
  --namespace identity-db \
  --version 2025.1.4 \
  -f platform/yugabyte/values-cluster-1.yaml

KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig helm install yugabyte yugabytedb/yugabyte \
  --namespace identity-db \
  --version 2025.1.4 \
  -f platform/yugabyte/values-cluster-2.yaml
```

## Verify pods Ready

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get pods -n identity-db
# yugabyte-yb-master-0   3/3 Running
# yugabyte-yb-tserver-0  3/3 Running
```

Same on cluster 2. First boot takes ~60s for tservers to bootstrap.

## Service names worth noting

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get svc -n identity-db
# yugabyte-yb-masters    7000/TCP, 7100/TCP, 15433/TCP   (headless)
# yugabyte-yb-tservers   9000/TCP, 5433/TCP, 9100/TCP    (headless)
```

Both are **headless** Services. That matters because:

