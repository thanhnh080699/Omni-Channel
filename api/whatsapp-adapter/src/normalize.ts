import type { InboundEnvelope, NormalizedInboundMessage } from "./contracts.js";

type BaileysLikeMessage = {
  key?: {
    id?: string | null;
    remoteJid?: string | null;
    participant?: string | null;
    fromMe?: boolean | null;
  };
  messageTimestamp?: number | LongLike | null;
  pushName?: string | null;
  message?: unknown;
};

type LongLike = {
  toNumber: () => number;
};

export function normalizeBaileysMessage(accountId: string, message: BaileysLikeMessage): NormalizedInboundMessage | null {
  if (message.key?.fromMe) return null;
  const externalMessageId = message.key?.id || "";
  const conversationId = message.key?.remoteJid || "";
  if (!externalMessageId || !conversationId) return null;

  const timestampSeconds = toTimestampSeconds(message.messageTimestamp);
  const text = extractText(message.message);
  return {
    channel: "whatsapp",
    channel_account_id: accountId,
    external_conversation_id: conversationId,
    external_message_id: externalMessageId,
    sender_external_id: message.key?.participant || conversationId,
    sender_display_name: message.pushName || undefined,
    direction: "inbound",
    text,
    event_time: new Date(timestampSeconds * 1000).toISOString(),
    raw_event_id: externalMessageId,
    raw_payload: message,
  };
}

export function buildInboundEnvelope(accountId: string, normalized: NormalizedInboundMessage, now = new Date()): InboundEnvelope {
  return {
    event_id: normalized.raw_event_id,
    idempotency_key: `${accountId}:${normalized.raw_event_id}`,
    channel: "whatsapp",
    channel_account_id: accountId,
    conversation_id: normalized.external_conversation_id,
    raw_payload: {
      external_conversation_id: normalized.external_conversation_id,
      external_message_id: normalized.external_message_id,
      sender_external_id: normalized.sender_external_id,
      sender_display_name: normalized.sender_display_name,
      text: normalized.text,
      event_time: normalized.event_time,
    },
    gateway_received_at: now.toISOString(),
    event_time: normalized.event_time,
    queued_at: now.toISOString(),
    attempt: 0,
    trace_id: `${accountId}:${normalized.raw_event_id}`,
    normalized,
  };
}

export function outboundToBaileys(payload: { external_conversation_id?: string; text?: string }) {
  if (!payload.external_conversation_id) {
    throw new Error("external_conversation_id is required");
  }
  return {
    jid: payload.external_conversation_id,
    content: { text: payload.text || "" },
  };
}

function extractText(message?: unknown) {
  if (!message || typeof message !== "object") return "";
  const value = message as Record<string, unknown>;
  const conversation = value.conversation;
  if (typeof conversation === "string") return conversation;
  const extended = value.extendedTextMessage as { text?: unknown } | undefined;
  if (typeof extended?.text === "string") return extended.text;
  const image = value.imageMessage as { caption?: unknown } | undefined;
  if (typeof image?.caption === "string") return image.caption;
  const video = value.videoMessage as { caption?: unknown } | undefined;
  if (typeof video?.caption === "string") return video.caption;
  return "";
}

function toTimestampSeconds(value: BaileysLikeMessage["messageTimestamp"]) {
  if (typeof value === "number") return value;
  if (value && typeof value.toNumber === "function") return value.toNumber();
  return Math.floor(Date.now() / 1000);
}
