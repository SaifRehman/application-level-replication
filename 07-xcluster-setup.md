# 07 xCluster bidirectional replication

With the cross-cluster network up (step 05) and broadcast addresses
fixed (step 06), `setup_universe_replication` actually completes
synchronously instead of looping forever.

## C1 -> C2 (run on C2)

`setup_universe_replication` runs on the **target** cluster. The
arguments are:
1. `<source_universe_uuid>` — the SOURCE universe's clusterUuid.
2. `<source_master_addresses>` — comma-separated source masters.
3. `<source_table_ids>` — table ids on the SOURCE to replicate.

```sh
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 \
  setup_universe_replication \
    3618389e-40de-4539-8894-c22e70c1fc7a \
    yb-masters-c1:7100 \
    000034cb000030008000000000004000
# Replication setup successfully
```

## C2 -> C1 (run on C1)

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 \
  setup_universe_replication \
    12a62ea0-54cc-4b9a-9f7f-01ae0fb6487d \
    yb-masters-c2:7100 \
    000034cb000030008000000000004000
# Replication setup successfully
```

## Verify both directions exist

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 list_universe_replications
# 1 Universe Replication Groups found:
# [12a62ea0-...]   <- inbound on C1, comes from C2

KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc exec -n identity-db yugabyte-yb-master-0 -c yb-master -- \
  /home/yugabyte/bin/yb-admin -master_addresses yugabyte-yb-masters:7100 list_universe_replications
# 1 Universe Replication Groups found:
# [3618389e-...]   <- inbound on C2, comes from C1
```

## Smoke test data flow at the SQL layer

```sh
KUBECONFIG=/tmp/ocp-kube/c1.kubeconfig oc exec -n identity-db yugabyte-yb-tserver-0 -c yb-tserver -- \
  /home/yugabyte/bin/ysqlsh -h yugabyte-yb-tservers -p 5433 -U yugabyte -d yugabyte -c \
  "INSERT INTO digital_identity (id, full_name, email)
     VALUES (gen_random_uuid(), 'Saif Rehman', 'srehman@redhat.com');"
# INSERT 0 1

# poll C2 (replication is async, sub-second on a healthy network)
KUBECONFIG=/tmp/ocp-kube/c2.kubeconfig oc exec -n identity-db yugabyte-yb-tserver-0 -c yb-tserver -- \
  /home/yugabyte/bin/ysqlsh -h yugabyte-yb-tservers -p 5433 -U yugabyte -d yugabyte -c \
  "SELECT full_name, email FROM digital_identity WHERE full_name='Saif Rehman'"
#   full_name   |       email
# --------------+-------------------
#  Saif rehman | srehman@redhat.com
```

Reverse direction works the same way (C2 insert -> visible on C1).

## Limitation

Bidirectional xCluster uses **last-writer-wins by hybrid logical timestamp** . If the same row gets written on both clusters at the same wall-clock instant, the loser is dropped silently. .

## Recovery: stale CDC stream

If you ever see this in a tserver log:

```
XClusterPoller GetChanges failure: Could not find CDC stream:
stream_id "<long-hex>"   (master error 3)
```

...the inbound replication group on **the cluster reading that log** is referencing a CDC stream that was deleted on the source. This happens after teardowns/restarts where one side cleaned up streams and the other still holds the old stream id.

Fix is to delete and re-establish the inbound group:

```sh
# Example: c2 → c1 broken
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
