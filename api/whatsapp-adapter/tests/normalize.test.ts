import assert from "node:assert/strict";
import test from "node:test";
import { buildInboundEnvelope, normalizeBaileysMessage, outboundToBaileys } from "../src/normalize.js";

test("normalizes Baileys text message", () => {
  const normalized = normalizeBaileysMessage("ca_1", {
    key: { id: "wamid_1", remoteJid: "84900000000@s.whatsapp.net", fromMe: false },
    messageTimestamp: 1780717200,
    pushName: "Customer",
    message: { conversation: "hello" },
  });
  assert.equal(normalized?.channel, "whatsapp");
  assert.equal(normalized?.external_message_id, "wamid_1");
  assert.equal(normalized?.external_conversation_id, "84900000000@s.whatsapp.net");
  assert.equal(normalized?.text, "hello");
  assert.equal(normalized?.event_time, "2026-06-06T03:40:00.000Z");
});

test("builds inbound envelope with idempotency key", () => {
  const normalized = normalizeBaileysMessage("ca_1", {
    key: { id: "wamid_2", remoteJid: "84900000000@s.whatsapp.net", fromMe: false },
    messageTimestamp: 1780717200,
    message: { conversation: "hello" },
  });
  assert.ok(normalized);
  const envelope = buildInboundEnvelope("ca_1", normalized, new Date("2026-06-06T01:00:01Z"));
  assert.equal(envelope.idempotency_key, "ca_1:wamid_2");
  assert.equal(envelope.queued_at, "2026-06-06T01:00:01.000Z");
});

test("maps outbound payload to Baileys sendMessage arguments", () => {
  const mapped = outboundToBaileys({ external_conversation_id: "84900000000@s.whatsapp.net", text: "reply" });
  assert.deepEqual(mapped, { jid: "84900000000@s.whatsapp.net", content: { text: "reply" } });
});
