import makeWASocket, { Browsers, DisconnectReason, fetchLatestBaileysVersion, useMultiFileAuthState } from "@whiskeysockets/baileys";
import { rm } from "node:fs/promises";
import pino from "pino";
import { buildInboundEnvelope, normalizeBaileysMessage, outboundToBaileys } from "./normalize.js";
import { SessionStore } from "./sessionStore.js";
import type { InboundEnvelope, MessageStatusPayload, OutboundPayload } from "./contracts.js";

type Publisher = {
  publish(envelope: InboundEnvelope): Promise<void>;
};

export class WhatsAppBaileysClient {
  private sockets = new Map<string, ReturnType<typeof makeWASocket>>();
  private reconnectAttempts = new Map<string, number>();
  private reconnectTimers = new Map<string, NodeJS.Timeout>();
  private manualDisconnects = new Set<string>();
  private presenceSubscriptions = new Map<string, number>();

  constructor(
    private readonly sessions: SessionStore,
    private readonly publisher: Publisher,
    private readonly sessionsDir: string,
    private readonly apiBaseURL: string,
    private readonly webhookSharedSecret: string,
  ) {}

  async connect(accountId: string) {
    this.manualDisconnects.delete(accountId);
    this.clearReconnect(accountId);
    const current = this.sessions.snapshot(accountId);
    if (this.sockets.has(accountId) && ["connecting", "qr", "connected"].includes(current.status)) {
      return current;
    }
    this.sessions.set(accountId, { status: "connecting" });
    const { state, saveCreds } = await useMultiFileAuthState(`${this.sessionsDir}/${accountId}`);
    const { version } = await fetchLatestBaileysVersion();
    const socket = makeWASocket({
      auth: state,
      version,
      browser: Browsers.ubuntu("Omni Channel"),
      connectTimeoutMs: 60_000,
      defaultQueryTimeoutMs: 60_000,
      keepAliveIntervalMs: 20_000,
      markOnlineOnConnect: false,
      retryRequestDelayMs: 2_000,
      syncFullHistory: true,
      logger: pino({ level: "silent" }),
    });
    this.sockets.set(accountId, socket);
    socket.ev.on("creds.update", saveCreds);
    socket.ev.on("connection.update", (update) => {
      if (update.qr) {
        const generatedAt = new Date();
        this.sessions.set(accountId, {
          status: "qr",
          qr: update.qr,
          qrGeneratedAt: generatedAt.toISOString(),
          qrExpiresAt: new Date(generatedAt.getTime() + 25_000).toISOString(),
        });
      }
      if (update.connection === "open") {
        this.reconnectAttempts.set(accountId, 0);
        this.sessions.set(accountId, { status: "connected", qr: undefined, qrExpiresAt: undefined, lastError: undefined });
      }
      if (update.connection === "close") {
        const statusCode = Number((update.lastDisconnect?.error as { output?: { statusCode?: number } } | undefined)?.output?.statusCode);
        this.sockets.delete(accountId);
        this.sessions.set(accountId, { status: "disconnected", qr: undefined, qrExpiresAt: undefined, lastError: update.lastDisconnect?.error?.message });
        if (statusCode !== DisconnectReason.loggedOut && !this.manualDisconnects.has(accountId)) this.scheduleReconnect(accountId);
      }
    });
    socket.ev.on("messages.upsert", async ({ messages }) => {
      for (const raw of messages) {
        const normalized = normalizeBaileysMessage(accountId, raw);
        if (!normalized) continue;
        const envelope = buildInboundEnvelope(accountId, normalized);
        if (!this.sessions.firstSeen(envelope.idempotency_key)) continue;
        await this.publisher.publish(envelope);
      }
    });
    socket.ev.on("presence.update", (update) => {
      const jid = update.id;
      if (!jid) return;
      const presences = Object.values(update.presences || {});
      const typing = presences.some((presence) => presence.lastKnownPresence === "composing" || presence.lastKnownPresence === "recording");
      const paused = presences.some((presence) => presence.lastKnownPresence === "paused" || presence.lastKnownPresence === "available" || presence.lastKnownPresence === "unavailable");
      if (typing || paused) {
        console.log(`whatsapp presence account=${accountId} jid=${jid} typing=${typing}`);
        this.sessions.setTyping(accountId, jid, typing);
      }
    });
    socket.ev.on("messages.update", (updates) => {
      for (const item of updates) {
        const messageID = item.key.id;
        if (!messageID || !item.key.fromMe) continue;
        const status = whatsappStatusLabel(Number(item.update.status));
        if (!status) continue;
        const timestamp = Number(item.update.messageTimestamp || Math.floor(Date.now() / 1000));
        void this.publishMessageStatus({
          channel_account_id: accountId,
          channel_message_id: messageID,
          status,
          event_time: new Date(timestamp * 1000).toISOString(),
        });
      }
    });
    return this.sessions.waitFor(accountId, (snapshot) => Boolean(snapshot.qr) || ["connected", "error"].includes(snapshot.status), 15000);
  }

  async send(payload: OutboundPayload) {
    const socket = this.sockets.get(payload.channel_account_id);
    if (!socket) throw new Error("whatsapp session is not connected");
    const mapped = outboundToBaileys(payload);
    
    // Normalize JID if it starts with 0 and is for s.whatsapp.net
    let jid = mapped.jid;
    if (jid.startsWith("0") && jid.endsWith("@s.whatsapp.net")) {
      jid = "84" + jid.slice(1);
    }

    const sendPromise = socket.sendMessage(jid, mapped.content);
    const timeoutPromise = new Promise<never>((_, reject) =>
      setTimeout(() => reject(new Error("Baileys sendMessage timeout")), 12000)
    );

    return Promise.race([sendPromise, timeoutPromise]);
  }

  async disconnect(accountId: string) {
    this.manualDisconnects.add(accountId);
    this.clearReconnect(accountId);
    this.clearPresenceSubscriptions(accountId);
    const socket = this.sockets.get(accountId);
    if (socket) {
      await socket.ws.close().catch(() => undefined);
      this.sockets.delete(accountId);
    }
    this.sessions.set(accountId, { status: "disconnected", qr: undefined, qrExpiresAt: undefined });
  }

  async resetSession(accountId: string) {
    this.manualDisconnects.add(accountId);
    this.clearReconnect(accountId);
    const socket = this.sockets.get(accountId);
    if (socket) {
      await socket.logout().catch(() => undefined);
      this.sockets.delete(accountId);
    }
    await rm(`${this.sessionsDir}/${accountId}`, { recursive: true, force: true });
    this.clearPresenceSubscriptions(accountId);
    this.sessions.clearSession(accountId);
    this.sessions.set(accountId, { status: "disconnected", qr: undefined, qrExpiresAt: undefined });
  }

  async resync(accountId?: string) {
    const ids = accountId ? [accountId] : [...this.sockets.keys()];
    for (const id of ids) {
      this.sessions.set(id, { lastSyncAt: new Date().toISOString() });
    }
  }

  async typing(accountId: string, jid: string) {
    await this.ensurePresenceSubscription(accountId, jid);
    return this.sessions.typingSnapshot(accountId, jid);
  }

  async onWhatsApp(accountId: string, phone: string): Promise<boolean> {
    const socket = this.sockets.get(accountId);
    if (!socket) return false;
    try {
      let cleanPhone = phone.replace(/[^0-9]/g, "");
      if (!cleanPhone) return false;
      const results = await socket.onWhatsApp(cleanPhone);
      if (results && results.length > 0) {
        return results[0].exists;
      }
      return false;
    } catch (error) {
      const msg = error instanceof Error ? error.message : "unknown error";
      console.warn(`whatsapp onWhatsApp check failed account=${accountId} phone=${phone}: ${msg}`);
      return false;
    }
  }

  async profilePictureUrl(accountId: string, jid: string): Promise<string> {
    const socket = this.sockets.get(accountId);
    if (!socket) return "";
    try {
      let cleanJid = jid;
      if (!cleanJid.includes("@")) {
        cleanJid = cleanJid + "@s.whatsapp.net";
      }
      const url = await socket.profilePictureUrl(cleanJid, "image");
      return url || "";
    } catch (error) {
      return "";
    }
  }

  private async ensurePresenceSubscription(accountId: string, jid: string) {
    const socket = this.sockets.get(accountId);
    if (!socket || !jid) return;
    const key = `${accountId}:${jid}`;
    const lastSubscribedAt = this.presenceSubscriptions.get(key) || 0;
    if (Date.now() - lastSubscribedAt < 60_000) return;
    try {
      await socket.presenceSubscribe(jid);
      this.presenceSubscriptions.set(key, Date.now());
      console.log(`whatsapp presence subscribed account=${accountId} jid=${jid}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : "unknown error";
      console.warn(`whatsapp presence subscribe failed account=${accountId} jid=${jid}: ${message}`);
    }
  }

  private scheduleReconnect(accountId: string) {
    this.clearReconnect(accountId);
    const attempt = (this.reconnectAttempts.get(accountId) || 0) + 1;
    this.reconnectAttempts.set(accountId, attempt);
    const delayMs = Math.min(60_000, 2_000 * 2 ** Math.min(attempt - 1, 5));
    this.sessions.set(accountId, {
      status: "connecting",
      lastError: `Disconnected; reconnecting in ${Math.round(delayMs / 1000)}s`,
    });
    const timer = setTimeout(() => {
      this.reconnectTimers.delete(accountId);
      if (!this.manualDisconnects.has(accountId)) void this.connect(accountId);
    }, delayMs);
    this.reconnectTimers.set(accountId, timer);
  }

  private clearReconnect(accountId: string) {
    const timer = this.reconnectTimers.get(accountId);
    if (timer) clearTimeout(timer);
    this.reconnectTimers.delete(accountId);
  }

  private clearPresenceSubscriptions(accountId: string) {
    for (const key of [...this.presenceSubscriptions.keys()]) {
      if (key.startsWith(`${accountId}:`)) this.presenceSubscriptions.delete(key);
    }
  }

  private async publishMessageStatus(payload: MessageStatusPayload) {
    try {
      const headers: Record<string, string> = { "Content-Type": "application/json" };
      if (this.webhookSharedSecret) headers["X-Webhook-Secret"] = this.webhookSharedSecret;
      const response = await fetch(`${this.apiBaseURL.replace(/\/$/, "")}/internal/whatsapp/message-status`, {
        method: "POST",
        headers,
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        console.warn(`whatsapp message status update failed message=${payload.channel_message_id} status=${response.status}`);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "unknown error";
      console.warn(`whatsapp message status update failed message=${payload.channel_message_id}: ${message}`);
    }
  }
}

function whatsappStatusLabel(status: number): MessageStatusPayload["status"] | null {
  if (status >= 4) return "read";
  if (status >= 3) return "delivered";
  if (status >= 2) return "sent";
  return null;
}
