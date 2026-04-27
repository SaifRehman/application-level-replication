#!/usr/bin/env bash
set -euo pipefail

KUBECONFIG_GRANTOR="${KUBECONFIG_GRANTOR:-/tmp/ocp-kube/c1.kubeconfig}"
KUBECONFIG_REDEEMER="${KUBECONFIG_REDEEMER:-/tmp/ocp-kube/c2.kubeconfig}"
NAMESPACE="${NAMESPACE:-identity-db}"
GRANT_NAME="${GRANT_NAME:-c2-access}"
LINK_NAME="${LINK_NAME:-link-to-c1}"

echo "Reading AccessGrant '$GRANT_NAME' from grantor cluster…"
URL=$(KUBECONFIG="$KUBECONFIG_GRANTOR" oc -n "$NAMESPACE" get accessgrant "$GRANT_NAME" \
  -o jsonpath='{.status.url}')
CODE=$(KUBECONFIG="$KUBECONFIG_GRANTOR" oc -n "$NAMESPACE" get accessgrant "$GRANT_NAME" \
  -o jsonpath='{.status.code}')
CA=$(KUBECONFIG="$KUBECONFIG_GRANTOR" oc -n "$NAMESPACE" get accessgrant "$GRANT_NAME" \
  -o jsonpath='{.status.ca}' | awk '{print "    "$0}')

if [[ -z "$URL" || -z "$CODE" || -z "$CA" ]]; then
  echo "AccessGrant not Ready yet (url/code/ca empty). Wait and retry." >&2
  exit 1
fi

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

cat > "$TMP" <<YAMLEOF
apiVersion: skupper.io/v2alpha1
kind: AccessToken
metadata:
  name: $LINK_NAME
  namespace: $NAMESPACE
spec:
  url: $URL
  code: $CODE
  ca: |
$CA
YAMLEOF

echo "Applying AccessToken on redeemer cluster…"
KUBECONFIG="$KUBECONFIG_REDEEMER" oc apply -f "$TMP"

echo "Waiting for Link '$LINK_NAME' to become Operational…"
for i in {1..30}; do
  status=$(KUBECONFIG="$KUBECONFIG_REDEEMER" oc -n "$NAMESPACE" get link "$LINK_NAME" \
    -o jsonpath='{.status.conditions[?(@.type=="Operational")].status}' 2>/dev/null || true)
  if [[ "$status" == "True" ]]; then
    echo "Link Operational ✓"
    exit 0
  fi
  sleep 5
done

echo "Link did not become Operational within 150s." >&2
KUBECONFIG="$KUBECONFIG_REDEEMER" oc -n "$NAMESPACE" get link "$LINK_NAME" -o yaml | tail -25 >&2
exit 1
