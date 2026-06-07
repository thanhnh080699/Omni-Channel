"use client";

import { BotMessageSquare, Loader2, MessageCircle, Plug, Power, RefreshCw, RotateCcw, Save } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { FormField } from "@/components/form-field";
import { StatusPill } from "@/components/status-pill";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Channel, ChannelAccount, WhatsAppSession } from "@/lib/types";

type FormState = {
  channelName: string;
  accountLabel: string;
  phone: string;
  browserName: string;
  enabled: boolean;
  autoConnect: boolean;
  syncFullHistory: boolean;
};

const initialForm: FormState = {
  channelName: "WhatsApp",
  accountLabel: "",
  phone: "",
  browserName: "Omni Channel CMS",
  enabled: true,
  autoConnect: true,
  syncFullHistory: true,
};

const statusLabel: Record<string, string> = {
  connected: "Da ket noi",
  qr: "Cho quet QR",
  connecting: "Dang ket noi",
  disconnected: "Chua ket noi",
  error: "Loi",
  unknown: "Chua ket noi",
};

export default function ChannelsPage() {
  const { token } = useAuth();
  const [channels, setChannels] = useState<Channel[]>([]);
  const [accounts, setAccounts] = useState<ChannelAccount[]>([]);
  const [session, setSession] = useState<WhatsAppSession | null>(null);
  const [form, setForm] = useState<FormState>(initialForm);
  const [selectedChannelID, setSelectedChannelID] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [working, setWorking] = useState("");
  const [error, setError] = useState("");

  const whatsAppChannel = useMemo(() => channels.find((channel) => channel.code === "whatsapp"), [channels]);
  const selectedChannel = useMemo(() => channels.find((channel) => channel.id === selectedChannelID), [channels, selectedChannelID]);
  const whatsAppAccount = useMemo(
    () => accounts.find((account) => account.channel_id === whatsAppChannel?.id),
    [accounts, whatsAppChannel?.id],
  );
  const effectiveStatus = session?.status || whatsAppAccount?.session_status || "unknown";
  const qr = session?.qr || "";

  async function load() {
    if (!token) return;
    setLoading(true);
    setError("");
    try {
      const [channelResult, accountResult] = await Promise.all([
        api.channelAdminChannels(token),
        api.channelAccounts(token),
      ]);
      const nextChannels = channelResult.data || [];
      const nextAccounts = accountResult.data || [];
      setChannels(nextChannels);
      setAccounts(nextAccounts);
      const whatsapp = nextChannels.find((channel) => channel.code === "whatsapp");
      const account = nextAccounts.find((item) => item.channel_id === whatsapp?.id);
      if (account) setForm(accountToForm(account));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load Omni Channel settings");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [token]);

  async function selectChannel(channel: Channel) {
    setSelectedChannelID(channel.id);
    setError("");
    if (channel.code !== "whatsapp") return;
    const account = accounts.find((item) => item.channel_id === channel.id);
    if (!account || !token) {
      setSession(null);
      setForm(initialForm);
      return;
    }
    setForm(accountToForm(account));
    try {
      const currentSession = await api.whatsAppSession(token, account.id);
      setSession(currentSession);
    } catch {
      setSession({ accountId: account.id, status: "error", lastError: "whatsapp adapter unavailable" });
    }
  }

  async function saveSettings() {
    if (!token || !whatsAppChannel) return;
    setSaving(true);
    setError("");
    try {
      const payload: Partial<ChannelAccount> = {
        channel_id: whatsAppChannel.id,
        name: form.accountLabel.trim() || form.channelName.trim() || "WhatsApp",
        enabled: form.enabled,
        shared_team_ids: [],
        shared_user_ids: [],
        metadata: {
          accountLabel: form.accountLabel.trim() || null,
          phone: form.phone.trim() || null,
          browserName: form.browserName.trim() || "Omni Channel CMS",
          autoConnect: form.autoConnect,
          syncFullHistory: form.syncFullHistory,
        },
      };
      if (whatsAppAccount) {
        await api.updateChannelAccount(token, whatsAppAccount.id, payload);
      } else {
        await api.createChannelAccount(token, payload);
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save settings");
    } finally {
      setSaving(false);
    }
  }

  async function withAccount(action: string, fn: (account: ChannelAccount) => Promise<WhatsAppSession | void>) {
    if (!whatsAppAccount) {
      setError("Save settings before connecting WhatsApp.");
      return;
    }
    setWorking(action);
    setError("");
    try {
      const result = await fn(whatsAppAccount);
      if (result) setSession(result);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "WhatsApp action failed");
    } finally {
      setWorking("");
    }
  }

  return (
    <AdminShell>
      <div className="mb-6 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-blue-600">
        <span>Dashboard</span>
        <span style={{ color: "var(--app-muted)" }}>/</span>
        <span>Settings</span>
        <span style={{ color: "var(--app-muted)" }}>/</span>
        <span style={{ color: "var(--app-muted)" }}>Omni Channel</span>
      </div>

      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <BotMessageSquare className="h-6 w-6" /> Omni Channel
          </h1>
          <p className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
            Thiet lap tai khoan channel truoc khi van hanh inbox Chat.
          </p>
        </div>
        <StatusPill value={statusLabel[effectiveStatus] || effectiveStatus} />
      </div>

      <section className="mb-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {(channels.length ? channels : [{ id: "ch_whatsapp", code: "whatsapp", name: "WhatsApp", kind: "whatsapp", official_api_available: false, status: "enabled" }]).map((channel) => {
          const account = accounts.find((item) => item.channel_id === channel.id);
          const status = channel.code === "whatsapp" ? effectiveStatus : account?.session_status || "not_configured";
          return (
            <button
              key={channel.id}
              className="panel p-4 text-left"
              onClick={() => void selectChannel(channel)}
              style={{
                outline: selectedChannelID === channel.id ? "2px solid var(--app-accent-soft-fg)" : "none",
                outlineOffset: "2px",
              }}
            >
              <div className="mb-3 flex items-start justify-between gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-md bg-[var(--app-accent-soft-bg)] text-[var(--app-accent-soft-fg)]">
                  <MessageCircle size={20} />
                </div>
                <StatusPill value={statusLabel[status] || status} />
              </div>
              <div className="font-semibold">{channel.name}</div>
              <div className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
                {channel.code === "whatsapp" ? "Baileys adapter" : channel.kind}
              </div>
            </button>
          );
        })}
      </section>

      {loading ? (
        <div className="panel flex min-h-80 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin" style={{ color: "var(--app-muted)" }} />
        </div>
      ) : !selectedChannel ? (
        <div className="panel flex min-h-64 items-center justify-center p-6 text-center text-sm" style={{ color: "var(--app-muted)" }}>
          Chon mot channel o phia tren de xem va thiet lap cau hinh.
        </div>
      ) : selectedChannel.code !== "whatsapp" ? (
        <div className="panel flex min-h-64 items-center justify-center p-6 text-center text-sm" style={{ color: "var(--app-muted)" }}>
          Channel {selectedChannel.name} chua co man hinh cau hinh rieng trong phien ban nay.
        </div>
      ) : (
        <div className="grid gap-6 xl:grid-cols-[1fr_420px]">
          <section className="panel p-5">
            <div className="mb-5 flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-md bg-[var(--app-accent-soft-bg)] text-[var(--app-accent-soft-fg)]">
                <MessageCircle className="h-5 w-5" />
              </span>
              <div>
                <h2 className="font-semibold">WhatsApp</h2>
                <p className="text-sm" style={{ color: "var(--app-muted)" }}>
                  Adapter dau tien cua Omni Channel qua Baileys.
                </p>
              </div>
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <FormField label="Ten channel">
                <input className="field" value={form.channelName} onChange={(event) => setForm({ ...form, channelName: event.target.value })} />
              </FormField>
              <FormField label="Ten tai khoan">
                <input className="field" value={form.accountLabel} onChange={(event) => setForm({ ...form, accountLabel: event.target.value })} placeholder="VD: Sale WhatsApp" />
              </FormField>
              <FormField label="So dien thoai">
                <input className="field" value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} placeholder="84901234567" />
              </FormField>
              <FormField label="Browser name">
                <input className="field" value={form.browserName} onChange={(event) => setForm({ ...form, browserName: event.target.value })} placeholder="Omni Channel CMS" />
              </FormField>
            </div>

            <div className="mt-6 space-y-3">
              <Checkbox checked={form.enabled} label="Bat channel nay" onChange={(checked) => setForm({ ...form, enabled: checked })} />
              <Checkbox checked={form.autoConnect} label="Tu ket noi khi API khoi dong" onChange={(checked) => setForm({ ...form, autoConnect: checked })} />
              <Checkbox checked={form.syncFullHistory} label="Dong bo them lich su WhatsApp khi ket noi" onChange={(checked) => setForm({ ...form, syncFullHistory: checked })} />
            </div>

            {error ? <div className="mt-5 rounded-md bg-red-50 p-3 text-sm text-danger">{error}</div> : null}

            <div className="mt-6 flex flex-wrap gap-2">
              <button className="btn btn-primary" onClick={saveSettings} disabled={saving || !whatsAppChannel}>
                {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save size={16} />} Luu cai dat
              </button>
              <button className="btn" onClick={() => withAccount("connect", (account) => api.whatsAppConnect(token!, account.id))} disabled={!whatsAppAccount || working === "connect" || effectiveStatus === "connected" || effectiveStatus === "connecting"}>
                {working === "connect" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plug size={16} />} Ket noi
              </button>
              <button className="btn" onClick={() => withAccount("disconnect", (account) => api.whatsAppDisconnect(token!, account.id))} disabled={!whatsAppAccount || working === "disconnect"}>
                {working === "disconnect" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Power size={16} />} Ngat ket noi
              </button>
              <button className="btn" onClick={() => withAccount("reset", (account) => api.whatsAppResetSession(token!, account.id))} disabled={!whatsAppAccount || working === "reset"}>
                {working === "reset" ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw size={16} />} Reset session
              </button>
              <button className="btn" onClick={() => withAccount("resync", async (account) => { await api.whatsAppResync(token!, account.id); return api.whatsAppSession(token!, account.id); })} disabled={!whatsAppAccount || working === "resync"}>
                {working === "resync" ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw size={16} />} Resync
              </button>
            </div>
          </section>

          <section className="panel p-5">
            <h2 className="font-semibold">Setup lan dau</h2>
            <p className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
              Luu cai dat, bam ket noi, roi quet QR bang WhatsApp tren dien thoai.
            </p>

            <div className="mt-5 flex min-h-[240px] items-center justify-center rounded-md border border-dashed p-4" style={{ borderColor: "var(--app-border)", background: "var(--app-bg)" }}>
              {qr ? (
                <div className="rounded-md bg-white p-3">
                  <QRCodeSVG value={qr} size={196} />
                </div>
              ) : effectiveStatus === "qr" || effectiveStatus === "connecting" ? (
                <div className="flex flex-col items-center gap-2 text-sm" style={{ color: "var(--app-muted)" }}>
                  <Loader2 className="h-6 w-6 animate-spin" />
                  Dang cho QR...
                </div>
              ) : (
                <div className="text-center text-sm" style={{ color: "var(--app-muted)" }}>
                  QR se hien thi khi trang thai la Cho quet QR.
                </div>
              )}
            </div>

            <div className="mt-5 space-y-2 text-sm" style={{ color: "var(--app-muted)" }}>
              {session?.cached ? <p className="font-medium text-amber-700">Dang dung QR da cache tren server de han che goi adapter lien tuc.</p> : null}
              <p>1. Mo WhatsApp tren dien thoai.</p>
              <p>2. Vao Linked devices.</p>
              <p>3. Quet QR va doi trang thai chuyen sang Da ket noi.</p>
            </div>
          </section>
        </div>
      )}
    </AdminShell>
  );
}

function accountToForm(account: ChannelAccount): FormState {
  return {
    channelName: "WhatsApp",
    accountLabel: account.metadata?.accountLabel || account.name || "",
    phone: account.metadata?.phone || "",
    browserName: account.metadata?.browserName || "Omni Channel CMS",
    enabled: account.enabled,
    autoConnect: account.metadata?.autoConnect !== false,
    syncFullHistory: account.metadata?.syncFullHistory !== false,
  };
}

function Checkbox({ checked, label, onChange }: { checked: boolean; label: string; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-center gap-3 text-sm">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      {label}
    </label>
  );
}
