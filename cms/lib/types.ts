export type Meta = {
  page: number;
  limit: number;
  total: number;
};

export type ListResponse<T> = {
  data: T[];
  meta?: Meta;
};

export type User = {
  id: string;
  email: string;
  display_name: string;
  status: string;
  role_ids: string[];
  team_ids: string[];
  last_login_at?: string;
  created_at: string;
  updated_at: string;
};

export type Role = {
  id: string;
  name: string;
  code: string;
  permission_codes: string[];
  is_system: boolean;
};

export type Permission = {
  id: string;
  code: string;
  resource: string;
  action: string;
  description: string;
};

export type Team = {
  id: string;
  name: string;
  parent_team_id?: string;
  manager_user_ids: string[];
  status: string;
};

export type Channel = {
  id: string;
  code: string;
  name: string;
  kind: string;
  official_api_available: boolean;
  status: string;
};

export type ChannelAccount = {
  id: string;
  channel_id: string;
  name: string;
  owner_team_id?: string;
  shared_team_ids: string[];
  shared_user_ids: string[];
  credential_ref?: string;
  webhook_secret_ref?: string;
  metadata?: {
    accountLabel?: string | null;
    phone?: string | null;
    browserName?: string | null;
    autoConnect?: boolean;
    syncFullHistory?: boolean;
  };
  session_status: string;
  last_webhook_at?: string;
  last_sync_at?: string;
  last_error?: string;
  enabled: boolean;
};

export type Conversation = {
  id: string;
  channel_account_id: string;
  external_conversation_id: string;
  customer_ref?: string;
  customer_name?: string;
  assigned_user_id?: string;
  assigned_team_id?: string;
  status: string;
  last_message_at: string;
  last_message_text?: string;
  unread_count: number;
  has_unread?: boolean;
  last_seen_at?: string;
  tags: string[];
  created_at: string;
  updated_at: string;
  deleted_at?: string;
};

export type ChatNotificationItem = {
  conversation_id: string;
  customer_name: string;
  customer_ref: string;
  channel_account_id: string;
  last_message_text: string;
  last_message_at: string;
  unread_count: number;
};

export type ChatNotificationSummary = {
  total_unread: number;
  missed_count: number;
  latest_at?: string;
  items: ChatNotificationItem[];
};

export type Message = {
  id: string;
  conversation_id: string;
  direction: "inbound" | "outbound" | string;
  sender_type: "customer" | "agent" | string;
  sender_user_id?: string;
  channel_message_id?: string;
  channel_message_key?: string;
  text: string;
  status: string;
  event_time: string;
  sent_at?: string;
  delivered_at?: string;
  read_at?: string;
  created_at: string;
  updated_at: string;
};

export type TypingStatus = {
  accountId?: string;
  jid?: string;
  typing: boolean;
  updatedAt?: string;
  expiresAt?: string;
  adapter_up?: boolean;
};

export type WhatsAppSession = {
  accountId: string;
  status: "disconnected" | "connecting" | "qr" | "connected" | "error";
  qr?: string;
  qrExpiresAt?: string;
  lastSyncAt?: string;
  lastError?: string;
  cached?: boolean;
  qr_cached_at?: string;
  qr_expires_at?: string;
};

export type AuditLog = {
  id: string;
  actor_user_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  metadata?: Record<string, unknown>;
  ip?: string;
  user_agent?: string;
  created_at: string;
};

export type PermissionMatrix = {
  roles: Role[];
  permissions: Permission[];
};

export type Profile = {
  user: User;
  permissions: Record<string, boolean>;
};
