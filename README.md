# WASAText

WASAText is a messaging application. The project includes a Go HTTP backend, a SQLite database, and a Vue 3 Web UI.

## Key features
- Simplified login: creates the user if missing and rotates the token on login.
- User profile updates: change name and photo (base64).
- Direct 1:1 conversations with name and photo derived from the peer.
- Groups: create, list, inspect, add members, rename, and update photo.
- Messages with text, base64 attachments, replies, and forwarding.
- Comments or reactions on messages (one per user per message).
- Delivery and read receipts per message.
- User search by username.

## Architecture and components
### Backend (Go)
- Entry point: `cmd/webapi` starts the HTTP server, loads config, sets up logging, and initializes SQLite.
- API and logic: `service/api` with `httprouter` routes and auth middleware.
- Storage: `service/database` implements all SQLite operations and creates the schema on startup.
- CORS: permissive policy for development (origins `*`, methods GET/POST/PUT/DELETE/OPTIONS).

### Frontend (Vue 3)
- `webui` uses Vue 3, Vue Router, Axios, and Vite.
- In embedded mode the frontend is served from `/dashboard/` (build with the `webui` tag).

### Database (SQLite)
- Local file (default `/tmp/decaf.db`).
- Schema is created automatically on backend startup.

## Repository layout
- `cmd/webapi` main API server.
- `cmd/healthcheck` CLI health probe for `/liveness`.
- `service/api` HTTP handlers, auth, and response serialization.
- `service/database` models and SQLite access layer.
- `service/globaltime` time wrapper to make tests deterministic.
- `doc/api.yaml` OpenAPI specification.
- `webui` Vue 3 frontend + Vite config.
- `Dockerfile.backend`, `Dockerfile.frontend`, `docker-compose.yml` for containers.
- `open-node.sh` launches a Node 20 container for safe frontend work.

## API
Full documentation is in `doc/api.yaml`. Routes are mounted at the root (for example `http://localhost:3000/session`). The OpenAPI file uses `/v1` as an example base path, but the current backend does not include a version prefix.

### Authentication
- `POST /session` with `{ "name": "alice" }` returns `{ "token": "...", "user": { ... } }`.
- All other routes require `Authorization: Bearer <token>`.
- Tokens are stored in `users.auth_token` and rotated on each login.

### Error format
All error responses use JSON:

```json
{"error": "message"}
```

### Main endpoints (summary)
- `PUT /users/name` JSON `{ "name": "..." }`
- `PUT /users/photo` raw base64 body
- `GET /conversations`, `POST /conversations` (recipientId), `GET /conversations/{id}`
- `POST /conversations/{id}/message` multipart (content, attachment, replyTo)
- `POST /conversations/{id}/message/{msgId}/forward` JSON (targetConversationId or targetUserId, forwarderName optional)
- `DELETE /conversations/{id}/message/{msgId}`
- `POST|DELETE /conversations/{id}/message/{msgId}/comment`
- `GET /search?username=...`
- `GET|POST /groups`, `GET|DELETE|POST /groups/{id}`
- `PUT /groups/{id}/name` JSON `{ "groupName": "..." }`
- `PUT /groups/{id}/photo` raw base64 body
- `GET /liveness`

### Validation and limits
- Username: 3-16 characters, only `[a-zA-Z0-9_]`.
- Group name: 3-50 characters, `[a-zA-Z0-9_ ]`.
- Message text: max 1000 characters; content or attachment required.
- Comment or reaction content: 1-500 characters; one per user per message.
- Binary payloads: max 10 MiB; multipart max 12 MiB.
- `/search` with empty query returns up to 200 users; with a query returns up to 100 matches.

### Message responses
Each message includes:
- `reactionCount`, `reactingUserIds`, and `reactions` (aggregated by emoji).
- `deliveredTo`, `readBy`, `recipientCount`, and `deliveryStatus` (sent, delivered, read).
- `replyTo`, `replyContent`, `replySenderName`, `replyAttachment` when replying.

## Database schema
The backend creates the schema automatically on startup.

Main tables:
- `users`: id (UUID v4 without dashes), name (unique), photo, auth_token, created_at.
- `conversations`: id, name, photo, is_group, created_at.
- `conversation_members`: conversation_id, user_id, added_at.
- `messages`: id, conversation_id, sender_id, sender_name, content, attachment, created_at, reply_*, status.
- `message_comments`: message_id, user_id, content, created_at (used for comments or reactions).
- `message_status`: message_id, user_id, delivered_at, read_at.

Behavior notes:
- Message creation inserts a status row for each recipient.
- `GET /conversations` marks messages as delivered for the current user.
- `GET /conversations/{id}` marks messages as delivered and read for that conversation.

## Configuration
The backend uses `github.com/ardanlabs/conf`. Priority order: environment -> CLI -> YAML file (if present). The config file path is set via `CFG_CONFIG_PATH` (or `-config-path`).

Example `config.yml`:

```yaml
config:
  path: /conf/config.yml
web:
  apihost: "0.0.0.0:3000"
  debughost: "0.0.0.0:4000"
  readtimeout: 5s
  writetimeout: 5s
  shutdowntimeout: 5s
debug: false
db:
  filename: /tmp/decaf.db
```

Useful env vars:
- `CFG_WEB_APIHOST`
- `CFG_DB_FILENAME`
- `CFG_DEBUG`
- `CFG_CONFIG_PATH`

## Development
Backend:

```shell
go run ./cmd/webapi/
```

Frontend (first time):

```shell
./open-node.sh
# inside the container
yarn install
yarn run dev
```

## Build
Backend only (no embedded WebUI):

```shell
go build ./cmd/webapi/
```

Backend with embedded WebUI:

```shell
./open-node.sh
# inside the container
yarn run build-embed
exit
# outside the container
go build -tags webui ./cmd/webapi/
```

Standalone frontend build:

```shell
./open-node.sh
# inside the container
yarn run build-prod
```

## Docker
Start backend + frontend:

```shell
docker compose up --build
```

- Backend: `http://localhost:3000`
- Frontend: `http://localhost:8080`

## WebUI and API URL
Vite exposes `__API_URL__` using `VITE_API_URL` or `API_URL` (default `http://localhost:3000`). For embedded builds the base path is `/dashboard/`.

## Healthcheck
- `GET /liveness` returns 200 if the server is up.
- `cmd/healthcheck` probes `http://localhost:<port>/liveness`.

## Go vendoring
This project uses [Go Vendoring](https://go.dev/ref/mod#vendoring). After changing dependencies, run `go mod vendor` and commit `vendor/`.

## Node and Yarn vendoring
The frontend uses `yarn` with offline mirror caching. Files under `.yarn/` should be committed.

## License
See [LICENSE](LICENSE).
