import assert from "node:assert/strict";
import test from "node:test";
import { SessionStore } from "../src/sessionStore.js";

test("tracks QR and connected session state", () => {
  const store = new SessionStore();
  assert.equal(store.snapshot("ca_1").status, "disconnected");
  store.set("ca_1", { status: "qr", qr: "qr-token" });
  assert.equal(store.snapshot("ca_1").qr, "qr-token");
  store.set("ca_1", { status: "connected", qr: undefined });
  assert.equal(store.snapshot("ca_1").status, "connected");
});

test("deduplicates resync and realtime events by idempotency key", () => {
  const store = new SessionStore();
  assert.equal(store.firstSeen("ca_1:wamid_1"), true);
  assert.equal(store.firstSeen("ca_1:wamid_1"), false);
});

test("clears session state and account-scoped dedupe keys", () => {
  const store = new SessionStore();
  store.set("ca_1", { status: "connected" });
  assert.equal(store.firstSeen("ca_1:wamid_1"), true);
  store.clearSession("ca_1");
  assert.equal(store.snapshot("ca_1").status, "disconnected");
  assert.equal(store.firstSeen("ca_1:wamid_1"), true);
});

test("waits until a QR snapshot is available", async () => {
  const store = new SessionStore();
  const promise = store.waitFor("ca_1", (snapshot) => Boolean(snapshot.qr), 1000);

  store.set("ca_1", { status: "qr", qr: "qr-token" });

  const snapshot = await promise;
  assert.equal(snapshot.qr, "qr-token");
});

test("does not return expired QR snapshots", () => {
  const store = new SessionStore();
  store.set("ca_1", {
    status: "qr",
    qr: "qr-token",
    qrExpiresAt: new Date(Date.now() - 1000).toISOString(),
  });

  const snapshot = store.snapshot("ca_1");

  assert.equal(snapshot.status, "connecting");
  assert.equal(snapshot.qr, undefined);
});

test("tracks typing snapshots with expiry", () => {
  const store = new SessionStore();
  store.setTyping("ca_1", "84900000000@s.whatsapp.net", true, 1000);
  assert.equal(store.typingSnapshot("ca_1", "84900000000@s.whatsapp.net").typing, true);
  store.setTyping("ca_1", "84900000000@s.whatsapp.net", false);
  assert.equal(store.typingSnapshot("ca_1", "84900000000@s.whatsapp.net").typing, false);
});
