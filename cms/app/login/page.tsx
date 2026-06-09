"use client";

import { LockKeyhole, Loader2 } from "lucide-react";
import { FormEvent, useState } from "react";
import { useAuth } from "@/lib/auth";

export default function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("admin123456");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      await login(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen bg-shell">
      <section className="hidden flex-1 border-r border-line bg-white p-10 lg:block">
        <div className="flex h-full max-w-xl flex-col justify-between">
          <div>
            <div className="mb-8 flex h-11 w-11 items-center justify-center rounded-md bg-blue-50 text-accent">
              <LockKeyhole size={22} />
            </div>
            <h1 className="text-3xl font-semibold text-ink">Omni Channel CMS</h1>
            <p className="mt-3 text-base leading-7 text-muted">
              Centralized administration for users, teams, permissions, and channel operations.
            </p>
          </div>
          <div className="panel p-4 text-sm text-muted">
            Admin flows are protected by backend RBAC. Use seeded credentials for local development.
          </div>
        </div>
      </section>
      <section className="flex flex-1 items-center justify-center p-4">
        <form className="panel w-full max-w-md p-5" onSubmit={onSubmit}>
          <div className="mb-5">
            <h2 className="text-xl font-semibold text-ink">Sign in</h2>
            <p className="mt-1 text-sm text-muted">Access the administration workspace.</p>
          </div>
          <div className="space-y-3">
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-ink">Email</span>
              <input className="field" value={email} onChange={(event) => setEmail(event.target.value)} type="email" />
            </label>
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-ink">Password</span>
              <input
                className="field"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                type="password"
              />
            </label>
          </div>
          {error ? (
            <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-danger flex items-center gap-2">
              {error.includes("Mất kết nối") && <Loader2 className="h-4 w-4 animate-spin text-danger" />}
              <span>{error}</span>
            </div>
          ) : null}
          <button className="btn btn-primary mt-5 w-full" disabled={submitting}>
            {submitting ? "Signing in..." : "Sign in"}
          </button>
        </form>
      </section>
    </main>
  );
}
