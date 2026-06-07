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
