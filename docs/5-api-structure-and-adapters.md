# 5. API Structure and Adapter Layout

The API keeps core platform code and channel-specific adapter code separated.

## Directory Ownership

- `api/internal/config`: loads the single API environment file and exposes typed runtime config.
- `api/internal/handlers`: HTTP route registration and request handling.
- `api/internal/workers`: queue consumers and background processors.
- `api/internal/queue`: RabbitMQ exchanges, queues, routing keys, and message payloads.
- `api/internal/channel`: shared channel contracts that adapters and workers normalize toward.
- `api/internal/adapterprocess`: local process supervision for adapters.
- `api/adapter/<channel>`: channel-specific SDK, session, transport, and normalization code.

## Adapter Rules

- Put every channel adapter under `api/adapter/<channel>`; do not place adapters at the API root.
- Keep adapter code out of core handlers and workers except through shared contracts and configured HTTP/process boundaries.
- Keep one environment surface for the API: `api/.env` with placeholders documented in `api/.env.example`.
- Do not add `.env.example` files inside adapter folders. Adapter-specific env values belong in the shared API env example.
- For local development, the API may autostart adapters through `internal/adapterprocess`; production can run adapters as separate processes using the same env keys.

This keeps room for future adapters such as Telegram, Facebook, and Zalo without mixing platform logic into the core API.
