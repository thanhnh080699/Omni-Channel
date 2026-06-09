# 6. CMS Chat Screen

The CMS now includes a first shared-inbox chat screen at `/chat`.

## UI

- Left column: conversation search, My/Team scope, status filter, channel account filter, unread badge, and synced conversation list.
- Center column: selected conversation header, channel/account context, message history, composer, and close/reopen action.
- Right column: customer detail tabs for information, tags, and loaded conversation history.

## Sync Behavior

- The first version uses REST polling every few seconds instead of a socket client because the repository does not yet include a socket service.
- The visible list is refreshed through `GET /api/conversations/my` or `GET /api/conversations/team`.
- Opening a conversation fetches messages with `GET /api/conversations/{conversationId}/messages` and marks the latest loaded message as read.
- New inbound conversations remain in the chat screen because the inbound worker continues to upsert by `channel_account_id + external_conversation_id`, and the CMS list poll picks up newly created rows.
- The API process always runs queue workers together with HTTP serving, so RabbitMQ queues have consumers during normal `go run . serve` operation.
- Outbound workers persist send results back to MongoDB by setting `messages.status=sent`, `messages.sent_at`, and `outbound_events.status=sent` after the channel adapter accepts a send request.
- The inbox displays `customer_name` from channel metadata when available, then falls back to a cleaned phone/JID from `customer_ref` or `external_conversation_id`.
- The active conversation polls `GET /api/conversations/{conversationId}/typing`; the WhatsApp adapter subscribes to the contact JID with Baileys `presenceSubscribe`, derives typing from `presence.update`, and expires typing snapshots after a short TTL.
- Conversation row action menus render through a body-level portal so the dropdown can extend outside the scrollable conversation list without being clipped.

## Channel Context

- Chat users read channel metadata through `GET /api/chat/channels` and `GET /api/chat/channel-accounts`.
- The CMS maps `conversation.channel_account_id` to the channel account and channel name so agents can see where each customer arrived from.

## Tags

- Tags are stored on the existing `conversations.tags` field.
- `PATCH /api/conversations/{conversationId}/tags` updates tags after conversation visibility and edit permission checks.
- The API trims, lowercases, deduplicates, caps tag length, and caps the number of tags before saving.
- Tag changes are audited as `conversation.tags.update`.
