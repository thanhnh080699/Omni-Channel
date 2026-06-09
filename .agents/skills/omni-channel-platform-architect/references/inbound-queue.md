# Inbound Queue

## Flow

```txt
Channel Webhook / Polling
-> Channel Gateway
-> Validate signature/session
-> Save raw event into inbound_events
-> Publish RabbitMQ
-> Inbound Worker
-> Normalize message
-> Upsert conversation
-> Upsert message
-> Emit socket event
-> Media worker upload CDN if attachments exist
```

## Required Timestamps

Store and surface these fields:

- `event_time`: timestamp from the channel.
- `gateway_received_at`: when platform received webhook/polling event.
- `queued_at`: when event was published to RabbitMQ.
- `processed_at`: when worker completed message persistence.
- `socket_emitted_at`: when realtime event was emitted.
- `agent_seen_at`: when frontend/user acknowledged visibility.

## Idempotency

- Create an idempotency key from `channel_account_id + channel_event_id` when available.
- Fallback to a stable hash of channel, sender, receiver, message timestamp, content fingerprint, and attachment IDs.
- Use unique indexes on `inbound_events.idempotency_key` and `messages.channel_message_key`.
- Make worker operations upsert-based so retry is safe.

## Failure Cases

- Webhook delay 1-2 minutes: sort messages by channel event time, show delayed marker when useful, and rely on conversation-open sync.
- Webhook miss: run periodic sync per channel account using `sync_checkpoints`.
- Duplicate event: store raw duplicate as duplicate/skipped or increment duplicate count, but do not create duplicate messages.
- Out-of-order messages: persist by event time and sequence when channel provides it; use deterministic tie-breaker.
- Queue backlog: monitor ready/unacked counts, worker concurrency, and processing latency.
- Worker fails mid-job: use idempotent checkpoints, ack only after durable writes, and retry transient errors.
- Media slow: create message with attachment status `pending` or `processing`; media worker updates later.
- Socket disconnect: frontend reconnect must call REST sync for messages after last seen cursor.
- Agent opens chat and new message is missing: call conversation sync and compare latest message cursor.

## RabbitMQ Design

Use separate exchanges/queues:

- `inbound.events`: initial event processing.
- `inbound.retry`: delayed retries with attempt count.
- `inbound.dlq`: poison events after max attempts.
- `media.jobs`: attachment downloads/uploads.

Message metadata:

- `event_id`
- `idempotency_key`
- `channel`
- `channel_account_id`
- `attempt`
- `trace_id`
- `queued_at`

## Sync Recovery

- Periodic sync per enabled channel account.
- Conversation-open sync when agent enters a conversation.
- Manual admin sync for channel health/debug.
- Store `sync_checkpoints` per channel account and conversation when the channel supports cursors.

## Debug Checklist

Start at the first missing timestamp:

1. Channel event exists but `gateway_received_at` absent: webhook/polling issue.
2. Gateway saved event but no `queued_at`: publish or RabbitMQ issue.
3. Queued but no `processed_at`: worker, backlog, duplicate key, or DLQ.
4. Processed but no `socket_emitted_at`: socket service or event bus issue.
5. Socket emitted but no `agent_seen_at`: frontend connection, room subscription, or sync issue.
