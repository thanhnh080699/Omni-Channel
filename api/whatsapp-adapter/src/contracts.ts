export type NormalizedInboundMessage = {
  channel: "whatsapp";
  channel_account_id: string;
  external_conversation_id: string;
  external_message_id: string;
  sender_external_id: string;
  sender_display_name?: string;
  direction: "inbound";
  text?: string;
  attachments?: Array<{
    id?: string;
    type: string;
    url?: string;
    mime_type?: string;
    size_bytes?: number;
    filename?: string;
    content_hash?: string;
  }>;
  event_time: string;
  raw_event_id: string;
  raw_payload?: unknown;
};

export type InboundEnvelope = {
  event_id: string;
  idempotency_key: string;
  channel: "whatsapp";
  channel_account_id: string;
  conversation_id: string;
  raw_payload: Record<string, unknown>;
  gateway_received_at: string;
  event_time: string;
  queued_at: string;
  attempt: number;
  trace_id: string;
  normalized: NormalizedInboundMessage;
};

export type OutboundPayload = {
  message_id: string;
  outbound_event_id?: string;
  channel_account_id: string;
  idempotency_key: string;
  attempt: number;
  queued_at: string;
  expires_at: string;
  external_conversation_id?: string;
  text?: string;
};

export type SessionSnapshot = {
  accountId: string;
  status: "disconnected" | "connecting" | "qr" | "connected" | "error";
  qr?: string;
  lastSyncAt?: string;
  lastError?: string;
};
