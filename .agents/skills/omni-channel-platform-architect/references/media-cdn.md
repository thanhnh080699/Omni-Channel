# Media CDN

The repository already includes a Go CDN service in `cdn/`. Integrate with it through API/env/volume boundaries instead of rewriting it.

## Media Flow

```txt
Inbound message contains media
-> Save message immediately
-> Create message_attachments with pending status
-> Publish media job
-> Media worker downloads from channel
-> Uploads to CDN local folder or CDN API
-> Updates attachment status and CDN URL
-> Emits socket attachment update
```

Webhook handling must not wait for media download/upload.

## Path Convention

Use:

```txt
/cdn/{channel}/{conversation_id}/{yyyy}/{mm}/{dd}/{message_id}/{filename}
```

Example:

```txt
/cdn/zalo/conv_1/2026/06/06/msg_1/photo.jpg
```

## Attachment Status

- `pending`: metadata created.
- `processing`: worker is downloading/uploading.
- `ready`: CDN URL/path is available.
- `failed`: retry exhausted or permanent failure.
- `expired`: source media no longer available.

## CDN Security

- Do not expose raw private files directly when the conversation requires access control.
- Use signed URLs or proxy downloads through an authorized API.
- Do not log channel media tokens, session cookies, CDN API keys, or signed URL secrets.
- Validate file type and size before serving or making it visible.

## Media Worker Requirements

- Use context timeout.
- Retry transient channel/CDN errors.
- Store `original_url`, `source_expires_at`, `cdn_path`, `cdn_url`, `mime_type`, `size_bytes`, and `error`.
- Emit socket update after status changes.
- Allow manual retry of failed attachments when source is still available.

## Existing CDN Notes

The existing `cdn` service supports API key auth, local file storage, folder management, image processing, signed URLs, rate limiting, and upload endpoints. Keep its real secrets out of `.env.example`; use placeholders such as `CHANGE_ME_CDN_API_KEY`.
