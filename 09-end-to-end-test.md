# 09 End-to-end test through the full app stack

This is the demo. POST to one cluster's frontend Route, GET from the
other cluster's frontend Route, observe the same row.

## Forward direction (C1 -> C2)

```sh
C1=identity-identity-staging.apps.cluster-6b699.6b699.sandbox3565.opentlc.com
C2=identity-identity-staging.apps.cluster-4rnqf.4rnqf.sandbox3259.opentlc.com

curl -sk -X POST "https://$C1/api/identities" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-change-me" \
  -d '{"full_name":"Mo Ajmal","email":"mo@example.com",
       "phone":"+33-1-23-45","passport_no":"P55555"}'
# {"id":"4ad7ea6b-557b-4ccc-88a2-dcb3c90c5b6c","full_name":"Moh Ajmal", ...}
```

Wait briefly (replication is async, typically sub-second), then read through C2's API:

```sh
curl -sk "https://$C2/api/identities" -H "X-API-Key: demo-key-change-me" \
  | python3 -m json.tool | grep -A1 Marie
# "full_name": "Mo Ajmal",
# "email":     "mo@example.com"
```

## Reverse direction (C2 -> C1)

```sh
NEW_ID=$(curl -sk -X POST "https://$C2/api/identities" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-change-me" \
  -d '{"full_name":"carlos galicia","email":"cg@gmail.com"}' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')

curl -sk "https://$C1/api/identities/$NEW_ID" -H "X-API-Key: demo-key-change-me"
# {"id":"ca4245d1-...","full_name":"Carlos Galicia","email":"cg@gmail.com", ...}
```

## What this proves

1. Each cluster has an independent YugabyteDB universe.
2. Each cluster's Go backend writes to its own local YSQL endpoint.
3. The application code has **no awareness** of replication — it just
   talks SQL to `yugabyte-ysql:5433`.
4. Replication happens entirely at the database layer (xCluster streams the WAL between the two universes via Skupper).
5. The application converges on both sides in well under a second on a healthy network.

## Open the UI

Each cluster has a UI at:

> you may find url in routes

- C1: https://identity-identity-staging.apps.cluster-6b699.6b699.sandbox3565.opentlc.com
- C2: https://identity-identity-staging.apps.cluster-4rnqf.4rnqf.sandbox3259.opentlc.com

The header badge shows `cluster-1` or `cluster-2` so it's obvious which side you're on. Create or edit an identity on one tab, hit "Refresh" on the other tab — the new row appears.

## Cleanup (when done)

```sh
for label in c1 c2; do
  K=/tmp/ocp-kube/$label.kubeconfig
  i=$([ "$label" = c1 ] && echo 1 || echo 2)
  KUBECONFIG=$K oc delete -k clusters/cluster-$i/staging/identity
  KUBECONFIG=$K helm uninstall yugabyte -n identity-db
  KUBECONFIG=$K oc delete pvc -n identity-db --all
  KUBECONFIG=$K oc delete -k clusters/cluster-$i/yb-universe
done
```
