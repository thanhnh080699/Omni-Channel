"use client";

import { useCallback, useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Conversation } from "@/lib/types";
import { Loader2, RotateCcw, Trash2 } from "lucide-react";

export default function TrashPage() {
  const { token } = useAuth();
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [actionId, setActionId] = useState("");

  const loadTrash = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const result = await api.trashConversations(token);
      setConversations(result.data || []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load trash conversations");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void loadTrash();
  }, [loadTrash]);

  async function handleRestore(id: string) {
    if (!token) return;
    setActionId(id);
    try {
      await api.restoreConversation(token, id);
      setConversations((current) => current.filter((c) => c.id !== id));
    } catch (err) {
      alert(err instanceof Error ? err.message : "Could not restore conversation");
    } finally {
      setActionId("");
    }
  }

  function displayCustomer(conversation: Conversation) {
    return conversation.customer_name || conversation.customer_ref || conversation.external_conversation_id || "Khách hàng";
  }

  function formatDate(value?: string) {
    if (!value) return "-";
    return new Intl.DateTimeFormat("vi-VN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
  }

  return (
    <AdminShell>
      <div className="space-y-6">
        <div className="flex items-center justify-between border-b pb-4" style={{ borderColor: "var(--app-border)" }}>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Thùng rác cuộc hội thoại</h1>
            <p className="text-sm mt-1" style={{ color: "var(--app-muted)" }}>
              Danh sách các cuộc hội thoại đã xoá mềm. Sau 30 ngày kể từ ngày xoá, các cuộc trò chuyện này sẽ tự động bị xoá vĩnh viễn.
            </p>
          </div>
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-[var(--app-surface-muted)] text-[var(--app-muted-strong)]">
            <Trash2 size={20} />
          </div>
        </div>

        {error && (
          <div className="rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        {loading ? (
          <div className="flex h-64 items-center justify-center">
            <Loader2 className="h-8 w-8 animate-spin" style={{ color: "var(--app-muted)" }} />
          </div>
        ) : conversations.length === 0 ? (
          <div className="rounded-lg border border-dashed py-24 text-center" style={{ borderColor: "var(--app-border)" }}>
            <Trash2 className="mx-auto h-12 w-12 text-[var(--app-muted)] opacity-50" />
            <h3 className="mt-4 text-sm font-semibold">Thùng rác trống</h3>
            <p className="mt-1 text-xs" style={{ color: "var(--app-muted)" }}>Không có cuộc hội thoại nào bị xoá.</p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border shadow-sm" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-left text-sm">
                <thead>
                  <tr className="border-b bg-[var(--app-surface-muted)] text-xs font-bold uppercase tracking-wider" style={{ borderColor: "var(--app-border)", color: "var(--app-muted-strong)" }}>
                    <th className="px-6 py-3.5">Khách hàng</th>
                    <th className="px-6 py-3.5">External ID</th>
                    <th className="px-6 py-3.5">Ngày xoá</th>
                    <th className="px-6 py-3.5 text-right">Thao tác</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--app-border)" }}>
                  {conversations.map((conv) => (
                    <tr key={conv.id} className="hover:bg-[var(--app-surface-hover)] transition-colors">
                      <td className="px-6 py-4">
                        <div className="font-semibold" style={{ color: "var(--app-text)" }}>
                          {displayCustomer(conv)}
                        </div>
                        <div className="text-xs" style={{ color: "var(--app-muted)" }}>
                          Kênh: WA · ID: {conv.channel_account_id.slice(0, 8)}
                        </div>
                      </td>
                      <td className="px-6 py-4 font-mono text-xs" style={{ color: "var(--app-muted-strong)" }}>
                        {conv.external_conversation_id}
                      </td>
                      <td className="px-6 py-4 text-xs" style={{ color: "var(--app-muted)" }}>
                        {formatDate(conv.deleted_at)}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          type="button"
                          disabled={actionId === conv.id}
                          onClick={() => void handleRestore(conv.id)}
                          className="inline-flex items-center gap-1.5 rounded-md border border-sky-200 px-3 py-1.5 text-xs font-semibold text-sky-600 hover:bg-sky-50 transition"
                        >
                          {actionId === conv.id ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <RotateCcw className="h-3.5 w-3.5" />
                          )}
                          Khôi phục
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </AdminShell>
  );
}
