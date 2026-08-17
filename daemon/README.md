# hoist daemon (`hoistd`)

The `hoistd` daemon is the hoist control plane. It runs on your target server and turns a `git push` into a live, TLS-terminated, zero-downtime deployment.

It receives source code over SSH, builds apps in isolated Docker containers with a canonical SDK, manages container lifecycles, and flips traffic atomically via Caddy.

> **Status**: Phase 2 — in active development.

---

## Architecture

`hoistd` is a single self-contained binary composed of seven internal packages:

```
daemon/
├── cmd/hoistd/        # Entrypoint
├── internal/
│   ├── config/        # Daemon configuration (hoistd.json)
│   ├── state/         # BoltDB state: apps, deployments, SSH keys
│   ├── gitserver/     # Embedded SSH server, bare repo management, push detection
│   ├── builder/       # Isolated Docker builds, canonical SDK injection, dep scanning
│   ├── runtime/       # Docker client wrapper: containers, limits, health, logs
│   ├── proxy/         # Caddy admin API client, route management
│   ├── deploy/        # Blue-green deploy orchestrator
│   └── api/           # Daemon HTTP API (status, log streams)
└── images/
    ├── builder/       # hoist-builder image: Go toolchain + canonical SDK
    └── runtime/       # Runtime image assembly (scratch + binary)
```

The daemon never imports `hoist/sdk`. The SDK is embedded in user apps; the daemon only *parses* `hoist.json` from pushed source to learn each app's port and health check path.

---

## Deploy Flow

```
git push ssh://git@your-server:2222/my-api.git
  ↓
gitserver: receives push into bare repo, detects default branch moved
  ↓
deploy: enqueues deployment (per-app mutex — one deploy per app at a time)
  ↓
builder: git archive <sha> → build container (resource-limited, timeout-bounded)
  - Injects canonical SDK: go.mod replace → /opt/hoist/sdk
  - go mod tidy && go build
  - Dependency scan (govulncheck), results logged
  - Assembles runtime image: scratch + binary + CA certs + nonroot user
  - Tagged hoist/app-my-api:<sha>
  ↓
runtime: starts green container alongside live blue
  - CPU/memory limits, labels, no published ports
  - Polls http://<container-ip>:<port><health_path> until healthy
  ↓
proxy: PUT to Caddy admin API → atomic upstream flip to green
  ↓
runtime: SIGTERM old blue container → SDK graceful shutdown (30s grace) → removed
```

**Failure at any step before the flip** → green is removed, blue stays live, deployment marked failed with logs retained. Failed builds never touch running traffic.

---

## Components

### Git Server

- Embedded SSH server (`gliderlabs/ssh`), default port **2222** — no system SSHD or `git` user setup required.
- Handles `git-receive-pack '<app>.git'` by shelling out to the system `git` binary against bare repos in `<data-dir>/repos/`.
- **Auto-provisioning**: pushing to a new (valid) app name creates the app on first push. Names must be DNS-safe — they become subdomains.
- **Push detection**: diffs repo refs before/after receive; a moved default branch triggers a deployment.
- Public key auth against keys in the state store (Phase 2: `authorized_keys`-style file in the data dir; real multi-user auth is Phase 4).

### Build System

- Builds happen inside a `hoist-builder` container: Go toolchain + canonical SDK source baked in at `/opt/hoist/sdk`. The daemon builds/verifies this image on startup.
- **Canonical SDK enforcement**: the build rewrites the app's `go.mod` with a `replace` directive pointing at the baked-in SDK. Users can't ship a modified SDK because they don't control the build.
- **Dependency scanning**: `go list -m all` is recorded per deployment; `govulncheck` runs and its results are logged (optionally failing the build — see config).
- Runtime images are assembled via the Docker API with an in-memory tar context — no Dockerfile ever touches the user's machine.

### Container Lifecycle

- All app containers run on a dedicated `hoist-apps` Docker bridge network with no published ports. Caddy (on the host) reaches them via their bridge IPs.
- Containers start with:
  - Resource limits (CPU/memory, from daemon config)
  - Labels: `hoist.app`, `hoist.color`, `hoist.deployment`, `hoist.managed=true`
  - `HOIST_*` environment defaults (the app's committed `hoist.json` takes precedence, per SDK config precedence)
- **Health polling**: `GET http://<container-ip>:<port><health_path>` every 500ms until HTTP 200 or timeout.
- **Log capture**: container stdout/stderr is streamed to per-app log files in the data dir, exposed via the HTTP API.

### Proxy

- Talks to a host-installed Caddy via its admin API (default `http://localhost:2019`).
- One route per app: `<app>.<base_domain>` → reverse proxy to the current container's `IP:port`, managed under a stable route ID.
- **Traffic flip is atomic**: a single admin API call swaps the upstream; Caddy drains in-flight connections.
- TLS: Caddy's internal CA serves `*.hoist.local`-style dev domains. Point `base_domain` at a real domain and Caddy obtains real ACME certificates automatically.

### State

- Embedded BoltDB (single file in the data dir), no external database.
- Buckets:
  - `apps` — live container ID, active color, parsed `hoist.json` snapshot
  - `deployments` — id, git SHA, status, timestamps, log paths
  - `keys` — authorized SSH public keys
- Last N deployment images are retained per app (future rollback support); older ones are pruned.

---

## Configuration

`hoistd` reads `hoistd.json` (searched in `./hoistd.json`, then `/etc/hoist/hoistd.json`):

```json
{
  "ssh_addr": ":2222",
  "http_addr": "127.0.0.1:7575",
  "docker_host": "unix:///var/run/docker.sock",
  "caddy_admin_url": "http://localhost:2019",
  "base_domain": "hoist.local",
  "data_dir": "/var/lib/hoist",
  "default_cpu_limit": 0.5,
  "default_memory_mb": 256,
  "build_timeout_sec": 600,
  "health_timeout_sec": 30,
  "fail_build_on_vulns": false,
  "retained_images": 3
}
```

All fields optional; shown values are the defaults. For development, point `data_dir` at a local directory.

---

## HTTP API

Minimal API bound to localhost by default — primarily the foundation for the Phase 3 CLI. No auth in Phase 2 (auth lands in Phase 4).

| Endpoint                                   | Description                              |
| ------------------------------------------ | ---------------------------------------- |
| `GET /apps`                                | List all apps and their status           |
| `GET /apps/{name}`                         | App detail: live color, container, config |
| `GET /apps/{name}/deployments`             | Deployment history                       |
| `GET /apps/{name}/deployments/{id}/logs`   | Build log for a deployment               |
| `GET /apps/{name}/logs`                    | Stream app logs (follow)                 |

---

## Security Model

- **Isolation**: every app runs in its own container with CPU/memory limits and no host port exposure.
- **Server-side builds**: users push source, never binaries. The build environment is daemon-controlled.
- **Canonical SDK**: baked into the builder image and forced via `go.mod` replace — user code cannot swap in a modified SDK.
- **SSH key auth**: git access is public-key only; there are no passwords.
- Phase 4 adds multi-user auth on the HTTP API, rate limiting, and resource quotas per user.

---

## Development

```sh
# Unit tests (Docker/Caddy clients are behind interfaces, faked in tests)
go test ./...

# Integration tests (require Docker; gated by the 'integration' build tag)
go test -tags integration ./...
```

Integration tests run the real end-to-end flow: push `examples/basic-api` to a local daemon, build, deploy, and hammer requests during a redeploy to verify zero downtime.
