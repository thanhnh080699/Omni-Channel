import makeWASocket, { DisconnectReason, fetchLatestBaileysVersion, useMultiFileAuthState } from "@whiskeysockets/baileys";
import { rm } from "node:fs/promises";
import pino from "pino";
import { buildInboundEnvelope, normalizeBaileysMessage, outboundToBaileys } from "./normalize.js";
import { SessionStore } from "./sessionStore.js";
import type { InboundEnvelope, OutboundPayload } from "./contracts.js";

type Publisher = {
  publish(envelope: InboundEnvelope): Promise<void>;
};

export class WhatsAppBaileysClient {
  private sockets = new Map<string, ReturnType<typeof makeWASocket>>();

  constructor(
    private readonly sessions: SessionStore,
    private readonly publisher: Publisher,
    private readonly sessionsDir: string,
  ) {}

  async connect(accountId: string) {
    this.sessions.set(accountId, { status: "connecting" });
    const { state, saveCreds } = await useMultiFileAuthState(`${this.sessionsDir}/${accountId}`);
    const { version } = await fetchLatestBaileysVersion();
    const socket = makeWASocket({
      auth: state,
      version,
      logger: pino({ level: "silent" }),
    });
    this.sockets.set(accountId, socket);
    socket.ev.on("creds.update", saveCreds);
    socket.ev.on("connection.update", (update) => {
      if (update.qr) this.sessions.set(accountId, { status: "qr", qr: update.qr });
      if (update.connection === "open") this.sessions.set(accountId, { status: "connected", qr: undefined, lastError: undefined });
      if (update.connection === "close") {
        const statusCode = Number((update.lastDisconnect?.error as { output?: { statusCode?: number } } | undefined)?.output?.statusCode);
        this.sessions.set(accountId, { status: "disconnected", lastError: update.lastDisconnect?.error?.message });
        if (statusCode !== DisconnectReason.loggedOut) void this.connect(accountId);
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
  }

  async send(payload: OutboundPayload) {
    const socket = this.sockets.get(payload.channel_account_id);
    if (!socket) throw new Error("whatsapp session is not connected");
    const mapped = outboundToBaileys(payload);
    return socket.sendMessage(mapped.jid, mapped.content);
  }

  async disconnect(accountId: string) {
    const socket = this.sockets.get(accountId);
    if (socket) {
      await socket.logout().catch(() => undefined);
      this.sockets.delete(accountId);
    }
    this.sessions.set(accountId, { status: "disconnected", qr: undefined });
  }

  async resetSession(accountId: string) {
    const socket = this.sockets.get(accountId);
    if (socket) {
      await socket.logout().catch(() => undefined);
      this.sockets.delete(accountId);
    }
    await rm(`${this.sessionsDir}/${accountId}`, { recursive: true, force: true });
    this.sessions.clearSession(accountId);
    this.sessions.set(accountId, { status: "disconnected", qr: undefined });
  }

  async resync(accountId?: string) {
    const ids = accountId ? [accountId] : [...this.sockets.keys()];
    for (const id of ids) {
      this.sessions.set(id, { lastSyncAt: new Date().toISOString() });
    }
  }
}
