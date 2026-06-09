# Architecture

## System Goal

Build an Omni Channel Chat Platform that receives conversations from multiple messaging channels and exposes one controlled operations workspace for staff, managers, auditors, and admins.

Primary flow:

```txt
Channel Gateway
-> Golang API
-> MongoDB
-> RabbitMQ
-> Redis
-> Socket realtime
-> CDN media storage
-> NextJS frontend
-> Docker Compose production
```

## Module Boundaries

- Channel Gateway: receive webhook or polling output, validate signatures/sessions, save raw events, publish queue jobs.
- Core API: auth, RBAC, users, teams, channels, conversations, messages, media metadata, audit logs.
- Inbound Worker: normalize events, upsert conversations/messages, emit socket events, enqueue media jobs.
- Outbound Worker: send messages through channel adapters, update delivery status, retry safely.
- Media Worker: download channel media, upload to CDN, update attachment status.
- Socket Service: emit conversation/message/status updates, track presence, support reconnect sync.
- Admin Frontend: manage users, roles, teams, channel accounts, routing, reports.
- Inbox Frontend: list conversations, show realtime chat, compose messages, preview attachments, assign/transfer conversations.

## Core Principles

- Keep core chat channel-agnostic. Channel-specific payloads stay in adapters and raw event collections.
- Persist raw inbound/outbound events for replay and audit.
- Treat sockets as acceleration, not source of truth. REST sync remains authoritative.
- Design every conversation access path with RBAC and team scope.
- Prefer async media handling so webhook and message visibility are not blocked by slow downloads.

## Key Trade-offs

- MongoDB fits flexible channel payloads and message metadata, but enforce indexes and idempotency keys carefully.
- RabbitMQ gives reliable queueing, retries, and DLQ, but requires visibility into backlog and consumer health.
- Redis is useful for short-lived state, but do not store authoritative messages only in Redis.
- Unofficial channel libraries can enable personal-account channels, but must be explicitly marked as operational and ToS risk.

## Expected Local Layout

```txt
backend/
  cmd/api
  cmd/worker
  internal/auth
  internal/rbac
  internal/channel
  internal/conversation
  internal/message
  internal/queue
  internal/socket
  internal/media
  internal/config
  internal/database
  pkg
frontend/
  src/app
  src/components
  src/features
  src/lib
  src/store
  src/types
  src/utils
cdn/
```

## Local Commands

Recommend a `Makefile` with:

```bash
make dev
make backend
make frontend
make worker
make docker-up
make docker-down
make logs
```
