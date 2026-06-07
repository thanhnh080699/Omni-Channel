import type { SessionSnapshot } from "./contracts.js";

export class SessionStore {
  private readonly sessions = new Map<string, SessionSnapshot>();
  private readonly seenEvents = new Set<string>();

  snapshot(accountId: string): SessionSnapshot {
    return this.sessions.get(accountId) || { accountId, status: "disconnected" };
  }

  set(accountId: string, patch: Partial<SessionSnapshot>) {
    const current = this.snapshot(accountId);
    const next = { ...current, ...patch, accountId };
    this.sessions.set(accountId, next);
    return next;
  }

  firstSeen(idempotencyKey: string) {
    if (this.seenEvents.has(idempotencyKey)) return false;
    this.seenEvents.add(idempotencyKey);
    return true;
  }

  clearSession(accountId: string) {
    this.sessions.delete(accountId);
    for (const key of [...this.seenEvents]) {
      if (key.startsWith(`${accountId}:`)) this.seenEvents.delete(key);
    }
  }
}
