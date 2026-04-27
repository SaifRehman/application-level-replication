# 14 Troubleshooting recipes

Recovery paths for the things that did break during the build-out.

## A. xCluster replication: one direction stops flowing

**Symptom:** insert on cluster X, doesn't appear on cluster Y after
several seconds. Both replication groups still listed via `yb-admin
list_universe_replications`.

**Quick diagnosis:**

```sh
KUBECONFIG=/tmp/ocp-kube/cY.kubeconfig oc logs -n identity-db \
  yugabyte-yb-tserver-0 -c yb-tserver --tail=300 2>&1 \
  | grep -i "Could not find CDC stream"
```

If you see `Could not find CDC stream: stream_id "<hex>"`, the
inbound replication group on cluster Y is referencing a CDC stream
that was deleted on cluster X. Recreate the group on cluster Y:

```sh
# Example: c2 → c1 broken (group lives on C1)
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 \
  delete_universe_replication 12a62ea0-54cc-4b9a-9f7f-01ae0fb6487d

KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 \
  setup_universe_replication \
    12a62ea0-54cc-4b9a-9f7f-01ae0fb6487d \
    yb-masters-c2:7100 \
    000034cb000030008000000000004000
```

`setup_universe_replication` issues a fresh CDC stream on the source
and bootstraps the existing source rows over.

## B. Skupper Link drops to Pending

```sh
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc -n identity-db get link
# link-to-c1   Pending   ...   Not Operational
```

Most often this is an expired AccessGrant (default 1h). Recreate the
grant on C1, redeem on C2:

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc -n identity-db delete accessgrant c2-access
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc apply -k clusters/cluster-1/yb-universe
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc -n identity-db delete accesstoken link-to-c1
./scripts/skupper-link.sh
```

Wait for `link-to-c1` to report `Operational`.

## C. App pods CrashLoopBackOff after enabling ambient

**Symptom:** `kubelet ... Readiness probe failed: read: connection reset by peer`.

**Cause:** `PeerAuthentication` set to `STRICT` rejects kubelet's
plaintext probe at ztunnel.

**Fix (already in the manifests):**

- `platform/identity/mesh-base/peer-auth.yaml` uses
  `mtls.mode: PERMISSIVE` so kubelet probes pass through.
- `platform/identity/base/{backend,frontend}.yaml` add pod label
  `istio.io/dataplane-mode: none` to opt-out (workaround for OSSM 3.1.7
  ztunnel cert-impersonation gap).

If pods still crashloop, check:

```sh
oc -n identity-staging logs <pod-name> --previous --tail=20
```

A `database not ready, context canceled` line means the backend can't
reach the YSQL endpoint — verify the wrapper Service:

```sh
oc -n identity-db get svc yugabyte-ysql -o wide
oc -n identity-db get endpoints yugabyte-ysql
```

## D. Tekton EventListener pod CrashLoopBackOff

**Symptom:** `el-identity-listener-...` container fails liveness on
`/live`.

**Cause 1:** Missing `github-webhook-secret`. Create:

```sh
oc -n cicd create secret generic github-webhook-secret \
  --from-literal=token=<value>
```

**Cause 2:** Missing Triggers RBAC. The pipeline SA needs:

```sh
oc apply -f pipelines/rbac.yaml
```

…which includes the `tekton-triggers-eventlistener-roles` and
`tekton-triggers-eventlistener-clusterroles` bindings.

## E. ArgoCD reverts manual patches

**Symptom:** `oc patch istio default ...` reverts within seconds.

**Cause:** This sandbox ships an ArgoCD Application named
`istio-system` that owns the Istio CR with `selfHeal: true`.

**Fix:**

```sh
oc -n openshift-gitops patch application istio-system --type=merge \
  -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false,"prune":false}}}}'
```

Re-apply the patch and it'll stick. If you re-enable selfHeal later,
push the desired state into the Git source the Application watches.

## F. SonarQube Elasticsearch fails on path.data

**Symptom:** `java.nio.file.AccessDeniedException: /opt/sonarqube/data/es8`

**Cause:** PVC mounted with restrictive perms; ES (uid 1000) can't write.

**Fix (already in the manifest):** `platform/sonarqube/sonarqube.yaml`
has an `init-container` `chown-data` that runs as uid 0 and sets
ownership before the SonarQube container starts.

## G. Buildah refuses short image names

**Symptom:** `Error: creating build container: short-name resolution
enforced but cannot prompt without a TTY`.

**Cause:** Buildah requires fully-qualified image names by default.

**Fix (already in the task):** `pipelines/tasks/buildah-build.yaml`
writes a `~/.config/containers/registries.conf` with
`short-name-mode = "permissive"` and `unqualified-search-registries =
["docker.io", ...]` before running `buildah bud`.

## H. govulncheck fails with "Go too old"

**Symptom:** `golang.org/x/vuln@v1.3.0 requires go >= 1.25.0`.

**Fix (already in the task):** `pipelines/tasks/govulncheck.yaml`
uses `image: golang:1.25`.

## I. Tekton Go tasks: "permission denied" on cache

**Symptom:** `mkdir /.cache: permission denied`.

**Cause:** OpenShift restricted SCC runs containers as a random UID without write access to `/`.

**Fix (already in the task):** Set `HOME=/tekton/home` plus
`GOCACHE`, `GOMODCACHE`, `GOBIN` to subdirectories of `/tekton/home`.
Same pattern for `TRIVY_CACHE_DIR`.
