# 08 Build images and deploy the identity app

For the demo we use OpenShift's `oc new-build --binary` + `oc start-build`, which runs the build in-cluster against the
Dockerfiles in the repo and pushes to each cluster's internal registry. The Tekton CI pipeline under `pipelines/` does the same thing automatically once a webhook is wired up — see step 11.

## Add a non-headless wrapper Service for YSQL

The Yugabyte chart ships a headless `yugabyte-yb-tservers` Service that maps to per-pod DNS. Headless Services play poorly with ambient mode and waypoint resolution. We add a regular `ClusterIP` wrapper for the YSQL endpoint so the backend's DATABASE_URL points at a stable address:

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc apply -f platform/yugabyte/ysql-service.yaml
done
# service/yugabyte-ysql created
```

The backend Secret in `platform/identity/base/backend.yaml` references
this wrapper:

```
DATABASE_URL: postgresql://yugabyte:yugabyte@yugabyte-ysql.identity-db.svc.cluster.local:5433/yugabyte?sslmode=disable
```

## Build the backend on each cluster

```sh
for label in c1 c2; do
  K=/tmp/ocp-kube/$label.kubeconfig
  KUBECONFIG=$K oc new-build --binary --strategy=docker \
    --name=identity-backend -n identity-staging
  (cd backend && KUBECONFIG=$K oc start-build identity-backend \
    --from-dir=. -n identity-staging --follow)
done
# Successfully pushed image-registry.openshift-image-registry.svc:5000/identity-staging/identity-backend@sha256:...
```

## Build the frontend on each cluster

```sh
for label in c1 c2; do
  K=/tmp/ocp-kube/$label.kubeconfig
  KUBECONFIG=$K oc new-build --binary --strategy=docker \
    --name=identity-frontend -n identity-staging
  (cd frontend && KUBECONFIG=$K oc start-build identity-frontend \
    --from-dir=. -n identity-staging --follow)
done
```

## Apply the Kustomize overlay

The overlay lives at `clusters/cluster-N/staging/identity/`. It pulls the base from `platform/identity/base/` and patches `config.js` (so the frontend banner shows the right cluster name).

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc apply -k clusters/cluster-1/staging/identity
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc apply -k clusters/cluster-2/staging/identity
# configmap, secret, service, deployment, route — created on both
```

Wait for the four Deployments to come up:

```sh
for label in c1 c2; do
  K=/tmp/ocp-kube/$label.kubeconfig
  KUBECONFIG=$K oc rollout status deploy/identity-backend  -n identity-staging --timeout=180s
  KUBECONFIG=$K oc rollout status deploy/identity-frontend -n identity-staging --timeout=180s
done
```

## Capture the public Routes

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get route identity -n identity-staging -o jsonpath='{.spec.host}'
# identity-identity-staging.apps.cluster-6b699.6b699.sandbox3565.opentlc.com
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc get route identity -n identity-staging -o jsonpath='{.spec.host}'
# identity-identity-staging.apps.cluster-4rnqf.4rnqf.sandbox3259.opentlc.com
```