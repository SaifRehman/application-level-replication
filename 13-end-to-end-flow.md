# 13 — Full DevSecOps + GitOps flow demonstration

This is the closing demo: one `git push` to `main` produces a green PipelineRun on cluster 1, which commits an image-tag bump back to the repo, which both clusters' ArgoCD pull and apply, which rolls new pods on both clusters — all while YugabyteDB xCluster keeps replicating identity rows in both directions.

## Pipeline shape

The full `identity-ci` pipeline has 11 tasks running in this DAG:

```
clone
   ├─ go-test ──┐
   ├─ govulncheck ┐
   └─ kube-linter ┴─→ sonarqube ─┐
                                 ├─→ build-backend  ─→ trivy-backend ─┐
                                 └─→ build-frontend ─→ trivy-frontend ┴→ update-gitops ─→ zap-baseline
```

After the second successful run (`identity-ci-pr9jn`):

```
identity-ci-pr9jn-build-backend     True Succeeded
identity-ci-pr9jn-build-frontend    True Succeeded
identity-ci-pr9jn-clone             True Succeeded
identity-ci-pr9jn-go-test           True Succeeded
identity-ci-pr9jn-govulncheck       True Succeeded
identity-ci-pr9jn-kube-linter       True Succeeded
identity-ci-pr9jn-sonarqube         True Succeeded
identity-ci-pr9jn-trivy-backend     True Succeeded
identity-ci-pr9jn-trivy-frontend    True Succeeded
identity-ci-pr9jn-update-gitops     True Succeeded
identity-ci-pr9jn-zap-baseline      True Succeeded
```

Total runtime: ~6 minutes.

## How the loop closes

1. Tekton's `update-gitops` task pushes a commit to `main`:
   ```
   [main 6b0a8eb] ci: bump images to latest/latest
    2 files changed, 8 insertions(+), 8 deletions(-)
   ```
2. ArgoCD on **both** clusters polls within ~3 minutes (sooner with a
   webhook) and reconciles to the new commit:
   ```
   identity-mesh-c1     Synced  Healthy  6b0a8eb
   identity-staging-c1  Synced  Healthy  6b0a8eb
   identity-mesh-c2     Synced  Healthy  6b0a8eb
   identity-staging-c2  Synced  Healthy  6b0a8eb
   ```
3. Both clusters roll out the new image tag.
4. YugabyteDB xCluster keeps replicating identity rows during the
   rollout — the data layer is invisible to the CI loop.

## Manual trigger (no GitHub webhook)

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc create -f - <<'EOF'
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  generateName: identity-ci-
  namespace: cicd
spec:
  pipelineRef: { name: identity-ci }
  taskRunTemplate: { serviceAccountName: identity-pipeline }
  params:
    - { name: GIT_URL,      value: https://github.com/saifrehman/application-level-replication }
    - { name: GIT_REVISION, value: main }
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: [ReadWriteOnce]
          resources: { requests: { storage: 2Gi } }
          storageClassName: gp3-csi
EOF
```

## Watch the pipeline

```sh
oc -n cicd get pipelinerun --watch
oc -n cicd get taskrun -l tekton.dev/pipelineRun=<pr-name>
oc -n cicd logs -l tekton.dev/taskRun=<tr-name> --tail=50
```

## Required Secrets in the `cicd` namespace

```sh
# Webhook HMAC secret (must match GitHub)
oc -n cicd create secret generic github-webhook-secret \
  --from-literal=token=<your-strong-secret>

# SonarQube admin token (created via Sonar UI or API)
oc -n cicd create secret generic sonarqube-token \
  --from-literal=token=<sonar-token>

# GitHub PAT for update-gitops to push the bumped image tag
oc -n cicd create secret generic github-push-token \
  --from-literal=token=<your-github-PAT>
```

If `github-push-token` is absent, `update-gitops` commits locally and
exits 0 — useful for smoke tests; just ArgoCD won't see the change.

## Demo script (live)

1. **Open four tabs:** C1 UI, C2 UI, C1 ArgoCD, C2 ArgoCD.
2. **Add an identity on C1 UI** — refresh C2 UI, see it appear (proves xCluster replication, no CI involved yet).
3. **Edit `backend/internal/identity/model.go`**, add a comment-free change (e.g. tighten validation), `git commit`, `git push`.
4. **Watch C1 ArgoCD console**: an `identity-ci-…` PipelineRun appears in the `cicd` namespace, churning through 11 tasks.
5. **After the pipeline goes green**, both ArgoCD apps tick to the new commit SHA and roll new backend pods.
6. **Add another identity on C1 UI** — proves the freshly-deployed pod still works and DB replication still flows.
