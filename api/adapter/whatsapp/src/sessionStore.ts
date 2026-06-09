import type { SessionSnapshot, TypingSnapshot } from "./contracts.js";

export class SessionStore {
  private readonly sessions = new Map<string, SessionSnapshot>();
  private readonly typing = new Map<string, TypingSnapshot>();
  private readonly seenEvents = new Set<string>();
  private readonly waiters = new Map<string, Array<(snapshot: SessionSnapshot) => void>>();

  snapshot(accountId: string): SessionSnapshot {
    const current = this.sessions.get(accountId) || { accountId, status: "disconnected" as const };
    if (current.qr && current.qrExpiresAt && Date.now() > Date.parse(current.qrExpiresAt)) {
      const next = { ...current, status: "connecting" as const, qr: undefined };
      this.sessions.set(accountId, next);
      return next;
    }
    return current;
  }

  set(accountId: string, patch: Partial<SessionSnapshot>) {
    const current = this.snapshot(accountId);
    const next = { ...current, ...patch, accountId };
    this.sessions.set(accountId, next);
    for (const waiter of this.waiters.get(accountId) || []) waiter(next);
    return next;
  }

  waitFor(accountId: string, predicate: (snapshot: SessionSnapshot) => boolean, timeoutMs: number) {
    const current = this.snapshot(accountId);
    if (predicate(current)) return Promise.resolve(current);

    return new Promise<SessionSnapshot>((resolve) => {
      const timeout = setTimeout(() => {
        removeWaiter();
        resolve(this.snapshot(accountId));
      }, timeoutMs);
      const waiter = (snapshot: SessionSnapshot) => {
        if (!predicate(snapshot)) return;
        clearTimeout(timeout);
        removeWaiter();
        resolve(snapshot);
      };
      const removeWaiter = () => {
        const next = (this.waiters.get(accountId) || []).filter((item) => item !== waiter);
        if (next.length) this.waiters.set(accountId, next);
        else this.waiters.delete(accountId);
      };
      this.waiters.set(accountId, [...(this.waiters.get(accountId) || []), waiter]);
    });
  }

  firstSeen(idempotencyKey: string) {
    if (this.seenEvents.has(idempotencyKey)) return false;
    this.seenEvents.add(idempotencyKey);
    return true;
  }

  clearSession(accountId: string) {
    this.sessions.delete(accountId);
    for (const key of [...this.typing.keys()]) {
      if (key.startsWith(`${accountId}:`)) this.typing.delete(key);
    }
    for (const key of [...this.seenEvents]) {
      if (key.startsWith(`${accountId}:`)) this.seenEvents.delete(key);
    }
  }

  setTyping(accountId: string, jid: string, typing: boolean, ttlMs = 7000) {
    const now = new Date();
    const snapshot: TypingSnapshot = {
      accountId,
      jid,
      typing,
      updatedAt: now.toISOString(),
      expiresAt: typing ? new Date(now.getTime() + ttlMs).toISOString() : now.toISOString(),
    };
    this.typing.set(this.typingKey(accountId, jid), snapshot);
    return snapshot;
  }

  typingSnapshot(accountId: string, jid: string): TypingSnapshot {
    const key = this.typingKey(accountId, jid);
    const snapshot = this.typing.get(key);
    if (!snapshot) return { accountId, jid, typing: false };
    if (snapshot.expiresAt && Date.now() > Date.parse(snapshot.expiresAt)) {
      this.typing.delete(key);
      return { accountId, jid, typing: false, updatedAt: snapshot.updatedAt, expiresAt: snapshot.expiresAt };
    }
    return snapshot;
  }

  private typingKey(accountId: string, jid: string) {
    return `${accountId}:${jid}`;
  }
}
