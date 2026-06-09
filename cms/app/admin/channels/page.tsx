"use client";

import { AlertTriangle, BotMessageSquare, CheckCircle2, Clock3, Loader2, MessageCircle, Plug, Power, RefreshCw, RotateCcw, Save, Settings2 } from "lucide-react";
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
  connected: "Đã kết nối",
  qr: "Chờ quét QR",
  connecting: "Đang kết nối",
  disconnected: "Chưa kết nối",
  error: "Lỗi",
  not_configured: "Chờ cấu hình",
  unknown: "Chưa kết nối",
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
  const [now, setNow] = useState(() => Date.now());

  const whatsAppChannel = useMemo(() => channels.find((channel) => channel.code === "whatsapp"), [channels]);
  const selectedChannel = useMemo(() => channels.find((channel) => channel.id === selectedChannelID), [channels, selectedChannelID]);
  const whatsAppAccount = useMemo(
    () => accounts.find((account) => account.channel_id === whatsAppChannel?.id),
    [accounts, whatsAppChannel?.id],
  );
  const effectiveStatus = session?.status || whatsAppAccount?.session_status || "unknown";
  const qrExpiresAt = session ? session.qr_expires_at || session.qrExpiresAt : undefined;
  const qrExpired = Boolean(qrExpiresAt && now > Date.parse(qrExpiresAt));
  const qr = qrExpired ? "" : session?.qr || "";
  const qrExpiresIn = qrExpiresAt ? Math.max(0, Math.ceil((Date.parse(qrExpiresAt) - now) / 1000)) : 0;

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

  useEffect(() => {
    if (!session?.qr && !session?.qr_expires_at && !session?.qrExpiresAt) return;
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, [session?.qr, session?.qr_expires_at, session?.qrExpiresAt]);

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
      setSession({ accountId: account.id, status: "error", lastError: "Không thể kết nối WhatsApp adapter" });
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
      setError("Vui lòng lưu cài đặt trước khi kết nối WhatsApp.");
      return;
    }
    setWorking(action);
    setError("");
    try {
      const result = await fn(whatsAppAccount);
      if (result) {
        setSession(result);
        if (action === "connect" && (!result.qr || isExpiredQR(result))) {
          setSession(await pollWhatsAppSession(whatsAppAccount.id));
        }
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Thao tác WhatsApp thất bại");
    } finally {
      setWorking("");
    }
  }

  async function pollWhatsAppSession(accountId: string) {
    let latest = session;
    for (let attempt = 0; attempt < 12; attempt += 1) {
      await sleep(attempt < 3 ? 1000 : 2500);
      const next = await api.whatsAppSession(token!, accountId);
      latest = next;
      setSession(next);
      if ((next.qr && !isExpiredQR(next)) || next.status === "connected" || next.status === "error" || next.status === "disconnected") {
        return next;
      }
    }
    return latest || { accountId, status: "connecting" as const };
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
            Thiết lập tài khoản channel trước khi vận hành inbox Chat.
          </p>
        </div>
        <div className="flex flex-col items-start gap-2 sm:items-end">
          <StatusPill value={statusLabel[effectiveStatus] || effectiveStatus} />
          <StatusColorLegend />
        </div>
      </div>

      <section className="mb-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {(channels.length ? channels : [{ id: "ch_whatsapp", code: "whatsapp", name: "WhatsApp", kind: "whatsapp", official_api_available: false, status: "enabled" }]).map((channel) => {
          const account = accounts.find((item) => item.channel_id === channel.id);
          const status = channel.code === "whatsapp" ? effectiveStatus : account?.session_status || "not_configured";
          const StatusIcon = statusIcon(status);
          const tone = statusTone(status);
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
                <div
                  className="flex h-10 w-10 items-center justify-center rounded-md"
                  style={{
                    background: tone.background,
                    color: tone.foreground,
                  }}
                >
                  <StatusIcon size={20} />
                </div>
                <StatusPill value={statusLabel[status] || status} />
              </div>
              <div className="font-semibold">{channel.name}</div>
              <div className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
                {channel.code === "whatsapp" ? channelStatusDescription(status) : channel.kind}
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
          Chọn một channel ở phía trên để xem và thiết lập cấu hình.
        </div>
      ) : selectedChannel.code !== "whatsapp" ? (
        <div className="panel flex min-h-64 items-center justify-center p-6 text-center text-sm" style={{ color: "var(--app-muted)" }}>
          Channel {selectedChannel.name} chưa có màn hình cấu hình riêng trong phiên bản này.
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
                  Adapter đầu tiên của Omni Channel qua Baileys.
                </p>
              </div>
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <FormField label="Tên channel">
                <input className="field" value={form.channelName} onChange={(event) => setForm({ ...form, channelName: event.target.value })} />
              </FormField>
              <FormField label="Tên tài khoản">
                <input className="field" value={form.accountLabel} onChange={(event) => setForm({ ...form, accountLabel: event.target.value })} placeholder="VD: Sale WhatsApp" />
              </FormField>
              <FormField label="Số điện thoại">
                <input className="field" value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} placeholder="84901234567" />
              </FormField>
              <FormField label="Browser name">
                <input className="field" value={form.browserName} onChange={(event) => setForm({ ...form, browserName: event.target.value })} placeholder="Omni Channel CMS" />
              </FormField>
            </div>

            <div className="mt-6 space-y-3">
              <Checkbox checked={form.enabled} label="Bật channel này" onChange={(checked) => setForm({ ...form, enabled: checked })} />
              <Checkbox checked={form.autoConnect} label="Tự kết nối khi API khởi động" onChange={(checked) => setForm({ ...form, autoConnect: checked })} />
              <Checkbox checked={form.syncFullHistory} label="Đồng bộ thêm lịch sử WhatsApp khi kết nối" onChange={(checked) => setForm({ ...form, syncFullHistory: checked })} />
            </div>

            {error ? (
              <div className="mt-5 rounded-md bg-red-50 p-3 text-sm text-danger flex items-center gap-2">
                {error.includes("Mất kết nối") && <Loader2 className="h-4 w-4 animate-spin text-danger" />}
                <span>{error}</span>
              </div>
            ) : null}

            <div className="mt-6 flex flex-wrap gap-2">
              <button className="btn btn-primary" onClick={saveSettings} disabled={saving || !whatsAppChannel}>
                {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save size={16} />} Lưu cài đặt
              </button>
              <button className="btn" onClick={() => withAccount("connect", (account) => api.whatsAppConnect(token!, account.id))} disabled={!whatsAppAccount || working === "connect" || effectiveStatus === "connected" || effectiveStatus === "connecting"}>
                {working === "connect" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plug size={16} />} Kết nối
              </button>
              <button className="btn" onClick={() => withAccount("disconnect", (account) => api.whatsAppDisconnect(token!, account.id))} disabled={!whatsAppAccount || working === "disconnect"}>
                {working === "disconnect" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Power size={16} />} Ngắt kết nối
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
            <h2 className="font-semibold">Setup lần đầu</h2>
            <p className="mt-1 text-sm" style={{ color: "var(--app-muted)" }}>
              Lưu cài đặt, bấm kết nối, rồi quét QR bằng WhatsApp trên điện thoại.
            </p>

            <div className="mt-5 flex min-h-[240px] items-center justify-center rounded-md border border-dashed p-4" style={{ borderColor: "var(--app-border)", background: "var(--app-bg)" }}>
              {qr ? (
                <div className="rounded-md bg-white p-3">
                  <QRCodeSVG value={qr} size={196} />
                </div>
              ) : effectiveStatus === "qr" || effectiveStatus === "connecting" ? (
                <div className="flex flex-col items-center gap-2 text-sm" style={{ color: "var(--app-muted)" }}>
                  <Loader2 className="h-6 w-6 animate-spin" />
                  Đang chờ QR...
                </div>
              ) : (
                <div className="text-center text-sm" style={{ color: "var(--app-muted)" }}>
                  {qrExpired ? "QR vừa hết hạn, hệ thống đang chờ QR mới." : "QR sẽ hiển thị khi trạng thái là Chờ quét QR."}
                </div>
              )}
            </div>

            <div className="mt-5 space-y-2 text-sm" style={{ color: "var(--app-muted)" }}>
              {qr && qrExpiresIn ? <p className="font-medium text-blue-700">QR còn hiệu lực khoảng {qrExpiresIn} giây.</p> : null}
              {session?.cached ? <p className="font-medium text-amber-700">Đang dùng QR đã cache trên server để hạn chế gọi adapter liên tục.</p> : null}
              <p>1. Mở WhatsApp trên điện thoại.</p>
              <p>2. Vào Linked devices.</p>
              <p>3. Quét QR và đợi trạng thái chuyển sang Đã kết nối.</p>
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

function StatusColorLegend() {
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs" style={{ color: "var(--app-muted)" }}>
      <LegendItem color="var(--app-success-soft-fg)" label="Xanh lá: đã kết nối" />
      <LegendItem color="var(--app-warning-soft-fg)" label="Vàng: đang kết nối / chờ QR / chờ cấu hình" />
      <LegendItem color="var(--app-danger-soft-fg)" label="Đỏ: lỗi" />
      <LegendItem color="var(--app-muted)" label="Xám: chưa kết nối" />
    </div>
  );
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 whitespace-nowrap">
      <span className="h-2 w-2 rounded-full" style={{ background: color }} />
      {label}
    </span>
  );
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isExpiredQR(session: WhatsAppSession) {
  const expiresAt = session.qr_expires_at || session.qrExpiresAt;
  return Boolean(expiresAt && Date.now() > Date.parse(expiresAt));
}

function statusIcon(status: string) {
  if (status === "connected") return CheckCircle2;
  if (status === "error") return AlertTriangle;
  if (status === "connecting" || status === "qr") return Clock3;
  if (status === "not_configured") return Settings2;
  return MessageCircle;
}

function channelStatusDescription(status: string) {
  if (status === "connected") return "Session đang hoạt động";
  if (status === "qr") return "Đang chờ quét QR";
  if (status === "connecting") return "Adapter đang kết nối";
  if (status === "error") return "Cần kiểm tra adapter hoặc session";
  if (status === "not_configured") return "Chưa tạo tài khoản channel";
  if (status === "disconnected" || status === "unknown") return "Đã cấu hình nhưng chưa kết nối";
  return "Baileys adapter";
}

function statusTone(status: string) {
  if (status === "connected") {
    return { background: "var(--app-success-soft-bg)", foreground: "var(--app-success-soft-fg)" };
  }
  if (status === "connecting" || status === "qr" || status === "not_configured") {
    return { background: "var(--app-warning-soft-bg)", foreground: "var(--app-warning-soft-fg)" };
  }
  if (status === "error") {
    return { background: "var(--app-danger-soft-bg)", foreground: "var(--app-danger-soft-fg)" };
  }
  return { background: "var(--app-surface-muted)", foreground: "var(--app-muted-strong)" };
}
