# 1. WhatsApp Channel

WhatsApp is implemented as a TypeScript adapter inside `api/whatsapp-adapter` using `@whiskeysockets/baileys`. The Go core does not import Baileys directly, but `go run main.go` autostarts and supervises the adapter for local development.

## Responsibilities

- Maintain Baileys session files per channel account.
- Expose QR/session/health endpoints for the admin UI.
- Normalize inbound WhatsApp Web events into the shared `NormalizedInboundMessage` shape.
- Publish inbound envelopes to `inbound.exchange` using routing key `inbound.dispatcher`.
- Accept outbound send requests from the Go outbound worker.
- Run `/resync` every 30 seconds from the Go worker to refresh session state and recover missed events.
- In local development, the API starts the adapter from `WHATSAPP_ADAPTER_DIR` when `WHATSAPP_ADAPTER_AUTOSTART=true`.
- If `WHATSAPP_ADAPTER_AUTO_INSTALL=true`, the API runs `npm install` in `api/whatsapp-adapter` when `node_modules` is missing.

## Risk Note

Baileys is an unofficial WhatsApp Web integration and is not affiliated with WhatsApp. Operators must expect session loss, QR re-login, device verification, protocol changes, rate limits, and possible account restrictions. Use only accounts the system owner is authorized to manage.
