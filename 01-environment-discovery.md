# 01 — Environment discovery

## Login to both clusters with isolated kubeconfigs

```sh
mkdir -p /tmp/ocp-kube
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc login \
  --token=sha256~<C1_TOKEN> \
  --server=https://api.cluster-6b699.6b699.sandbox3565.opentlc.com:6443 \
  --insecure-skip-tls-verify=true

KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc login \
  --token=sha256~<C2_TOKEN> \
  --server=https://api.cluster-4rnqf.4rnqf.sandbox3259.opentlc.com:6443 \
  --insecure-skip-tls-verify=true
```

## Capture facts that drive the rest of the design

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc version | grep Server
# Server Version: 4.17.52

KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get ingresses.config/cluster -o jsonpath='{.spec.domain}'
# apps.cluster-6b699.6b699.sandbox3565.opentlc.com

KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get nodes

KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc get storageclass
# gp3-csi (default), gp2-csi, this should change for your clusters


> cluster 2 should have identical steps
