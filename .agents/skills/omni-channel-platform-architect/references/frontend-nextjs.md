# Frontend NextJS

## Routes

```txt
/app/login
/app/dashboard
/app/inbox
/app/inbox/[conversationId]
/app/admin/users
/app/admin/roles
/app/admin/teams
/app/admin/channels
/app/reports
```

## Inbox UI

Required modules:

- Sidebar conversation list.
- Chat window realtime.
- Message composer.
- Attachment preview.
- Staff assignment control.
- Team filter.
- Channel filter.
- Unread badge.
- Message status.
- Reconnect and sync handling.

## State Model

Keep normalized client state:

- `conversationsById`
- `conversationIdsByFilter`
- `messagesByConversationId`
- `attachmentsByMessageId`
- `presenceByUserId`
- `lastSyncCursorByConversationId`

## Socket Behavior

- Subscribe only to authorized conversation/team rooms.
- On socket reconnect, call REST sync for current conversation and visible conversation list.
- Never assume socket events are complete or ordered.
- Deduplicate messages by `message_id` or `channel_message_key`.
- Show attachment status updates separately from message creation.

## Conversation Open Sync

When opening `/app/inbox/[conversationId]`:

1. Fetch conversation detail with permission check.
2. Fetch latest messages by cursor.
3. Join socket room after authorization.
4. Mark visible messages as read.
5. If socket emits older/newer items, merge deterministically.

## Admin UI

Admin pages should support:

- User CRUD.
- Role CRUD and permission matrix.
- Team membership and manager assignment.
- Channel account create/update/enable/disable.
- Channel health display.
- Routing rule configuration if implemented.

## UX Failure States

- Socket disconnected: show subtle reconnect state and keep REST actions available.
- Message pending/sending/failed: show status and retry for permitted users.
- Attachment pending/processing/failed: show placeholder and retry if allowed.
- Permission denied: do not leak hidden conversation details.
- Channel connection status indicator mapping:
  - Green (Xanh lá): `connected`
  - Yellow (Vàng): `connecting` / `waiting_qr` / `waiting_config`
  - Red (Đỏ): `error` (e.g., API authorization failure, rate-limit block, expired credentials)
  - Gray (Xám): `not_connected` / `disconnected` (e.g., setup not yet initiated, or manually logged out)
  - Do not merge Error (Red) and Not Connected (Gray) into a single status to ensure clear and actionable feedback.

