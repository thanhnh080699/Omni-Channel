"use client";

import { Search } from "lucide-react";
import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { DataTable } from "@/components/data-table";
import { PageHeader } from "@/components/page-header";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { AuditLog } from "@/lib/types";

export default function AuditPage() {
  const { token } = useAuth();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [filters, setFilters] = useState({ action: "", resource_type: "", actor_user_id: "" });

  async function load() {
    if (!token) return;
    const result = await api.auditLogs(token, filters);
    setLogs(result.data || []);
  }

  useEffect(() => {
    void load();
  }, [token]);

  return (
    <AdminShell>
      <PageHeader title="Audit" description="Review administrative actions and security-relevant changes." />
      <div className="mb-3 grid gap-2 sm:grid-cols-[1fr_1fr_1fr_auto]">
        <input
          className="field"
          placeholder="Action"
          value={filters.action}
          onChange={(event) => setFilters({ ...filters, action: event.target.value })}
        />
        <input
          className="field"
          placeholder="Resource type"
          value={filters.resource_type}
          onChange={(event) => setFilters({ ...filters, resource_type: event.target.value })}
        />
        <input
          className="field"
          placeholder="Actor user ID"
          value={filters.actor_user_id}
          onChange={(event) => setFilters({ ...filters, actor_user_id: event.target.value })}
        />
        <button className="btn" onClick={load} title="Search">
          <Search size={16} />
          Search
        </button>
      </div>
      <DataTable
        data={logs}
        empty="No audit logs found."
        columns={[
          {
            key: "action",
            label: "Action",
            render: (row) => (
              <div>
                <div className="font-medium text-ink">{row.action}</div>
                <div className="text-xs text-muted">{new Date(row.created_at).toLocaleString()}</div>
              </div>
            ),
          },
          {
            key: "resource",
            label: "Resource",
            render: (row) => (
              <div className="text-sm text-muted">
                {row.resource_type}:{row.resource_id}
              </div>
            ),
          },
          { key: "actor", label: "Actor", render: (row) => <div className="text-sm text-muted">{row.actor_user_id || "-"}</div> },
          { key: "ip", label: "IP", render: (row) => <div className="text-sm text-muted">{row.ip || "-"}</div> },
          {
            key: "metadata",
            label: "Metadata",
            render: (row) => (
              <pre className="max-w-lg overflow-hidden text-ellipsis whitespace-nowrap text-xs text-muted">
                {row.metadata ? JSON.stringify(row.metadata) : "-"}
              </pre>
            ),
          },
        ]}
      />
    </AdminShell>
  );
}
