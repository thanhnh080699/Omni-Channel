# Database Design

Use MongoDB as the primary store. Prefer ObjectId or UUID consistently, store `created_at` and `updated_at`, and add `deleted_at` for soft-delete where needed.

## Collections

### users

Purpose: staff, managers, admins, supervisors, auditors.

Fields: `_id`, `email`, `password_hash`, `display_name`, `status`, `role_ids`, `team_ids`, `last_login_at`.

Indexes: unique `email`, index `status`, index `team_ids`.

Example:

```json
{"_id":"usr_1","email":"staff@example.com","display_name":"Staff 1","status":"active","role_ids":["role_staff"],"team_ids":["team_sales"],"created_at":"2026-06-06T00:00:00Z"}
```

### roles

Purpose: group permissions.

Fields: `_id`, `name`, `code`, `permission_codes`, `is_system`.

Indexes: unique `code`.

Example:

```json
{"_id":"role_staff","name":"Staff","code":"staff","permission_codes":["conversation:view_assigned","message:send_assigned"]}
```

### permissions

Purpose: atomic capabilities.

Fields: `_id`, `code`, `resource`, `action`, `description`.

Indexes: unique `code`, compound `resource + action`.

Example:

```json
{"_id":"perm_1","code":"conversation:view_team","resource":"conversation","action":"view_team"}
```

### teams

Purpose: team hierarchy and manager ownership.

Fields: `_id`, `name`, `parent_team_id`, `manager_user_ids`, `status`.

Indexes: index `parent_team_id`, index `manager_user_ids`.

Example:

```json
{"_id":"team_sales","name":"Sales","manager_user_ids":["usr_manager"],"status":"active"}
```

### team_members

Purpose: explicit team membership and reporting relationship.

Fields: `_id`, `team_id`, `user_id`, `member_role`, `manager_user_id`, `joined_at`.

Indexes: unique `team_id + user_id`, index `user_id`, index `manager_user_id`.

Example:

```json
{"_id":"tm_1","team_id":"team_sales","user_id":"usr_1","member_role":"staff","manager_user_id":"usr_manager"}
```

### channels

Purpose: channel type registry.

Fields: `_id`, `code`, `name`, `kind`, `official_api_available`, `status`.

Indexes: unique `code`.

Example:

```json
{"_id":"ch_zalo","code":"zalo_personal","name":"Zalo Personal","kind":"zalo","official_api_available":false,"status":"enabled"}
```

### channel_accounts

Purpose: configured accounts/pages/bots.

Fields: `_id`, `channel_id`, `name`, `owner_team_id`, `credential_ref`, `webhook_secret_ref`, `session_status` (enum: `connected`, `connecting`, `waiting_qr`, `waiting_config`, `error`, `not_connected`, `disconnected`), `last_webhook_at`, `last_sync_at`, `last_error`, `enabled`.

Indexes: index `channel_id`, index `owner_team_id`, index `enabled + session_status`.

Example:

```json
{"_id":"ca_1","channel_id":"ch_zalo","name":"Zalo Sales","owner_team_id":"team_sales","session_status":"connected","enabled":true}
```

### conversations

Purpose: normalized conversation thread.

Fields: `_id`, `channel_account_id`, `external_conversation_id`, `customer_ref`, `assigned_user_id`, `assigned_team_id`, `status`, `last_message_at`, `unread_count`, `tags`.

Indexes: unique `channel_account_id + external_conversation_id`, index `assigned_user_id + status`, index `assigned_team_id + status`, index `last_message_at`.

Example:

```json
{"_id":"conv_1","channel_account_id":"ca_1","external_conversation_id":"zalo_thread_123","assigned_user_id":"usr_1","assigned_team_id":"team_sales","status":"open","last_message_at":"2026-06-06T01:00:00Z"}
```

### conversation_members

Purpose: explicit access participants.

Fields: `_id`, `conversation_id`, `user_id`, `access_level`, `source`, `last_seen_message_id`.

Indexes: unique `conversation_id + user_id`, index `user_id`.

Example:

```json
{"_id":"cm_1","conversation_id":"conv_1","user_id":"usr_1","access_level":"assigned","source":"assignment"}
```

### messages

Purpose: normalized messages.

Fields: `_id`, `conversation_id`, `direction`, `sender_type`, `sender_user_id`, `channel_message_id`, `channel_message_key`, `text`, `status`, `event_time`, `sent_at`, `delivered_at`, `read_at`.

Indexes: unique sparse `channel_message_key`, index `conversation_id + event_time`, index `conversation_id + _id`, index `status`.

Example:

```json
{"_id":"msg_1","conversation_id":"conv_1","direction":"inbound","sender_type":"customer","channel_message_key":"ca_1:evt_1","text":"Hello","status":"delivered","event_time":"2026-06-06T01:00:00Z"}
```

### message_attachments

Purpose: media/file metadata.

Fields: `_id`, `message_id`, `conversation_id`, `type`, `original_url`, `cdn_path`, `cdn_url`, `status`, `size_bytes`, `mime_type`, `error`.

Indexes: index `message_id`, index `conversation_id + status`, index `status`.

Example:

```json
{"_id":"att_1","message_id":"msg_1","conversation_id":"conv_1","type":"image","cdn_path":"/cdn/zalo/conv_1/2026/06/06/msg_1/photo.jpg","status":"ready"}
```

### inbound_events

Purpose: raw inbound event storage, replay, idempotency.

Fields: `_id`, `channel_account_id`, `event_id`, `idempotency_key`, `raw_payload`, `status`, timestamps, `attempt_count`, `error`.

Indexes: unique `idempotency_key`, index `status + queued_at`, index `channel_account_id + event_time`.

Example:

```json
{"_id":"in_1","channel_account_id":"ca_1","event_id":"evt_1","idempotency_key":"ca_1:evt_1","status":"processed","event_time":"2026-06-06T01:00:00Z"}
```

### outbound_events

Purpose: outbound send audit and retry state.

Fields: `_id`, `message_id`, `channel_account_id`, `idempotency_key`, `status`, `attempt_count`, `last_error`, timestamps.

Indexes: unique `idempotency_key`, index `status + created_at`, index `message_id`.

Example:

```json
{"_id":"out_1","message_id":"msg_2","channel_account_id":"ca_1","idempotency_key":"msg_2:send","status":"sent","attempt_count":1}
```

### queue_jobs

Purpose: optional app-level queue visibility.

Fields: `_id`, `queue_name`, `job_type`, `ref_id`, `status`, `attempt_count`, `next_run_at`, `last_error`.

Indexes: index `queue_name + status + next_run_at`, index `ref_id`.

Example:

```json
{"_id":"job_1","queue_name":"media.jobs","job_type":"download_attachment","ref_id":"att_1","status":"processing","attempt_count":1}
```

### sync_checkpoints

Purpose: recovery cursor per channel/account/conversation.

Fields: `_id`, `channel_account_id`, `conversation_id`, `cursor`, `last_synced_at`, `status`, `last_error`.

Indexes: unique `channel_account_id + conversation_id`, index `last_synced_at`.

Example:

```json
{"_id":"sync_1","channel_account_id":"ca_1","conversation_id":"conv_1","cursor":"abc123","last_synced_at":"2026-06-06T01:05:00Z"}
```

### audit_logs

Purpose: security and operational audit.

Fields: `_id`, `actor_user_id`, `action`, `resource_type`, `resource_id`, `metadata`, `ip`, `user_agent`, `created_at`.

Indexes: index `actor_user_id + created_at`, index `resource_type + resource_id`, index `action + created_at`.

Example:

```json
{"_id":"aud_1","actor_user_id":"usr_1","action":"message.send","resource_type":"conversation","resource_id":"conv_1","metadata":{"message_id":"msg_2"}}
```
