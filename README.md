# OIDC Platform

Current version: **v1.13**

A self-hosted, developer-oriented identity platform built on top of OpenID Connect / OAuth 2.0. Users register with email and bind third-party accounts (GitHub, Gitee, GitLab, Discord, Google, Microsoft, Apple, Telegram, QQ, WeChat, phone, etc.) to raise their **trust level**. Downstream systems integrate once via standard OIDC and gain low-cost risk control by declaring a minimum trust level plus optional conditions.

> One-liner: **a drop-in risk-control layer for your apps, served over standard OIDC.**

中文说明请见 [README.zh-CN.md](./README.zh-CN.md)。

## Goals

- **Standards first.** Full OIDC (Authorization Code, Refresh Token, ID Token) and OAuth 2.0 endpoints. Downstream systems integrate via spec, not via platform-specific SDKs.
- **Centralised trust.** A user binds accounts once on this platform; every connected system shares the same trust view.
- **Zero risk-control work for relying parties.** A relying party only declares "I require Lv3 + bound phone + GitHub age ≥ 30 days". Validation, prompts, and refusal all happen here.
- **Closed-loop abuse feedback.** Relying parties report abusive accounts; the platform lowers the user's trust level and pushes the bound identifiers into a shared risk list visible to other relying parties.
- **Restrictions live here.** Allowlists, alias rules, email-domain rules, IP/region risk are all configured once on the platform instead of being re-implemented in every app.

## Core Concepts

### Trust Level
Each user has a dynamic trust level (e.g. Lv0–Lv5) derived from bound provider types, binding age, email/phone verification, MFA status, and risk history. Each level's threshold is configurable, for example:

- **Lv1** — verified email
- **Lv2** — Lv1 + any bound social account
- **Lv3** — Lv2 + bound phone + TOTP enabled
- **Lv4** — Lv3 + GitHub age ≥ 90 days, or bound WeChat / Apple
- **Lv5** — Lv4 + manual review

### Access Rule
A relying party declares, in the developer portal:

- minimum trust level
- required bindings (e.g. phone must be bound)
- extra conditions (GitHub age, allowed email domains, region, alias allow/deny lists)

At login time the platform evaluates the rule, prompts the user to add missing bindings if needed, and only issues an ID Token when the policy is satisfied.

### Risk Feedback
Relying parties can report abuse via `POST /api/v1/risk/report`. On report the platform:

- lowers the user's trust level
- adds their bound identifiers (GitHub login, phone hash, device fingerprint, …) to a shared risk list
- other relying parties see the updated trust score on the next login

### One-tap Login
Beyond standard OIDC, the platform offers a "bind once, sign in everywhere" experience. A user who already bound GitHub on system A can hit the GitHub button on system B; the platform reuses the same identity and applies system B's access rule.

## Supported Bindings

| Region / type | Providers |
| ------------- | --------- |
| Global        | Google, GitHub, GitLab, Microsoft, Apple, Discord, Telegram |
| China         | Gitee, QQ, WeChat, Linux DO |
| Generic       | Email, phone (SMS), TOTP / MFA, Passkey |

> Each provider is pluggable; enable / disable and configure credentials independently in the admin console.

## Feature Modules

- **User account center** — email register / login, password recovery, email verification, third-party binding & unbinding, session management, authorized apps, recent activity, trust-level view, TOTP / MFA, and Passkey management.
- **Developer portal** — self-service app creation, redirect URI and secret management, access-rule configuration, authorized-user management, user blocking, and abuse reporting.
- **Admin console** — users, clients, authorized users, social providers, per-provider login/register toggles, signing-key rotation, audit log, risk policy, risk list, security rules, system settings, version/update check, alias / allowlist / email-domain restrictions.
- **OIDC / OAuth server** — built on [ory/fosite](https://github.com/ory/fosite); ships `/.well-known/openid-configuration`, `/authorize`, `/token`, `/userinfo`, `/jwks.json`.
- **Risk & security** — login lockout, Cloudflare Turnstile / hCaptcha captcha support, platform risk blocking, rate limiting, request audit, password policy, Passkeys, periodic key rotation.

## Tech Stack

- **Backend** — Go 1.23+, [chi](https://github.com/go-chi/chi) router, [ory/fosite](https://github.com/ory/fosite), [viper](https://github.com/spf13/viper), pgx / modernc.org/sqlite.
- **Storage** — SQLite (default, zero-dep single file) or PostgreSQL + Redis (production).
- **Frontend** — Vue 3 + Vite + TypeScript + Pinia. One bundle hosts the user center, developer portal, and admin console.
- **Deployment** — recommended Docker image deployment from GHCR, plus single binary + frontend dist for manual/local use.

## Repository Layout

```
cmd/server/            # entrypoint and dependency wiring
configs/               # default config.yaml and config.sqlite.yaml
db/migrations/         # SQL migrations (SQLite + PostgreSQL)
db/queries/            # sqlc query definitions
internal/adapter/      # storage and external service adapters
internal/domain/       # domain types
internal/handler/      # HTTP handlers and middleware
internal/oidcprovider/ # OIDC / OAuth server wrapper
internal/port/         # interface definitions
internal/router/       # chi router wiring
internal/service/      # business services
frontend/              # Vue 3 SPA (admin + user portal)
data/                  # runtime SQLite database (gitignored)
```

## Quick Start

### Recommended: Docker image

The recommended deployment path is to pull the published GHCR image and run it with PostgreSQL + Redis via Docker Compose.

```bash
cp .env.example .env
# Edit .env before production use, especially issuer/public URL, admin password, and encryption key.
docker compose pull
docker compose up -d
```

Default image:

```text
ghcr.io/luotianyi-0712/openoidc:latest
```

Use a fixed tag instead of `latest` when you want a pinned release:

```bash
OIDC_IMAGE=ghcr.io/luotianyi-0712/openoidc:v1.13 docker compose up -d
```

The server listens on `http://localhost:8080` by default. PostgreSQL and Redis are not exposed to the host by default; only the app container can reach them on the Compose network. Change `OIDC_SERVER_ISSUER`, `OIDC_SERVER_PUBLIC_URL`, `OIDC_ADMIN_PASSWORD`, and `OIDC_SECRETS_CLIENT_SECRET_ENCRYPTION_KEY` in `.env` before any non-local use.

### Source / single-binary build

Requires Go 1.23+ and Node 18+.

```bash
cd frontend
npm install
npm run build
cd ..

go build -o oidc ./cmd/server
./oidc    # uses configs/config.yaml (SQLite by default)
```

On Windows you can also run `start.bat` or build `oidc.exe`.

## Configuration

- `configs/config.yaml` — main config, supports both SQLite and PostgreSQL.
- `configs/config.sqlite.yaml` — minimal SQLite-only override.
- `.env` — environment overrides loaded by Docker Compose; see `.env.example`.

Every YAML key can be overridden via `OIDC_*` environment variables, e.g. `OIDC_DATABASE_DRIVER=postgres`.

Files named `configs/config.local.yaml` or `configs/config.*.local.yaml` are gitignored and intended for machine-specific overrides.

## Development

```bash
# Live reload (requires air)
air

# Front-end dev server
cd frontend && npm run dev
```

## Migrations

Migrations under `db/migrations` are applied automatically on startup. Regenerate sqlc code after editing `db/queries/*.sql`:

```bash
sqlc generate
```

## Roadmap

- [x] Email register / login / recovery
- [x] OIDC / OAuth 2.0 standard endpoints
- [x] GitHub / Google / GitLab / Gitee / Linux DO / Microsoft / Discord / Apple / Telegram / QQ / WeChat / phone bindings
- [x] Multi-level trust model
- [x] Per-app access rule (min level + required bindings + extra conditions)
- [x] Alias / email-domain / IP / region restrictions
- [x] Abuse reporting, admin review, and shared risk list
- [x] Platform risk policy and blocking controls
- [x] Cloudflare Turnstile / hCaptcha captcha support
- [x] WebAuthn / Passkey management
- [x] User activity history and admin audit trails
- [x] Version display and release update check
- [x] Docker image workflow and GHCR deployment path
- [ ] Multi-tenant isolation
- [ ] SDKs: Go / Node / Python integration samples

## License

Private project. All rights reserved.
