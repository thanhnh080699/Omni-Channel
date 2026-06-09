# 1. WhatsApp Channel

WhatsApp is implemented as a TypeScript adapter inside `api/adapter/whatsapp` using `@whiskeysockets/baileys`. The Go core does not import Baileys directly, but `go run main.go` autostarts and supervises the adapter for local development.

## Responsibilities

- Maintain Baileys session files per channel account.
- Expose QR/session/health endpoints for the admin UI.
- Normalize inbound WhatsApp Web events into the shared `NormalizedInboundMessage` shape.
- Publish inbound envelopes to `inbound.exchange` using routing key `inbound.dispatcher`.
- Accept outbound send requests from the Go outbound worker.
- Run `/resync` every 30 seconds from the Go worker to refresh session state and recover missed events.
- In local development, the API starts the adapter from `WHATSAPP_ADAPTER_DIR` when `WHATSAPP_ADAPTER_AUTOSTART=true`.
- If `WHATSAPP_ADAPTER_AUTO_INSTALL=true`, the API runs `npm install` in `api/adapter/whatsapp` when `node_modules` is missing.
- WhatsApp adapter configuration lives in the shared API env file, `api/.env`; do not create a separate adapter `.env.example`.
- `POST /connect/{accountId}` waits briefly for the first QR snapshot before returning. Each QR carries `qrExpiresAt`; the API cache follows that expiry and drops stale QR values so the CMS never renders a QR that WhatsApp may reject as expired.
- When the API starts, enabled WhatsApp channel accounts auto-connect by default unless metadata explicitly sets `autoConnect=false`. Full-history sync metadata also defaults to `syncFullHistory=true`.
- The adapter keeps Baileys auth files under the configured session directory, uses keep-alive socket settings, and reconnects non-logout disconnects with bounded exponential backoff. Manual disconnect and reset session stop reconnect until a user or API startup calls connect again.

## Risk Note

Baileys is an unofficial WhatsApp Web integration and is not affiliated with WhatsApp. Operators must expect session loss, QR re-login, device verification, protocol changes, rate limits, and possible account restrictions. Use only accounts the system owner is authorized to manage.
