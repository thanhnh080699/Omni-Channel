# API Design

For every API, specify method, path, request body, response body, permission check, error codes, logging, and audit behavior.

## Auth

- `POST /api/auth/login`: login with email/password. Audit failed/successful attempts without logging password.
- `POST /api/auth/logout`: revoke session/refresh token.
- `POST /api/auth/refresh`: rotate access token.
- `GET /api/auth/profile`: return current user, roles, permissions, teams.

## User/Admin

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

Permission: admin-only unless explicitly delegated.

## Channel

- `POST /api/admin/channel-accounts`
- `PATCH /api/admin/channel-accounts/{accountId}`
- `POST /api/admin/channel-accounts/{accountId}/enable`
- `POST /api/admin/channel-accounts/{accountId}/disable`
- `GET /api/admin/channel-accounts/{accountId}/health`

Never return raw credentials.

## Conversation

- `GET /api/conversations/my`: staff assigned/member conversations.
- `GET /api/conversations/team`: manager team conversations.
- `GET /api/conversations/{conversationId}`: detail with permission check.
- `POST /api/conversations/{conversationId}/assign`
- `POST /api/conversations/{conversationId}/transfer`
- `POST /api/conversations/{conversationId}/close`
- `POST /api/conversations/{conversationId}/reopen`

Audit assignment, transfer, close, reopen, and sensitive views.

## Message

- `GET /api/conversations/{conversationId}/messages?after=&before=&limit=`
- `POST /api/conversations/{conversationId}/messages`
- `POST /api/messages/{messageId}/retry`
- `POST /api/conversations/{conversationId}/read`
- `POST /api/conversations/{conversationId}/attachments`

Send flow:

```txt
Frontend -> API permission check -> save pending outbound message -> publish outbound queue -> adapter sends -> update status -> socket update
```

## Webhook

- `POST /webhooks/{channel}/{accountId}`.

Requirements:

- Validate signature/session when supported.
- Save raw event before processing.
- Return quickly after durable save and queue publish.
- Do not log secrets or full sensitive payloads.

## Monitoring

- `GET /api/monitoring/queues`
- `GET /api/monitoring/channels`
- `GET /api/monitoring/workers`
- `GET /api/monitoring/delays`

Restrict to admin/ops roles.

## Common Error Codes

- `400`: invalid input.
- `401`: unauthenticated.
- `403`: permission denied.
- `404`: resource not found or hidden.
- `409`: duplicate/idempotency conflict or invalid state.
- `422`: channel cannot process request.
- `429`: rate limited.
- `500`: unexpected error.
- `503`: channel, queue, or worker unavailable.

## WebSocket Events

- `conversation.created`
- `conversation.updated`
- `message.created`
- `message.status_updated`
- `attachment.updated`
- `conversation.assigned`
- `presence.updated`
- `sync.required`

Use REST sync after reconnect.
