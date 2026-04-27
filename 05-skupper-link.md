# 05 — Service Interconnect (Skupper) for cross-cluster Yugabyte

YugabyteDB xCluster replication needs L4 (binary gRPC) connectivity
between the master pods of both clusters on port 7100, and between the
tserver pods on port 9100. OpenShift `Route` only handles HTTP(S), so we
use **Skupper**, already pre-installed on this sandbox, to create a
virtual application network across the two clusters.

## What gets created

| Cluster | Resource (kind) | Name | Purpose |
|---|---|---|---|
| C1 | Site | `cluster-1` | Skupper endpoint in this namespace |
| C1 | AccessGrant | `c2-access` | One-time secret used by C2 to join the network |
| C1 | Connector | `yb-masters-c1` | "Local yb-master is reachable on the network as `yb-masters-c1`" |
| C1 | Connector | `yb-tservers-c1` | Same for tservers, port 9100 |
| C1 | Listener | `yb-masters-c2` | "Create local Service `yb-masters-c2` that proxies to the network" |
| C1 | Listener | `yb-tservers-c2` | Same for tservers |
| C1 | Service | `yb-masters-c1` | **NEW** — local self-pointing Service so the broadcast address resolves on C1 too |
| C1 | Service | `yb-tservers-c1` | Same for tservers |
| C2 | Site | `cluster-2` | |
| C2 | AccessToken | `link-to-c1` | Redeems the AccessGrant from C1 |
| C2 | Link | `link-to-c1` | The actual operational tunnel |
| C2 | Connector | `yb-masters-c2`, `yb-tservers-c2` | |
| C2 | Listener | `yb-masters-c1`, `yb-tservers-c1` | |
| C2 | Service | `yb-masters-c2`, `yb-tservers-c2` | self-pointing |

## Files in the repo

```
clusters/cluster-1/yb-universe/
  ├── skupper-site.yaml      # Site, AccessGrant, Connectors, Listeners
  ├── self-services.yaml     # local Services so broadcast addrs resolve here too
  ├── xcluster-config.yaml   # ConfigMap consumed by yb-admin setup_universe_replication
  └── kustomization.yaml
clusters/cluster-2/yb-universe/   # mirror, with -c2 names
scripts/skupper-link.sh           # AccessToken handshake (dynamic; can't be static YAML)
```

The AccessToken **cannot be a static manifest** because its `spec.url`,`spec.code`, and `spec.ca` are issued at runtime by C1's AccessGrant controller. The script reads those fields from the grantor and applies the AccessToken on the redeemer.

## Apply on each cluster (declarative parts)

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc apply -k clusters/cluster-1/yb-universe
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc apply -k clusters/cluster-2/yb-universe
```

Wait for both Sites to become Ready (~30s):

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc -n identity-db wait --for=condition=Ready site --all --timeout=120s
done
```

## Run the AccessToken handshake (the dynamic part)

```sh
./scripts/skupper-link.sh
# Reading AccessGrant 'c2-access' from grantor cluster…
# Applying AccessToken on redeemer cluster…
# Waiting for Link 'link-to-c1' to become Operational…
# Link Operational ✓
```

The script reads `accessgrant/c2-access.status.{url,code,ca}` on C1 and
applies the resulting `AccessToken` on C2. Within ~30s the `Link`
transitions to `Operational` and a single mTLS tunnel exists between
the two clusters' `skupper-router` pods.

## Verify connectors and listeners are matched

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc -n identity-db get listener
# yb-masters-c2    Ready  true   OK
# yb-tservers-c2   Ready  true   OK

KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc -n identity-db get connector
# yb-masters-c2    Ready  true   OK
# yb-tservers-c2   Ready  true   OK
```

If a Connector says `Status: Pending, Not Configured`, its `selector`
doesn't match any local pods. The community Yugabyte Helm chart labels
its pods `app.kubernetes.io/name=yb-master` (and `yb-tserver`) — make
sure the connector selector matches.

## Smoke test from a master pod

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yb-masters-c2:7100 list_all_tablet_servers
# Tablet Server UUID  RPC Host/Port  Heartbeat delay  Status
# c125c486...          ...:9100       0.79s            ALIVE
```

That's C2's tserver, reached from a pod on C1 via the Skupper-routed
name. The L4 path is live.

## Why the self-pointing Services are required

A Skupper Listener creates a local Service named after its routing key
**only on the side that has the Listener** — i.e. C1's Listener
`yb-masters-c2` creates `Service/yb-masters-c2` *on C1*, pointing into
the network.

But once we tell each cluster's Yugabyte to advertise itself as
`yb-tservers-cN:9100` (via `--server_broadcast_addresses` — see
step 06), the broadcast address has to **also resolve on the cluster
that owns it**. There's no Listener for `yb-tservers-c1` *on C1* (C1 is
the owner, not the consumer), so we manually create a Service with the
same name pointing at the local pods. That makes the broadcast address
resolvable from both sides:

- On C1: local Service routes to local C1 tserver.
- On C2: Skupper Listener routes to C1 over the network.

## Re-establishing a broken link

If `Link` ever drops out of `Operational` (most often after a complete
re-deploy of one side):

```sh
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc -n identity-db delete accesstoken link-to-c1
./scripts/skupper-link.sh
```

If the *AccessGrant* itself has expired (default `expirationWindow: 1h`
in our YAML), bump or delete it on C1 first:

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc -n identity-db delete accessgrant c2-access
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc apply -k clusters/cluster-1/yb-universe
```

...then re-run `scripts/skupper-link.sh`.

## Health check one-liner

```sh
for L in c1 c2; do
  echo "=== $L ==="
  KUBECONFIG=/tmp/ocp-kube/$L.kubeconfig oc -n identity-db get site,link,listener,connector \
    -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[-1].status 2>/dev/null
done
```
