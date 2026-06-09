# Channel Adapters

## Adapter Pattern

Keep channel logic outside core chat. Core services speak to a normalized adapter interface:

```go
type ChannelAdapter interface {
    ChannelCode() string
    ValidateWebhook(ctx context.Context, account ChannelAccount, headers map[string]string, body []byte) error
    NormalizeInbound(ctx context.Context, account ChannelAccount, raw []byte) ([]NormalizedMessage, error)
    SendMessage(ctx context.Context, account ChannelAccount, req OutboundMessage) (SendResult, error)
    SyncConversation(ctx context.Context, account ChannelAccount, cursor SyncCursor) (SyncResult, error)
    Health(ctx context.Context, account ChannelAccount) (ChannelHealth, error)
}
```

Each channel must have a separate implementation and configuration.

## Normalized Message

Required fields:

- `channel`
- `channel_account_id`
- `external_conversation_id`
- `external_message_id`
- `sender_external_id`
- `sender_display_name`
- `direction`
- `text`
- `attachments`
- `event_time`
- `raw_event_id`

## Channel Scope

Target channels:

- WhatsApp
- Telegram
- Facebook personal
- Facebook Page
- Zalo personal

References:

- Baileys: https://github.com/WhiskeySockets/Baileys
- zca-js: https://github.com/RFS-ADRENO/zca-js
- Zalo docs: https://developers.zalo.me/docs

## Official API First

Prefer official APIs for reliability, compliance, webhook signatures, rate-limit clarity, and long-term maintenance.

## Unofficial Connector Risk

When proposing Facebook personal, Zalo personal, WhatsApp Web, or any unofficial/personal-account connector, clearly state:

- ToS/account policy risk.
- Session loss and re-login risk.
- Checkpoint, captcha, device verification, and account block risk.
- Breaking changes due to private protocol updates.
- Rate limits and anti-automation controls.
- Operational need for health checks and manual reconnect.

Do not design spam, scraping, bypass, credential theft, or unauthorized account access. Only process messages for accounts/channels the system owner has valid rights to manage.

## Outbound Safety

- Rate-limit per channel and account.
- Use idempotency key per outbound message.
- Classify errors as transient, permanent, auth/session, rate-limited, blocked, or unknown.
- Emit status updates: `pending`, `sending`, `sent`, `delivered`, `read`, `failed`, `cancelled`.
- Keep channel response payloads in `outbound_events` for audit/debug, excluding secrets.
