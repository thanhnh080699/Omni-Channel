"use client";

import { Cable, CheckCircle2, HeartPulse, Loader2, Plus, Power, Save, Share2 } from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { DataTable } from "@/components/data-table";
import { FormField } from "@/components/form-field";
import { Modal } from "@/components/modal";
import { PageHeader } from "@/components/page-header";
import { StatusPill } from "@/components/status-pill";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Channel, ChannelAccount, Team, User } from "@/lib/types";

const preferredChannelCodes = ["facebook_page", "zalo_oa", "whatsapp", "telegram"];

const emptyAccount = {
  channel_id: "",
  name: "",
  owner_team_id: "",
  shared_team_ids: [] as string[],
  shared_user_ids: [] as string[],
  credential_ref: "",
  webhook_secret_ref: "",
  enabled: true,
};

export default function ChannelsPage() {
  const { token, profile } = useAuth();
  const [channels, setChannels] = useState<Channel[]>([]);
  const [accounts, setAccounts] = useState<ChannelAccount[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [health, setHealth] = useState<Record<string, string>>({});
  const [editing, setEditing] = useState<ChannelAccount | null>(null);
  const [form, setForm] = useState(emptyAccount);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const canManageChannels = Boolean(profile?.permissions["admin:manage"] || profile?.permissions["channel:manage"]);
  const availableChannels = useMemo(() => {
    const preferred = channels.filter((channel) => preferredChannelCodes.includes(channel.code));
    return preferred.length > 0 ? preferred : channels;
  }, [channels]);

  async function load() {
    if (!token) return;
    setLoading(true);
    const [channelResult, accountResult, teamResult, userResult] = await Promise.all([
      api.channelAdminChannels(token),
      api.channelAccounts(token),
      api.channelAdminTeams(token),
      api.channelAdminUsers(token),
    ]);
    setChannels(channelResult.data || []);
    setAccounts(accountResult.data || []);
    setTeams(teamResult.data || []);
    setUsers(userResult.data || []);
    setLoading(false);
  }

  useEffect(() => {
    void load();
  }, [token]);

  function channelLabel(id: string) {
    return channels.find((channel) => channel.id === id)?.name || id;
  }

  function edit(account?: ChannelAccount) {
    setError("");
    setEditing(account || null);
    setForm(
      account
        ? {
            channel_id: account.channel_id,
            name: account.name,
            owner_team_id: account.owner_team_id || "",
            shared_team_ids: account.shared_team_ids || [],
            shared_user_ids: account.shared_user_ids || [],
            credential_ref: account.credential_ref || "",
            webhook_secret_ref: account.webhook_secret_ref || "",
            enabled: account.enabled,
          }
        : { ...emptyAccount, channel_id: availableChannels[0]?.id || "" },
    );
    setOpen(true);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    try {
      const payload = {
        ...form,
        shared_team_ids: form.shared_team_ids || [],
        shared_user_ids: form.shared_user_ids || [],
      };
      if (editing) {
        await api.updateChannelAccount(token, editing.id, payload);
      } else {
        await api.createChannelAccount(token, payload);
      }
      setOpen(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save channel account");
    }
  }

  async function setEnabled(account: ChannelAccount, enabled: boolean) {
    if (!token) return;
    if (enabled) {
      await api.enableChannelAccount(token, account.id);
    } else {
      await api.disableChannelAccount(token, account.id);
    }
    await load();
  }

  async function checkHealth(account: ChannelAccount) {
    if (!token) return;
    const result = await api.channelAccountHealth(token, account.id);
    setHealth((current) => ({ ...current, [account.id]: String(result.session_status || result.channel_health || "unknown") }));
  }

  return (
    <AdminShell>
      <PageHeader
        title="Channel accounts"
        description="Configure system-wide channel accounts and share message visibility with teams or selected users."
        actions={
          canManageChannels ? (
            <button className="btn btn-primary" onClick={() => edit()}>
              <Plus size={16} /> Add channel
            </button>
          ) : null
        }
      />

      <section className="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {availableChannels.map((channel) => (
          <button
            key={channel.id}
            className="panel group p-4 text-left hover:-translate-y-0.5 hover:shadow-md"
            onClick={() => {
              setForm({ ...emptyAccount, channel_id: channel.id, name: channel.name });
              setEditing(null);
              setOpen(true);
            }}
          >
            <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-md bg-[var(--app-accent-soft-bg)] text-[var(--app-accent-soft-fg)]">
              <Cable size={20} />
            </div>
            <div className="font-semibold">{channel.name}</div>
            <div className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
              {channel.official_api_available ? "Official API" : "Unofficial connector"} · {channel.kind}
            </div>
          </button>
        ))}
      </section>

      <section className="panel mb-5 p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="font-semibold">Operational status</h2>
            <p className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
              Shared accounts make messages visible to selected teams/users after backend conversation permission checks.
            </p>
          </div>
          <div className="flex gap-2 text-sm">
            <div className="rounded-md border px-3 py-2" style={{ borderColor: "var(--app-border)" }}>
              {accounts.length} accounts
            </div>
            <div className="rounded-md border px-3 py-2" style={{ borderColor: "var(--app-border)" }}>
              {accounts.filter((account) => account.enabled).length} enabled
            </div>
          </div>
        </div>
      </section>

      {loading ? (
        <div className="panel flex min-h-64 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin" style={{ color: "var(--app-muted)" }} />
        </div>
      ) : (
        <DataTable
          data={accounts}
          empty="No channel accounts found."
          columns={[
            {
              key: "name",
              label: "Account",
              render: (row) => (
                <div>
                  <div className="font-medium">{row.name}</div>
                  <div className="text-xs" style={{ color: "var(--app-muted)" }}>{channelLabel(row.channel_id)}</div>
                </div>
              ),
            },
            { key: "enabled", label: "Enabled", render: (row) => <StatusPill value={row.enabled} /> },
            { key: "session", label: "Session", render: (row) => <StatusPill value={health[row.id] || row.session_status} /> },
            {
              key: "sharing",
              label: "Shared access",
              render: (row) => (
                <div className="flex flex-wrap gap-1 text-xs">
                  {(row.shared_team_ids || []).map((id) => (
                    <span key={id} className="rounded-full bg-[var(--app-accent-soft-bg)] px-2 py-1 text-[var(--app-accent-soft-fg)]">
                      {teams.find((team) => team.id === id)?.name || id}
                    </span>
                  ))}
                  {(row.shared_user_ids || []).map((id) => (
                    <span key={id} className="rounded-full bg-[var(--app-success-soft-bg)] px-2 py-1 text-[var(--app-success-soft-fg)]">
                      {users.find((user) => user.id === id)?.display_name || id}
                    </span>
                  ))}
                  {(row.shared_team_ids || []).length + (row.shared_user_ids || []).length === 0 ? <span style={{ color: "var(--app-muted)" }}>Owner only</span> : null}
                </div>
              ),
            },
            {
              key: "actions",
              label: "",
              className: "w-80",
              render: (row) => (
                <div className="flex flex-wrap justify-end gap-2">
                  <button className="btn h-8" onClick={() => checkHealth(row)} title="Check health">
                    <HeartPulse size={15} /> Health
                  </button>
                  <button className="btn h-8" onClick={() => edit(row)}>
                    <Share2 size={15} /> Configure
                  </button>
                  <button className="btn h-8" onClick={() => setEnabled(row, !row.enabled)}>
                    <Power size={15} /> {row.enabled ? "Disable" : "Enable"}
                  </button>
                </div>
              ),
            },
          ]}
        />
      )}

      <Modal title={editing ? "Configure channel account" : "Add system channel"} open={open} onClose={() => setOpen(false)}>
        <form className="space-y-5" onSubmit={save}>
          <section className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-semibold">
              <CheckCircle2 size={17} className="text-sky-600" /> Channel identity
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <FormField label="Channel type">
                <select className="field" value={form.channel_id} onChange={(event) => setForm({ ...form, channel_id: event.target.value })} required>
                  <option value="">Select channel</option>
                  {availableChannels.map((channel) => (
                    <option key={channel.id} value={channel.id}>
                      {channel.name}
                    </option>
                  ))}
                </select>
              </FormField>
              <FormField label="Account display name">
                <input className="field" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="Sales FB Page" required />
              </FormField>
            </div>
            <FormField label="Owner team">
              <select className="field" value={form.owner_team_id} onChange={(event) => setForm({ ...form, owner_team_id: event.target.value })}>
                <option value="">System owned</option>
                {teams.map((team) => (
                  <option key={team.id} value={team.id}>{team.name}</option>
                ))}
              </select>
            </FormField>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-semibold">
              <Share2 size={17} className="text-sky-600" /> Message sharing
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <FormField label="Shared teams">
                <select
                  className="field h-32"
                  multiple
                  value={form.shared_team_ids}
                  onChange={(event) => setForm({ ...form, shared_team_ids: Array.from(event.target.selectedOptions).map((option) => option.value) })}
                >
                  {teams.map((team) => (
                    <option key={team.id} value={team.id}>{team.name}</option>
                  ))}
                </select>
              </FormField>
              <FormField label="Shared users">
                <select
                  className="field h-32"
                  multiple
                  value={form.shared_user_ids}
                  onChange={(event) => setForm({ ...form, shared_user_ids: Array.from(event.target.selectedOptions).map((option) => option.value) })}
                >
                  {users.map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name} ({user.email})</option>
                  ))}
                </select>
              </FormField>
            </div>
          </section>

          <section className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <FormField label="Credential reference">
                <input className="field" value={form.credential_ref} onChange={(event) => setForm({ ...form, credential_ref: event.target.value })} placeholder="secret://fb-page-sales" />
              </FormField>
              <FormField label="Webhook secret reference">
                <input className="field" value={form.webhook_secret_ref} onChange={(event) => setForm({ ...form, webhook_secret_ref: event.target.value })} placeholder="secret://webhook-sales" />
              </FormField>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />
              Enable account
            </label>
          </section>

          {error ? <div className="rounded-md bg-red-50 p-3 text-sm text-danger">{error}</div> : null}
          <div className="flex justify-end gap-2">
            <button className="btn" type="button" onClick={() => setOpen(false)}>Cancel</button>
            <button className="btn btn-primary">
              <Save size={16} /> Save channel
            </button>
          </div>
        </form>
      </Modal>
    </AdminShell>
  );
}
