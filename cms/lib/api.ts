import type {
  AuditLog,
  Channel,
  ChannelAccount,
  ListResponse,
  PermissionMatrix,
  Profile,
  Role,
  Team,
  User,
} from "@/lib/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:18080";

type RequestOptions = RequestInit & {
  token?: string | null;
};

function buildQuery(params?: Record<string, string | number | boolean | undefined>) {
  const search = new URLSearchParams();
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined && value !== "") {
      search.set(key, String(value));
    }
  });
  const query = search.toString();
  return query ? `?${query}` : "";
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || "Request failed");
  }
  return payload as T;
}

export const api = {
  login: (email: string, password: string) =>
    request<{ access_token: string; expires_at: string; user: User }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  profile: (token: string) => request<Profile>("/api/auth/profile", { token }),
  updateProfile: (token: string, body: { display_name: string; email: string }) =>
    request<{ data: Profile["user"] }>("/api/auth/profile", { method: "PATCH", token, body: JSON.stringify(body) }),
  changePassword: (token: string, body: { current_password: string; new_password: string }) =>
    request<{ status: string }>("/api/auth/change-password", { method: "POST", token, body: JSON.stringify(body) }),
  users: (token: string, params?: Record<string, string | number | boolean | undefined>) =>
    request<ListResponse<User>>(`/api/admin/users${buildQuery(params)}`, { token }),
  createUser: (token: string, body: Partial<User> & { password: string }) =>
    request<{ data: User }>("/api/admin/users", { method: "POST", token, body: JSON.stringify(body) }),
  updateUser: (token: string, id: string, body: Partial<User> & { password?: string }) =>
    request<{ status: string }>(`/api/admin/users/${id}`, { method: "PATCH", token, body: JSON.stringify(body) }),
  deleteUser: (token: string, id: string) =>
    request<{ status: string }>(`/api/admin/users/${id}`, { method: "DELETE", token }),
  roles: (token: string, params?: Record<string, string | number | boolean | undefined>) =>
    request<ListResponse<Role>>(`/api/admin/roles${buildQuery(params)}`, { token }),
  createRole: (token: string, body: Pick<Role, "name" | "code" | "permission_codes">) =>
    request<{ data: Role }>("/api/admin/roles", { method: "POST", token, body: JSON.stringify(body) }),
  updateRole: (token: string, id: string, body: Pick<Role, "name" | "code" | "permission_codes">) =>
    request<{ status: string }>(`/api/admin/roles/${id}`, { method: "PATCH", token, body: JSON.stringify(body) }),
  permissionMatrix: (token: string) => request<PermissionMatrix>("/api/admin/permissions/matrix", { token }),
  teams: (token: string, params?: Record<string, string | number | boolean | undefined>) =>
    request<ListResponse<Team>>(`/api/admin/teams${buildQuery(params)}`, { token }),
  createTeam: (token: string, body: Partial<Team>) =>
    request<{ data: Team }>("/api/admin/teams", { method: "POST", token, body: JSON.stringify(body) }),
  updateTeam: (token: string, id: string, body: Partial<Team>) =>
    request<{ status: string }>(`/api/admin/teams/${id}`, { method: "PATCH", token, body: JSON.stringify(body) }),
  deleteTeam: (token: string, id: string) =>
    request<{ status: string }>(`/api/admin/teams/${id}`, { method: "DELETE", token }),
  channels: (token: string) => request<ListResponse<Channel>>("/api/admin/channels", { token }),
  channelAccounts: (token: string, params?: Record<string, string | number | boolean | undefined>) =>
    request<ListResponse<ChannelAccount>>(`/api/channel-admin/channel-accounts${buildQuery(params)}`, { token }),
  channelAdminChannels: (token: string) => request<ListResponse<Channel>>("/api/channel-admin/channels", { token }),
  channelAdminTeams: (token: string) => request<ListResponse<Team>>("/api/channel-admin/teams", { token }),
  channelAdminUsers: (token: string) => request<ListResponse<User>>("/api/channel-admin/users", { token }),
  createChannelAccount: (token: string, body: Partial<ChannelAccount>) =>
    request<{ data: ChannelAccount }>("/api/channel-admin/channel-accounts", {
      method: "POST",
      token,
      body: JSON.stringify(body),
    }),
  updateChannelAccount: (token: string, id: string, body: Partial<ChannelAccount>) =>
    request<{ status: string }>(`/api/channel-admin/channel-accounts/${id}`, {
      method: "PATCH",
      token,
      body: JSON.stringify(body),
    }),
  enableChannelAccount: (token: string, id: string) =>
    request<{ status: string }>(`/api/channel-admin/channel-accounts/${id}/enable`, { method: "POST", token }),
  disableChannelAccount: (token: string, id: string) =>
    request<{ status: string }>(`/api/channel-admin/channel-accounts/${id}/disable`, { method: "POST", token }),
  channelAccountHealth: (token: string, id: string) =>
    request<Record<string, unknown>>(`/api/channel-admin/channel-accounts/${id}/health`, { token }),
  auditLogs: (token: string, params?: Record<string, string | number | boolean | undefined>) =>
    request<ListResponse<AuditLog>>(`/api/admin/audit-logs${buildQuery(params)}`, { token }),
};
