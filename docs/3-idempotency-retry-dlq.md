# 3. Idempotency, Retry, and DLQ

Redis is used only as an ephemeral idempotency gate with TTL. MongoDB unique indexes on `inbound_events.idempotency_key` and `messages.channel_message_key` remain the durable protection.

## Inbound

- Webhook handler verifies and enqueues only.
- Dispatcher checks Redis key `inbound:{idempotency_key}`.
- Duplicate webhook/resync events are acknowledged and skipped.
- Conversation workers upsert raw inbound events, conversations, and messages.

## Outbound

- Outbound queue payloads include `expires_at`.
- Worker acknowledges expired messages without sending.
- Retry schedule is 5 seconds, 30 seconds, then 5 minutes.
- After retry 3, messages are published to `dlq.queue`.
