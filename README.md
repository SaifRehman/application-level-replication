<div align="center">

<img src="https://raw.githubusercontent.com/cncf/artwork/main/projects/kubernetes/icon/color/kubernetes-icon-color.png" height="80" alt="Kubernetes" />
&nbsp;&nbsp;&nbsp;
<img src="https://cdn.simpleicons.org/redhatopenshift/EE0000" height="80" alt="Red Hat OpenShift" />
&nbsp;&nbsp;&nbsp;
<img src="https://avatars.githubusercontent.com/u/17074854?s=200&v=4" height="80" alt="YugabyteDB" />
&nbsp;&nbsp;&nbsp;
<img src="https://avatars.githubusercontent.com/u/68505716?s=200&v=4" height="80" alt="Skupper" />
&nbsp;&nbsp;&nbsp;
<img src="https://raw.githubusercontent.com/cncf/artwork/main/projects/argo/icon/color/argo-icon-color.png" height="80" alt="ArgoCD" />
&nbsp;&nbsp;&nbsp;
<img src="https://cdn.simpleicons.org/tekton/FD495C" height="80" alt="Tekton" />

<h1>application-level-replication</h1>

<p><b>A two-cluster OpenShift demo of database-level cross-cluster replication,<br/>
wrapped in a full DevSecOps + GitOps loop around a Go &amp; HTML digital-identity app.</b></p>

<p>
  <img src="https://img.shields.io/badge/OpenShift-4.17-EE0000?logo=redhatopenshift&logoColor=white" alt="OpenShift 4.17" />
  <img src="https://img.shields.io/badge/YugabyteDB-2025.1.4-FF6900?logo=yugabytedb&logoColor=white" alt="YugabyteDB" />
  <img src="https://img.shields.io/badge/Skupper-v2-2BB673?logoColor=white" alt="Skupper v2" />
  <img src="https://img.shields.io/badge/Tekton-Pipelines-FD495C?logo=tekton&logoColor=white" alt="Tekton" />
  <img src="https://img.shields.io/badge/ArgoCD-GitOps-EF7B4D?logo=argo&logoColor=white" alt="ArgoCD" />
  <img src="https://img.shields.io/badge/Istio-Ambient-466BB0?logo=istio&logoColor=white" alt="Istio Ambient" />
  <br/>
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/SonarQube-SAST-4E9BCD?logo=sonarqube&logoColor=white" alt="SonarQube" />
  <img src="https://img.shields.io/badge/Trivy-CVE%20scan-1904DA?logo=aqua&logoColor=white" alt="Trivy" />
  <img src="https://img.shields.io/badge/OWASP%20ZAP-DAST-000000?logo=owasp&logoColor=white" alt="OWASP ZAP" />
  <img src="https://img.shields.io/badge/govulncheck-SCA-00ADD8?logo=go&logoColor=white" alt="govulncheck" />
  <img src="https://img.shields.io/badge/kube--linter-policy-326CE5?logo=kubernetes&logoColor=white" alt="kube-linter" />
</p>


</div>

---

## Overview

Two independent OpenShift clusters each run their own instance of **YugabyteDB**. The two universes are joined by **xCluster bidirectional replication**, transported over a **Skupper service-interconnect tunnel** so no database port is exposed publicly. A simple Go backend writes toc its **local** Yugabyte; the data appears in the peer cluster within
about a second with no application involvement.

A **Tekton pipeline on cluster 1** (build, test, scan, sign, scan-image, scan-app) bumps the image tag in this repo. **OpenShift GitOps on both clusters** pulls the new tag and rolls out the change. **Cluster 2 has no CI** — it's pure CD, watching the same repo on a different path.


---

## What this demo shows

| Capability | Where it lives |
|---|---|
| **DB-level bidirectional replication** without exposing the DB | YugabyteDB xCluster + Skupper service interconnect |
| **App is unaware of replication** — speaks plain SQL to its local DB | `backend/` (Go + `pgx` + `chi`) |
| **DevSecOps pipeline** — test, vuln scan, lint, SAST, container scan, DAST | `pipelines/` (OpenShift Pipelines / Tekton on C1) |
| **GitOps CD across both clusters** from a single source repo | `clusters/cluster-{1,2}/_bootstrap/` (OpenShift GitOps) |
| **Cross-cluster L7 connectivity** without public DB endpoints | `clusters/cluster-{1,2}/yb-universe/` (Skupper Site/Connector/Listener) |
| **Test-first Go backend** with CRUD + 20 unit tests | `backend/internal/identity/*_test.go` |

---

## Components

| Layer | Choice | Why |
|---|---|---|
| **Database** | YugabyteDB 2025.1.4 (open-source) | PostgreSQL-wire compatible; xCluster replication is in the free Community Edition; no enterprise license server. |
| **Service interconnect** | Skupper v2 (pre-installed on the sandbox) | Carries the binary gRPC traffic Yugabyte's master/tserver speak — Routes can't proxy that. Outbound mTLS tunnel, no inbound DB ports. |
| **Backend** | Go 1.24 + `chi` + `pgx/v5` + `testify` | Smallest production-ish stack with first-class Postgres-wire support and a minimal HTTP surface. |
| **Frontend** | Plain HTML + vanilla JS + nginx-unprivileged | "No fancy" per the brief — no framework, no bundler. |
| **CI** | OpenShift Pipelines (Tekton) on cluster 1 only | Runs all DevSecOps tasks, then commits an image-tag bump back to Git. |
| **DevSecOps tools** | SonarQube · govulncheck · kube-linter · Trivy · OWASP ZAP | Coverage of SAST, SCA, container CVE, manifest lint, DAST. |
| **CD** | OpenShift GitOps (ArgoCD) on both clusters | Pulls the same repo on different paths; both clusters auto-converge. |
| **Service mesh** *(installed, enforcement in progress)* | OSSM 3 ambient (Istio + IstioCNI + ZTunnel + waypoint) | Policies authored as YAML; enforcement pending OSSM 3.1.7 RBAC fix — see `steps.md/10-service-mesh.md`. |

---

## Repository structure

```
.
├── README.md                          ← you are here
│
├── backend/                           Go service (chi + pgx)
│   ├── cmd/server/main.go
│   ├── internal/identity/             model, store, handler, pgxstore + tests
│   ├── go.mod   go.sum
│   ├── Dockerfile
│   └── sonar-project.properties
│
├── frontend/                          static site (HTML/JS/CSS) + nginx
│   ├── index.html  app.js  config.js  style.css
│   ├── nginx.conf
│   └── Dockerfile
│
├── platform/                          cluster-agnostic manifests (kustomize bases)
│   ├── namespaces/                    identity-staging, identity-db, cicd, sonarqube
│   ├── identity/base/                 backend + frontend Deployments, Services, Route
│   ├── identity/mesh-base/            PeerAuth, Waypoint Gateway, AuthZ, EnvoyFilter
│   ├── operators/                     OpenShift Pipelines + SonarQube subscriptions
│   ├── ossm/                          Istio + IstioCNI + ZTunnel ambient CRs
│   ├── argocd/                        ArgoCD CR
│   ├── sonarqube/                     SonarQube + Postgres + Route
│   └── yugabyte/                      Helm values per cluster, schema-init job, ysql wrapper Service
│
├── clusters/
│   ├── cluster-1/
│   │   ├── _bootstrap/                ArgoCD app-of-apps
│   │   ├── platform/                  overlay → ../../platform
│   │   ├── staging/identity/          identity overlay (config.js patch)
│   │   ├── staging/mesh/              mesh overlay
│   │   └── yb-universe/               Skupper Site, AccessGrant, Connectors, Listeners,
│   │                                  self-Services, xcluster-config ConfigMap
│   └── cluster-2/                     mirror of cluster-1 with -c2 names
│
├── pipelines/                         Tekton (CI on C1 only)
│   ├── pipeline.yaml                  the identity-ci Pipeline
│   ├── tasks/                         task definitions
│   ├── triggers/                      EventListener + TriggerBinding + TriggerTemplate
│   └── rbac.yaml                      pipeline SA + Triggers RBAC
│
├── scripts/
│   └── skupper-link.sh                AccessToken handshake (the only step
│                                      that can't be a static manifest)
│

```

---

## Step-by-step guide


| # | Step | What it does |
|---|---|---|
| [01](01-environment-discovery.md) | Environment discovery | Login both clusters, capture facts |
| [02](02-platform-namespaces.md)   | Platform namespaces | Namespaces + SCC grants |
| [03](03-yugabyte-helm-install.md) | Install YugabyteDB | Helm install on both clusters (1m+1t, RF=1) |
| [04](04-schema.md)                | Apply schema | `digital_identity` DDL + capture universe UUIDs |
| [05](05-skupper-link.md)          | Service interconnect | Skupper Sites + AccessToken handshake |
| [06](06-broadcast-fix.md)         | Broadcast addresses | `server_broadcast_addresses` so xCluster data flows |
| [07](07-xcluster-setup.md)        | xCluster replication | `setup_universe_replication` in both directions |
| [08](08-app-build-deploy.md)      | Build & deploy app | Build images on each cluster + apply Kustomize |
| [09](09-end-to-end-test.md)       | End-to-end test | API write on C1 → read on C2 (and reverse) |
| [10](10-service-mesh.md)          | Service mesh *(in progress)* | OSSM 3 ambient — installed, enforcement pending |
| [11](11-cicd-pipeline.md)         | CI pipeline | OpenShift Pipelines stack + secrets + webhook |
| [12](12-gitops.md)                | GitOps CD | ArgoCD apps on both clusters watching this repo |
| [13](13-end-to-end-flow.md)       | Full DevSecOps flow | git push → Tekton → image bump → both clusters roll |
| [14](14-troubleshooting.md)       | Troubleshooting | Recovery for every breakage we hit |

---

## Quick start

If you have two OpenShift clusters and the kubeconfigs at
`/tmp/ocp-kube/c{1,2}.kubeconfig`, the rough order is:

```sh
# 1. Namespaces + SCCs (both clusters)
oc apply -f platform/namespaces/namespaces.yaml

# 2. Yugabyte (Helm, both clusters)
helm install yugabyte yugabytedb/yugabyte -n identity-db \
  -f platform/yugabyte/values-cluster-1.yaml          # values-cluster-2.yaml on the other side

# 3. Skupper service interconnect (both clusters + handshake)
oc apply -k clusters/cluster-1/yb-universe
oc apply -k clusters/cluster-2/yb-universe
./scripts/skupper-link.sh

# 4. Schema + xCluster replication
#    yb-admin commands — see steps.md/04 and 07

# 5. Build + deploy the app (both clusters)
oc apply -k clusters/cluster-1/staging/identity
oc apply -k clusters/cluster-2/staging/identity

# 6. Tekton CI on cluster 1 only
oc apply -f pipelines/rbac.yaml
oc apply -f pipelines/tasks/
oc apply -f pipelines/pipeline.yaml
oc apply -f pipelines/triggers/

# 7. ArgoCD applications on each cluster
oc apply -f clusters/cluster-1/_bootstrap/applications.yaml
oc apply -f clusters/cluster-2/_bootstrap/applications.yaml
```

For the actual commands, expected output, and recovery paths, follow
[`steps.md/`](./steps.md/) top to bottom.

---

## Demo highlights

- Open both cluster UIs side by side. Add an identity on cluster 1 — it
  shows up on cluster 2 within a second. Reverse direction works the
  same.
- Push any change to `main`. A PipelineRun spawns on cluster 1, runs
  through every DevSecOps stage in ~6 minutes, and commits a tag bump
  back to the repo. Both clusters' ArgoCD pull it and roll the new
  pods.
- The database is **never** publicly reachable. Cross-cluster traffic
  flows through a single mTLS Skupper tunnel between the
  `skupper-router` pods.
