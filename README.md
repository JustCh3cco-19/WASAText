# WASAText

WASAText is a messaging application built with a Go HTTP API, SQLite, and a Vue 3 Web UI. It supports direct and group conversations, attachments, replies, forwarding, reactions, and delivery/read receipts.

## Quick start

Docker is the simplest way to run the complete application:

```shell
docker compose up --build
```

Open:

- Web UI: `http://localhost:8080`
- API: `http://localhost:3000`
- Liveness: `http://localhost:3000/liveness`
- Readiness: `http://localhost:3000/readiness`
- Metrics: `http://localhost:3000/metrics`

SQLite data is persisted in the `wasatext-data` named volume. Stop the application without deleting its data with:

```shell
docker compose down
```

## Features

- Password-protected registration and login.
- Direct one-to-one conversations.
- Group creation, membership, name, and photo management.
- Text messages and base64-encoded attachments.
- Replies and message forwarding.
- One reaction or comment per user and message.
- Per-recipient delivery and read receipts.
- Username search.
- Paginated users, conversations, and messages.
- Responsive Vue Web UI.

## Architecture

### Backend

- `cmd/webapi`: configuration, SQLite connection, HTTP server, CORS, and graceful shutdown.
- `cmd/healthcheck`: lightweight container health probe.
- `service/api`: routing, authentication, validation, serialization, rate limiting, and observability.
- `service/application`: application boundary between transport and persistence.
- `service/database`: SQLite schema, migrations, models, queries, and authorization checks.
- `service/globaltime`: controllable time wrapper used by deterministic code and tests.

The API uses `httprouter`; SQLite runs with foreign keys enabled, WAL mode, a busy timeout, and a single pooled connection to preserve connection-scoped settings.

### Frontend

The `webui` directory contains a Vue 3 SPA using Vue Router, Axios, Vite, and npm. The standalone image serves the compiled application through an unprivileged Nginx process.

### API contract

The OpenAPI specification is available at [doc/api.yaml](doc/api.yaml) and is validated by Spectral in CI.

## Authentication and security

Register an account:

```http
POST /users
Content-Type: application/json

{"name":"alice","password":"a-secure-password"}
```

Log in with the same payload:

```http
POST /session
```

Successful registration and login set a `wasatext_session` cookie with `HttpOnly`, `SameSite=Lax`, and a 24-hour expiry. The Web UI never receives or stores the underlying bearer token. Logout revokes the server-side session:

```http
DELETE /session
```

Security measures include:

- salted PBKDF2-HMAC-SHA256 password hashes;
- opaque session tokens stored only as SHA-256 hashes;
- token rotation on login and explicit revocation on logout;
- rate limiting on registration and login;
- strict request-size limits and JSON unknown-field rejection;
- base64 and image signature validation;
- configurable CORS allow-list;
- CSP and other browser security headers;
- panic recovery and generic internal-error responses;
- non-root backend and frontend containers.

Legacy accounts created before password authentication have no usable password and are deliberately invalidated by the migration. They must be recreated or reset through an administrative recovery procedure.

## Main API endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/users` | Register an account |
| `POST` | `/session` | Log in and rotate the session |
| `DELETE` | `/session` | Revoke the current session |
| `PUT` | `/users/name` | Change the current username |
| `PUT` | `/users/photo` | Change the current profile photo |
| `GET` | `/search` | Search users |
| `GET`, `POST` | `/conversations` | List or start conversations |
| `GET` | `/conversations/{id}` | Read a conversation and its messages |
| `POST` | `/conversations/{id}/message` | Send a message |
| `POST` | `/conversations/{id}/message/{messageId}/forward` | Forward a message |
| `DELETE` | `/conversations/{id}/message/{messageId}` | Delete a sent message |
| `POST`, `DELETE` | `/conversations/{id}/message/{messageId}/comment` | Add or remove a reaction/comment |
| `GET`, `POST` | `/groups` | List or create groups |
| `GET`, `POST`, `DELETE` | `/groups/{id}` | Inspect, add a member, or leave a group |
| `PUT` | `/groups/{id}/name` | Rename a group |
| `PUT` | `/groups/{id}/photo` | Change a group photo |
| `GET` | `/liveness` | Process health probe |
| `GET` | `/readiness` | SQLite readiness probe |
| `GET` | `/metrics` | Prometheus-compatible metrics |

Errors consistently use:

```json
{"error":"message"}
```

### Pagination

Collection endpoints accept non-negative `offset` and a bounded `limit`:

```http
GET /conversations?limit=25&offset=0
GET /conversations/{id}?limit=100&offset=0
GET /search?username=ali&limit=50&offset=0
```

Defaults and maximums:

| Resource | Default | Maximum |
|---|---:|---:|
| Conversations | 50 | 100 |
| Messages | 100 | 200 |
| Users | 50 | 200 |

### Validation limits

- Username: 3–16 characters, matching `[a-zA-Z0-9_]+`.
- Password: 10–128 characters.
- Group name: 3–50 characters, matching `[a-zA-Z0-9_ ]+`.
- Message text: at most 1,000 characters; content or attachment is required.
- Reaction/comment: 1–500 characters.
- Binary payload: at most 10 MiB.
- Multipart request: at most 13 MiB including encoding overhead.
- Profile and group photos: valid base64 PNG, JPEG, or GIF.

## Configuration

Configuration is loaded with `github.com/ardanlabs/conf`. Environment variables use the `CFG_` prefix. Command-line arguments override environment defaults, while an explicitly loaded YAML file is applied last.

```yaml
config:
  path: /conf/config.yml
web:
  apihost: "0.0.0.0:3000"
  debughost: "0.0.0.0:4000"
  readtimeout: 5s
  writetimeout: 5s
  shutdowntimeout: 5s
  corsorigins: "http://localhost:8080"
debug: false
db:
  filename: /data/wasatext.db
  busytimeout: 5s
```

Common environment variables:

- `CFG_CONFIG_PATH`
- `CFG_WEB_APIHOST`
- `CFG_WEB_CORSORIGINS` — comma-separated origins
- `CFG_DB_FILENAME`
- `CFG_DB_BUSYTIMEOUT`
- `CFG_DEBUG`

## Local development

Requirements:

- Go 1.23;
- Node.js 22.18 or newer and npm 10.9;
- a C compiler and SQLite development support for `go-sqlite3`;
- Docker for container and E2E workflows.

Run the backend:

```shell
go run ./cmd/webapi
```

Run the frontend:

```shell
cd webui
npm ci
npm run dev
```

If the correct Node version is not installed locally, `./open-node.sh` opens a matching Node container in `webui`.

The frontend API URL is selected through `VITE_API_URL` or `API_URL` and defaults to `http://localhost:3000`.

## Build

```shell
# Backend
go build ./cmd/webapi

# Standalone frontend
cd webui
npm ci
npm run build-prod

# Backend with embedded frontend
npm run build-embed
cd ..
go build -tags webui ./cmd/webapi
```

## Tests and quality gates

Backend:

```shell
go test -race ./...
go vet ./...
```

Frontend:

```shell
cd webui
npm ci
npm run lint
npm run build-prod
npm audit --audit-level=high
```

OpenAPI:

```shell
npx @stoplight/spectral-cli lint doc/api.yaml
```

End-to-end, with the Docker stack already running:

```shell
./scripts/e2e.sh
```

The E2E test verifies registration, cookie authentication, pagination, logout, token revocation, the frontend, and browser security headers. All gates run automatically through [.github/workflows/ci.yml](.github/workflows/ci.yml).

## Dependency management

Go dependencies are vendored. After changing `go.mod` or `go.sum`:

```shell
go mod tidy
go mod vendor
```

Frontend dependencies use npm exclusively. Commit `webui/package-lock.json`; do not add Yarn lockfiles or offline caches.

## License

See [LICENSE](LICENSE).
