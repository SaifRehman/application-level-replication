# 12 GitOps (OpenShift GitOps / ArgoCD)

OpenShift GitOps was already installed on both clusters as part of the sandbox baseline. We only needed to add three Applications per cluster.

## What ArgoCD watches

**Cluster 1's ArgoCD** owns:

```
platform-c1            → repoURL: github.com/saifrehman/application-level-replication
                          path:    platform/namespaces
identity-staging-c1    → path:    clusters/cluster-1/staging/identity
identity-mesh-c1       → path:    clusters/cluster-1/staging/mesh
```

**Cluster 2's ArgoCD** owns the same three with `-c2` suffixes pointing at `clusters/cluster-2/...`. Both clusters watch the same repository on the same branch (`main`).

Apply on each cluster:

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc apply -f \
  clusters/cluster-1/_bootstrap/applications.yaml
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc apply -f \
  clusters/cluster-2/_bootstrap/applications.yaml
```

## Force a hard refresh after first creation

The default polling interval is 3 minutes. After applying the Applications, force them to refresh so they pick up the repo immediately:

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  label=$([ "$K" = "/tmp/ocp-kube/c1.kubeconfig" ] && echo c1 || echo c2)
  for app in platform-${label} identity-staging-${label} identity-mesh-${label}; do
    KUBECONFIG=$K oc -n openshift-gitops annotate application $app \
      argocd.argoproj.io/refresh=hard --overwrite
  done
done
```

Confirm `Synced + Healthy`:

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get application -n openshift-gitops
# identity-mesh-c1     Synced  Healthy
# identity-staging-c1  Synced  Healthy
# platform-c1          Synced  Healthy
```

## How a CI commit propagates

1. Tekton's `update-gitops` task on **C1** edits
   `clusters/cluster-1/staging/identity/kustomization.yaml` AND
   `clusters/cluster-2/staging/identity/kustomization.yaml`, bumping
   the image tag.
2. It commits and pushes to `origin/main` using the `github-push-token` Secret.
3. Within ~3 minutes (or instantly via webhook if you wire one), each cluster's ArgoCD diffs the desired state against live and
   triggers a `kubectl apply` to roll out the new image tag.
4. Both clusters end up with the new image without any direct
   communication between them.

## Gotcha: ArgoCD can fight your manual patches

The sandbox ships its own ArgoCD `Application` for `istio-system`. Anything we patched on the `Istio` CR via `oc patch` was reverted within seconds. We disabled self-heal on that Application:

```sh
oc -n openshift-gitops patch application istio-system --type=merge \
  -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false,"prune":false}}}}'
```

If you re-enable it, also push the desired Istio profile change into the Git source that `istio-system` watches.

## Optional — a webhook from GitHub to ArgoCD

Without a webhook, ArgoCD polls every 3 min. To make pushes propagate in seconds, point a second GitHub webhook (or a shared one) at the ArgoCD server route:

```sh
oc get route openshift-gitops-server -n openshift-gitops \
  -o jsonpath='{"https://"}{.spec.host}{"/api/webhook"}{"\n"}'
```

GitHub webhook fields:
- Payload URL: (above)
- Content type: `application/json`
- Secret: ArgoCD webhook secret (`spec.repo.webhookSecret` in
  `Application` or via the `argocd-secret` Secret).

For the demo we left polling on; the 3-minute lag is acceptable.
