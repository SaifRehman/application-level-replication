# 02 Platform namespaces + SCC grants

## Apply on both clusters

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc apply -f platform/namespaces/namespaces.yaml
done
```

Output (each cluster):

```
namespace/identity-staging created
namespace/identity-prod    created
namespace/identity-db      created
namespace/cicd             created
namespace/sonarqube        created
```

## SCC grants for Yugabyte

The community Yugabyte Helm chart starts containers as the `yugabyte`
user (uid 1000). OpenShift's restricted SCC blocks that, so we grant
`anyuid` and `privileged` SCCs to the `default` ServiceAccount in
`identity-db`:

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc adm policy add-scc-to-user anyuid     -z default -n identity-db
  KUBECONFIG=$K oc adm policy add-scc-to-user privileged -z default -n identity-db
done
```

## SCC grants for the app

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc adm policy add-scc-to-user anyuid -z default -n identity-staging
done
```

> must needed

## SCC grants for SonarQube

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc adm policy add-scc-to-user anyuid -z default -n sonarqube
done
```

SonarQube starts as uid 1000 and chowns its data dir; restricted SCC
prevents it.

