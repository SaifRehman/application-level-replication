# Cross-cluster YugabyteDB demo on OpenShift

A two-OpenShift-cluster demo of database-level bidirectional replication
(YugabyteDB xCluster) with a digital-identity CRUD app, OpenShift
Pipelines CI, OpenShift GitOps CD, and OSSM 3 ambient-mode service
mesh.

The full design is at
`docs/superpowers/specs/2026-04-26-cross-cluster-yugabyte-demo-design.md`.

## Project structure

```
backend/                 Go service (chi + pgx)
frontend/                vanilla HTML/JS/CSS, served by nginx
pipelines/               Tekton pipeline + tasks + triggers
platform/                shared platform manifests (kustomize base)
clusters/cluster-1/      cluster 1 GitOps tree (overlays + apps)
clusters/cluster-2/      cluster 2 GitOps tree (overlays + apps)
docs/                    design + runbooks
```

## Build and test

```
cd backend
go test ./...                 # unit + integration
go vet ./...
go build ./cmd/server         # produces ./server binary

cd frontend
docker build -t identity-frontend .   # static-only, no JS build step
```

## Conventions

- **No comments in code.** Naming and structure must carry intent. The
  only allowed exceptions: license headers (none here), `//go:embed`
  directives, and required toolchain directives (e.g. `// +build`).
- **No frameworks on the frontend.** Plain HTML, plain JS, plain CSS.
  No bundler, no transpiler.
- **Backend uses Go 1.24, `chi`, `pgx/v5`, and `testify`.** Don't
  introduce other routers, ORMs, or test runners.
- **YugabyteDB speaks the Postgres wire protocol.** Use `pgx`, not a
  Yugabyte-specific driver.
- **Service mesh enforces authn/authz/rate-limit.** Application code
  does not check the `X-API-Key` header — that's the waypoint's job.
- **One identity table.** `digital_identity` is the only schema entity.
  Don't add tables, joins, or transactions.
- **Demo-grade only.** No HPA, PDBs, backups, or human RBAC. Don't add
  them; the design explicitly defers them.

## Doing things

- Always run `go test ./...` from `backend/` before considering a Go
  change done. Coverage gate (in CI) is 70 %.
- Always run `kube-linter lint platform/ pipelines/ clusters/` before
  committing manifest changes.
- Every Go function/method that has logic gets a test in the same
  package. Pure-data structs don't need tests.
- Manifests live under either `platform/` (cluster-agnostic base) or
  `clusters/cluster-N/` (overlay). Never inline a manifest somewhere
  else.

## Don't do

- Don't add comments. If the code needs one, rename or restructure.
- Don't add a build step to the frontend.
- Don't add app-level auth checks. Service mesh handles it.
- Don't push images to a public registry; pipelines push to cluster
  1's exposed internal OCP registry only.
- Don't introduce CockroachDB anywhere. The demo uses YugabyteDB only.
- Don't use `database/sql`; use `pgx/v5` directly.
- Don't pin to specific cluster hostnames in `platform/` — only in
  `clusters/cluster-*/` overlays.

## Running locally (smoke check before pushing manifests)

```
cd backend
DATABASE_URL=postgres://yugabyte:yugabyte@localhost:5433/yugabyte \
  go run ./cmd/server
```

Open `frontend/index.html` directly in a browser, or
`docker run -p 8080:8080 identity-frontend` to serve it via nginx.
The frontend reads its API URL and API key from
`window.__CONFIG__` set by `app.js` from a `<meta>` tag rendered in
`index.html`.
