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
  session_status: string;
  last_webhook_at?: string;
  last_sync_at?: string;
  last_error?: string;
  enabled: boolean;
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
