---
name: omni-channel-platform-architect
description: Use this skill whenever the user is designing, building, debugging, reviewing, or extending an Omni Channel chat platform using Golang, MongoDB, RabbitMQ, Redis, Socket realtime, NextJS frontend, CDN media storage, RBAC permissions, and channel integrations such as WhatsApp, Telegram, Facebook, Zalo, or similar messaging platforms. This skill should trigger even if the user only mentions inbound queue delay, webhook reliability, chat assignment, manager visibility, channel gateway, local/docker deployment, or multi-agent chat operations.
---

# Omni Channel Platform Architect

## 1. Project Context

Use this skill to design, scaffold, review, debug, and operate an Omni Channel Chat Platform:

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

The target platform centralizes conversations from multiple channels, gives each staff member their own inbox, allows managers to view assigned subordinates' conversations, and gives admins full control over users, roles, teams, channels, and routing rules. Always account for webhook delay, retry, duplicate events, queue backlog, socket disconnects, and local plus Docker Compose deployment. The project already has a `cdn` service for chat/user media.

## 2. Quick Reference Map

Load only the relevant reference file:

- Architecture, module boundaries, and trade-offs: `references/architecture.md`
- Inbound webhook, queue, idempotency, retry, DLQ, sync: `references/inbound-queue.md`
- MongoDB collections, fields, indexes, JSON examples: `references/database-design.md`
- RBAC, team visibility, conversation/media checks, audit: `references/rbac-permission.md`
- Channel adapter pattern and platform risks: `references/channel-adapters.md`
- Media worker and CDN path design: `references/media-cdn.md`
- REST/WebSocket API design checklist: `references/api-design.md`
- NextJS frontend modules and realtime sync behavior: `references/frontend-nextjs.md`
- Docker Compose, env, healthcheck, production notes: `references/docker-production.md`
- Metrics, logs, debugging, eval guidance: `references/testing-observability.md`

## 3. Technology Stack

Backend:

- Golang with REST API or REST plus WebSocket.
- MongoDB as the primary conversation/message store.
- RabbitMQ for inbound/outbound queues, retries, and DLQ.
- Redis for cache, distributed lock, session, socket presence, and rate limit.
- Socket realtime for new messages and status updates.

Frontend:

- NextJS, React, TypeScript.
- Inbox, users, roles, teams, channels, conversations, and media management.

Infrastructure:

- Local development and production using Docker Compose.
- `.env.example`, healthchecks, optional init/migration scripts, structured logs, and basic tracing.

## 4. Mandatory Design Rules

- Design channels with adapter pattern. Do not hard-code channel-specific behavior into core chat services.
- Every implementation run must include unit tests and document completed modules under `docs/` using numbered files such as `1-module-name.md`, `2-next-module.md`, and so on.
- Use official APIs when available. For personal/unofficial connectors, explicitly state technical risk, ToS risk, login/session loss, checkpoint, captcha, rate limit, and account block risk.
- Do not design spam, unauthorized scraping, credential bypass, security bypass, or systems that process accounts/channels the owner cannot legally manage.
- Make inbound processing idempotent with raw event storage, dedupe keys, retry queues, DLQ, and sync recovery.
- Do not wait for media upload inside webhook handling. Store message first, then process attachment asynchronously.
- Never rely only on sockets. On reconnect or conversation open, call API sync to fetch missed messages.
- When designing API, database, or UI behavior, always check conversation-level and attachment-level permissions.
- Never commit real secrets. `.env.example` must contain placeholders only.

## 5. Core Workflows

### Architecture Requests

Answer with:

- Flow diagram.
- Module list and ownership boundaries.
- Database and queue shape.
- Trade-offs and failure modes.
- References to exact observability points.

Read `references/architecture.md`, then any domain file that matches the request.

### Code Implementation Requests

Prefer Golang backend and NextJS frontend patterns:

- Split code by domain; do not create one large file.
- Include error handling, context timeout, validation, permission checks, and audit hooks.
- Use repository/service/handler separation for backend.
- Use feature modules, typed API clients, and socket reconnect sync for frontend.
- Use existing repo patterns if the user is editing an existing codebase.

Use `scripts/scaffold-backend.sh` and `scripts/scaffold-frontend.sh` only when the user asks to scaffold.

### Inbound Delay Debugging

Analyze in this order:

```txt
Channel -> Gateway -> RabbitMQ -> Worker -> MongoDB -> Socket -> Frontend
```

Check:

- `event_time`, `gateway_received_at`, `queued_at`, `processed_at`, `socket_emitted_at`, `agent_seen_at`.
- Queue ready/unacked counts, worker errors, duplicate keys, DLQ, media job status.
- Whether conversation-open sync or periodic sync can recover missed/delayed events.

Read `references/inbound-queue.md` and `references/testing-observability.md`.

### Permission Requests

Always state:

- What Admin, Manager, Staff, Supervisor, and Auditor can do.
- Which team/subordinate relationship grants manager visibility.
- Whether the user can view conversation, send message, view attachment, reassign chat, or view audit logs.
- Which actions produce audit records.

Read `references/rbac-permission.md`.

### Channel Adapter Requests

Always provide:

- Common adapter interface.
- One connector implementation per channel.
- Normalized inbound/outbound message contracts.
- Risk notes for unofficial/personal connectors.
- Rate-limit, session-health, reconnect, and webhook/polling strategy.

Read `references/channel-adapters.md`.

## 6. Scaffold Scripts

- `scripts/scaffold-backend.sh <target-dir>` creates a minimal Golang backend skeleton.
- `scripts/scaffold-frontend.sh <target-dir>` creates a minimal NextJS/TypeScript skeleton.
- `scripts/docker-compose.example.yml` provides a safe compose reference for production-like local deployment.

Scripts create placeholders only. They do not implement business logic until the user asks for a concrete implementation.

## 7. Evaluation Prompts

Use `evals/evals.json` to test trigger behavior and output quality. Trigger prompts cover inbound delay, MongoDB design, RBAC, Docker Compose, channel adapters, CDN, and NextJS socket sync. Non-trigger prompts cover unrelated writing, translation, logo, marketing ROI, and Excel tasks.
