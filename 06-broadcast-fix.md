# 06 — The broadcast-address fix (the actual hard part)

## The trap

After step 05 the link was up and `yb-admin` could query across
clusters. I called `setup_universe_replication` in both directions and
`yb-admin` said it succeeded. CDC streams showed `state=ACTIVE`. But:

```sh
# inserted on C1:
INSERT 0 1

# wait 30s, query C2:
SELECT count(*) FROM digital_identity;
 count
-------
     0
```

No rows replicated. Master logs on C2 looped on:

```
xcluster_manager.cc:875 IsSetupUniverseReplicationDone: ...
```

## The cause

When a yb-tserver registers with its master, it advertises a
**broadcast address**. By default that address is the in-cluster pod
DNS:

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 list_all_tablet_servers
# RPC Host/Port:        yugabyte-yb-tserver-0.yugabyte-yb-tservers.identity-db.svc.cluster.local:9100
# Broadcast Host/Port:  yugabyte-yb-tserver-0.yugabyte-yb-tservers.identity-db.svc.cluster.local:9100
```

That hostname is **only** resolvable inside its own cluster. xCluster
data flow is tserver→tserver: C2's tservers ask C1's master "where is
the source tserver?", the master returns the in-cluster DNS, and the
connection times out because that DNS doesn't exist on C2.

## The fix in two parts

### Part A — give each cluster a Service named like its peer expects

Skupper Listeners create Services like `yb-tservers-c1` *on the remote
side*. We also need `yb-tservers-c1` to resolve **on its own cluster**
(C1) so that C1's own master can talk to its own tserver via the same
hostname it advertises.

These live in `clusters/cluster-{1,2}/yb-universe/self-services.yaml`
and are part of the `yb-universe` kustomization (so `oc apply -k`
brings them in along with the Skupper resources).

```yaml
# clusters/cluster-1/yb-universe/self-services.yaml
apiVersion: v1
kind: Service
metadata:
  name: yb-tservers-c1
  namespace: identity-db
spec:
  selector:
    app.kubernetes.io/name: yb-tserver
  ports: [{name: rpc, port: 9100, targetPort: 9100}]
---
apiVersion: v1
kind: Service
metadata:
  name: yb-masters-c1
  namespace: identity-db
spec:
  selector:
    app.kubernetes.io/name: yb-master
  ports: [{name: rpc, port: 7100, targetPort: 7100}]
```

Now `yb-tservers-c1` resolves on **both** clusters: locally (to the C1
tserver pod) on C1 and over Skupper (to C1) from C2.

### Part B — make tservers advertise that name

In `platform/yugabyte/values-cluster-{1,2}.yaml`:

```yaml
gflags:
  master:
    server_broadcast_addresses: "yb-masters-c1:7100"   # or yb-masters-c2 on C2
    use_private_ip: "never"
  tserver:
    server_broadcast_addresses: "yb-tservers-c1:9100"  # or yb-tservers-c2 on C2
    use_private_ip: "never"
```

Apply and rotate the StatefulSets:

```sh
for label in c1 c2; do
  K=/tmp/ocp-kube/$label.kubeconfig
  KUBECONFIG=$K helm upgrade yugabyte yugabytedb/yugabyte \
    -n identity-db --version 2025.1.4 \
    -f platform/yugabyte/values-$label.yaml
  KUBECONFIG=$K oc rollout restart statefulset/yugabyte-yb-master  -n identity-db
  KUBECONFIG=$K oc rollout restart statefulset/yugabyte-yb-tserver -n identity-db
done
```

Wait for `3/3 Running` on all four pods.

## Verify the broadcast is now correct

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 list_all_tablet_servers
# RPC Host/Port      yugabyte-yb-tserver-0.yugabyte-yb-tservers.identity-db.svc.cluster.local:9100
# Broadcast Host/Port  yb-tservers-c1:9100   <-- fixed
```

The RPC bind address is still the in-cluster DNS (clients on the same
cluster keep working), but the **broadcast** is now the cross-cluster
name, which Skupper routes correctly from the peer.
