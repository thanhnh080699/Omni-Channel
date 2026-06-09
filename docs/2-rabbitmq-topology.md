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

## Runtime

Run the API server with one command:

```bash
go run . serve
```

The API process starts the HTTP server, channel adapter supervision, dispatcher, conversation workers, outbound worker, and resync loop together. This is the default and recommended runtime so inbound queues always have consumers.

For advanced debugging only, workers can still be run without the HTTP API:

```bash
go run . worker
```
