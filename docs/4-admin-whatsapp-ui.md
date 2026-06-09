# 4. Admin WhatsApp UI

The channel admin page is now a fixed Omni Channel settings flow for WhatsApp-first setup. It no longer exposes the generic "Add channel" modal.

## Controls

- Top channel tiles show default channel status such as connected, disconnected, QR, connecting, or session state.
- Entering the page does not open any channel detail by default; users must click a channel tile to configure it.
- WhatsApp settings omit credential and webhook secret fields while the first slice uses Baileys.
- Owner team is assigned by the backend from the current user's first team when omitted by the CMS.
- New WhatsApp accounts default to enabled, auto-connect on API startup, and full-history sync enabled.
- QR is displayed in the setup panel and read through the API proxy, which caches QR values per user/account to avoid repeatedly fetching from the adapter.
- In development the Go API autostarts `api/adapter/whatsapp`; if the adapter is still unavailable, the session endpoint returns an error-state session instead of breaking the page with a 502.
- Connect, disconnect, reset session, and resync actions are exposed for the WhatsApp account.
- After clicking connect, the CMS keeps polling the session endpoint until it receives QR, connected, disconnected, or error status. This prevents the setup panel from staying permanently at "Đang chờ QR..." when the adapter needs a few seconds to emit the QR.
- The setup panel hides expired QR values and waits for the next QR snapshot. If the phone reports that it cannot link the device, the most likely app-side cause is a stale QR response; cache expiry must stay aligned with `qrExpiresAt`.

The UI still uses the existing channel account permission model, so only admins or users with `channel:manage` can operate these controls.
