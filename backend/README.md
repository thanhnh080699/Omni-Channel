# Omni Channel Backend API

Golang API for the Omni Channel Chat Platform. This first backend slice includes MongoDB connection, collection indexes, RBAC seed data, auth, admin CRUD foundations, channel account management, conversation/message endpoints, audit logs, and raw inbound webhook storage.

## Run Locally

MongoDB local credentials currently expected:

```txt
host: localhost:27017
username: root
password: root
authSource: admin
database: omni_channel
```

Start the API:

```bash
cd backend
MONGO_URI='mongodb://root:root@localhost:27017/omni_channel?authSource=admin' \
MONGO_DATABASE='omni_channel' \
JWT_SECRET='local-dev-secret' \
ADMIN_EMAIL='admin@example.com' \
ADMIN_PASSWORD='admin123456' \
API_PORT=18080 \
go run ./cmd/api
```

If port `8080` is free, set `API_PORT=8080`.

## Seeded Login

The server creates default permissions, roles, channel registry, and one admin user when missing.

```bash
curl -X POST http://localhost:18080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123456"}'
```

## Implemented Endpoints

- `GET /health`
- `POST /api/auth/login`
- `GET /api/auth/profile`
- `POST /api/auth/logout`
- `POST /api/auth/refresh`
- `GET /api/admin/users`
- `POST /api/admin/users`
- `PATCH /api/admin/users/{userId}`
- `DELETE /api/admin/users/{userId}`
- `GET /api/admin/roles`
- `POST /api/admin/roles`
- `PATCH /api/admin/roles/{roleId}`
- `POST /api/admin/users/{userId}/roles`
- `POST /api/admin/users/{userId}/teams`
- `GET /api/admin/permissions/matrix`
- `GET /api/admin/teams`
- `POST /api/admin/teams`
- `GET /api/admin/channels`
- `POST /api/admin/channel-accounts`
- `PATCH /api/admin/channel-accounts/{accountId}`
- `POST /api/admin/channel-accounts/{accountId}/enable`
- `POST /api/admin/channel-accounts/{accountId}/disable`
- `GET /api/admin/channel-accounts/{accountId}/health`
- `GET /api/conversations/my`
- `GET /api/conversations/team`
- `GET /api/conversations/{conversationId}`
- `POST /api/conversations/{conversationId}/assign`
- `POST /api/conversations/{conversationId}/transfer`
- `POST /api/conversations/{conversationId}/close`
- `POST /api/conversations/{conversationId}/reopen`
- `GET /api/conversations/{conversationId}/messages`
- `POST /api/conversations/{conversationId}/messages`
- `POST /api/conversations/{conversationId}/read`
- `POST /api/messages/{messageId}/retry`
- `POST /webhooks/{channel}/{accountId}`

## Next Backend Slices

- RabbitMQ publish/consume for inbound/outbound/media jobs.
- Conversation creation from normalized inbound workers.
- Socket realtime events.
- Attachment upload and CDN authorization.
- Integration tests for RBAC and idempotent webhook storage.
