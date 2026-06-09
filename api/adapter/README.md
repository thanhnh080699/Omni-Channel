# API Channel Adapters

Channel adapters live under `api/adapter/<channel>`.

The Go API owns common platform behavior: auth, RBAC, channel account records, inbound queue publication, workers, audit, and conversation storage. Each adapter owns only the channel-specific transport/session logic needed to connect one external platform to the shared contracts.

## Current Layout

```txt
api/adapter/
  whatsapp/  TypeScript Baileys adapter for WhatsApp Web sessions.
```

## Environment Policy

The API uses one environment file: `api/.env`, copied from `api/.env.example`.

Adapter-specific settings such as `WHATSAPP_ADAPTER_URL`, `WHATSAPP_ADAPTER_DIR`, `WHATSAPP_ADAPTER_PORT`, and `WHATSAPP_SESSION_DIR` must be added to the shared API env example. Do not create `.env.example` files inside adapter folders.

When the API autostarts an adapter, `internal/adapterprocess` passes the required env values into that child process. When an adapter is run manually, run it with values from the same `api/.env` file or export those variables in the shell.

## Adding Another Adapter

1. Create `api/adapter/<channel>`.
2. Keep channel-specific SDK/session/webhook code inside that folder.
3. Add shared config fields under `api/internal/config` only when the API must supervise or call the adapter.
4. Add adapter env placeholders to `api/.env.example`.
5. Keep normalized message contracts aligned with `api/internal/channel` and queue payloads in `api/internal/queue`.
