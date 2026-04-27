# 11 CI/CD: OpenShift Pipelines on C1, GitOps on both

The DevSecOps story is split: **C1 runs the CI**, **both clusters run
GitOps CD**. C2 has no Tekton at all — it just watches Git and pulls
manifests + images from C1's exposed internal registry.

## C1: Tekton stack

```sh
# operator
oc apply -f platform/operators/openshift-pipelines-subscription.yaml

# RBAC + tasks + pipeline
oc apply -f pipelines/rbac.yaml
oc apply -f pipelines/tasks/
oc apply -f pipelines/pipeline.yaml

# triggers
oc apply -f pipelines/triggers/
```

The pipeline has 13 tasks:
`clone -> (go-test || govulncheck || kube-linter) -> sonarqube -> (build-backend ‖ build-frontend) -> (trivy-backend ‖ trivy-frontend) -> (cosign-backend ‖ cosign-frontend) -> update-gitops -> zap-baseline`.

## Required secrets in `cicd` namespace

```sh
# webhook HMAC secret (must match GitHub)
oc -n cicd create secret generic github-webhook-secret \
  --from-literal=token=<your-strong-secret>

# SonarQube admin token (created via Sonar UI or API)
oc -n cicd create secret generic sonarqube-token \
  --from-literal=token=<sonar-token>

# GitHub PAT for the update-gitops task to push the bumped image tag
oc -n cicd create secret generic github-push-token \
  --from-literal=token=<your-github-PAT-with-repo-scope>
```

If `github-push-token` is absent, `update-gitops` commits locally and
exits 0 - useful for smoke tests; just ArgoCD won't see the change.

## SCCs the pipeline needs

```sh
oc adm policy add-scc-to-user privileged -z identity-pipeline -n cicd
oc adm policy add-scc-to-user anyuid -z default -n identity-staging
```

`privileged` is for `buildah` and `cosign` to mount the host-side overlay storage. `anyuid` on `identity-staging` is for the nginx-unprivileged frontend (uid 101).

In GitHub `Settings → Webhooks → Add webhook`:

| Field | Value |
|---|---|
| Payload URL | (route above) |
| Content type | `application/json` |
| Secret | (matches `github-webhook-secret/token`) |
| SSL | Disable verification (sandbox CA) |
| Events | Just the push event |



## End-to-end demo run (manual, no GitHub push)

```sh
oc create -f - <<'EOF'
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

