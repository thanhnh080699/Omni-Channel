# Testing and Observability

## Message Delay Metrics

Track:

- `channel_event_delay_ms`
- `gateway_to_queue_delay_ms`
- `queue_wait_ms`
- `worker_process_ms`
- `socket_emit_delay_ms`

Compute from:

```txt
event_time
gateway_received_at
queued_at
processed_at
socket_emitted_at
agent_seen_at
```

## Queue Metrics

Track per queue:

- `ready_count`
- `unacked_count`
- `consumer_count`
- `retry_count`
- `dlq_count`

## Channel Metrics

Track:

- `channel_health`
- `last_webhook_at`
- `last_sync_at`
- `last_error`
- `session_status`

## Audit Events

Track:

- `who_viewed_conversation`
- `who_sent_message`
- `who_reassigned_conversation`
- `who_downloaded_attachment`

## Debug Playbooks

Inbound delay:

1. Check channel webhook/polling timestamp.
2. Check raw `inbound_events` persistence.
3. Check RabbitMQ ready/unacked and retry/DLQ.
4. Check worker logs by trace ID.
5. Check MongoDB message upsert and duplicate indexes.
6. Check socket emit and room authorization.
7. Check frontend reconnect and REST sync.

Outbound failure:

1. Check permission and conversation state.
2. Check pending message and outbound event.
3. Check RabbitMQ outbound queue.
4. Check adapter classification: transient, permanent, auth, rate-limited, blocked.
5. Check retry/DLQ and message status update.

Permission issue:

1. Check role permissions.
2. Check team membership and manager relationship.
3. Check conversation assignment/members.
4. Check attachment conversation ownership.
5. Check audit log.

## Test Coverage

Backend:

- RBAC decision table tests.
- Idempotent inbound event processing.
- Duplicate and out-of-order message handling.
- Retry/DLQ behavior.
- Outbound send status transitions.
- Attachment async processing.

Frontend:

- Conversation list filters.
- Chat message merge/dedup.
- Socket reconnect sync.
- Permission-denied states.
- Attachment status rendering.

Integration:

- Webhook -> queue -> worker -> MongoDB -> socket.
- Agent send -> outbound queue -> adapter -> status update.
- CDN upload/download authorization.

## Eval Prompts

Use `evals/evals.json` for should-trigger and should-not-trigger cases. Good skill output should mention permission checks, idempotency, retry/DLQ, sync recovery, channel-adapter separation, and security where relevant.
