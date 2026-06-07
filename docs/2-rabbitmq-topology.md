# 2. RabbitMQ Topology

The backend declares durable direct exchanges and queues on startup.

```txt
inbound.exchange -> dispatcher.queue
dispatcher.queue -> conv.queue.0..N
outbound.exchange -> outbound.queue
outbound.retry.1 -> outbound.exchange after 5s
outbound.retry.2 -> outbound.exchange after 30s
outbound.retry.3 -> outbound.exchange after 5m
dlq.exchange -> dlq.queue
```

## Ordering

The dispatcher hashes `conversationId` with FNV-1a and routes every event for the same conversation to the same `conv.queue.N`. Each conversation queue is consumed with prefetch `1`; do not increase the consumer count for a conversation partition.

## Worker Commands

Run the API server with:

```bash
go run . serve
```

Run queue workers with:

```bash
go run . worker
```
