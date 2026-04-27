# 04 Apply the schema on both clusters

The same DDL runs on each cluster. The Go backend's
`internal/identity/pgxstore.go` holds the canonical `Schema` constant
— we run it via `ysqlsh` on the tserver pod.

```sh
for K in /tmp/ocp-kube/c1.kubeconfig /tmp/ocp-kube/c2.kubeconfig; do
  KUBECONFIG=$K oc exec -n identity-db yugabyte-yb-tserver-0 -c yb-tserver -- \
    /home/yugabyte/bin/ysqlsh -h yugabyte-yb-tservers -p 5433 -U yugabyte -d yugabyte -c "
CREATE TABLE IF NOT EXISTS digital_identity (
  id          UUID PRIMARY KEY,
  full_name   TEXT NOT NULL,
  phone       TEXT NOT NULL DEFAULT '',
  address     TEXT NOT NULL DEFAULT '',
  email       TEXT NOT NULL DEFAULT '',
  passport_no TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);"
done
```



## Capture each cluster's table id and universe UUID

xCluster setup needs both. Both end up identical because the tables
were created in the same order on each cluster.

```sh
for label in C1 C2; do
  K=/tmp/ocp-kube/$([ $label = C1 ] && echo c1 || echo c2).kubeconfig
  KUBECONFIG=$K oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
    /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 \
    list_tables include_table_id | grep digital_identity
done
# C1: yugabyte.digital_identity ... 000034cb000030008000000000004000
# C2: yugabyte.digital_identity ... 000034cb000030008000000000004000

for label in C1 C2; do
  K=/tmp/ocp-kube/$([ $label = C1 ] && echo c1 || echo c2).kubeconfig
  KUBECONFIG=$K oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
    /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 \
    get_universe_config | python3 -c 'import sys,json; print(json.load(sys.stdin)["clusterUuid"])'
done
# C1: 3618389e-40de-4539-8894-c22e70c1fc7a
# C2: 12a62ea0-54cc-4b9a-9f7f-01ae0fb6487d
```

These two dbs UUIDs are the names we'll use for the xCluster
replication groups in step 07.
