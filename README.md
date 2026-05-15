# OIDC Platform

A self-hosted OpenID Connect / OAuth 2.0 identity provider written in Go, with a Vue 3 admin and end-user front end. Supports SQLite for single-binary deployments and PostgreSQL + Redis for production.

## Features

- OIDC / OAuth 2.0 server built on top of [ory/fosite](https://github.com/ory/fosite)
- Authorization code, refresh token, ID token, and client credentials flows
- Pluggable storage: SQLite (default, zero-dep) or PostgreSQL + Redis
- Social login providers: Google, GitHub, GitLab, Gitee, Microsoft, Apple, QQ, WeChat, Discord, Telegram, phone (SMS), and more
- Admin console (Vue 3 + Vite) for users, clients, providers, signing keys, audit logs, risk and security rules
- Self-service developer portal: register applications, manage redirect URIs and secrets
- End-user account center: sessions, social bindings, MFA, authorized apps
- JWT signing key rotation
- Rate limiting, Turnstile (Cloudflare) protection, audit logging, risk scoring

## Repository Layout

```
cmd/server/          # entrypoint (main.go) and bootstrap wiring
configs/             # default config.yaml and config.sqlite.yaml
db/migrations/       # SQL migrations (SQLite + PostgreSQL)
db/queries/          # sqlc query definitions
internal/adapter/    # storage and external service adapters
internal/domain/     # domain types
internal/handler/    # HTTP handlers and middleware
internal/router/     # chi router wiring
frontend/            # Vue 3 SPA (admin + user portal)
data/                # runtime SQLite database (gitignored)
```

## Quick Start

### SQLite (single binary)

Requires Go 1.23+ and Node 18+.

```bash
# 1. Build the front end
cd frontend
npm install
npm run build
cd ..

# 2. Build and run the server
go build -o oidc-platform ./cmd/server
./oidc-platform   # uses configs/config.yaml (SQLite by default)
```

Or on Windows:

```cmd
start.bat
```

The server listens on `http://localhost:8080` by default. The admin login is configured in `configs/config.yaml` (`admin.email` / `admin.password`) — change it before any non-local use.

### PostgreSQL + Redis (Docker Compose)

```bash
cp .env.example .env
docker compose up --build
```

This starts PostgreSQL, Redis, and the OIDC server on port 8080.

## Configuration

- `configs/config.yaml` — main config, supports both SQLite and PostgreSQL
- `configs/config.sqlite.yaml` — minimal SQLite-only override
- `.env` — environment overrides (loaded by Docker Compose; see `.env.example`)

Every YAML key can be overridden via `OIDC_*` environment variables, e.g. `OIDC_DATABASE_DRIVER=postgres`.

### Local overrides

Files named `configs/config.local.yaml` or `configs/config.*.local.yaml` are gitignored and intended for machine-specific tweaks.

## Development

```bash
# Run the server with live reload (requires air)
air

# Run the front end dev server
cd frontend && npm run dev
```

Front-end dev server proxies API requests to the Go server (see `frontend/vite.config.ts`).

## Migrations

Migrations live in `db/migrations` and are applied automatically on startup. To regenerate sqlc code after editing `db/queries/*.sql`:

```bash
sqlc generate
```

## License

Private project. All rights reserved.
