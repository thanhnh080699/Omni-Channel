"use client";

import { Activity, Cable, ShieldCheck, Users } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { PageHeader } from "@/components/page-header";
import { useAuth } from "@/lib/auth";

const cards = [
  { label: "User administration", value: "Users", icon: Users },
  { label: "Permission matrix", value: "Roles", icon: ShieldCheck },
  { label: "Channel operations", value: "Channels", icon: Cable },
  { label: "Audit trail", value: "Audit", icon: Activity },
];

export default function DashboardPage() {
  const { profile } = useAuth();
  return (
    <AdminShell>
      <PageHeader title="Dashboard" description="Operational snapshot for the administration workspace." />
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => {
          const Icon = card.icon;
          return (
            <div key={card.label} className="panel p-4">
              <div className="mb-4 flex h-9 w-9 items-center justify-center rounded-md bg-blue-50 text-accent">
                <Icon size={18} />
              </div>
              <div className="text-2xl font-semibold text-ink">{card.value}</div>
              <div className="mt-1 text-sm text-muted">{card.label}</div>
            </div>
          );
        })}
      </div>
      <div className="panel mt-4 p-4">
        <div className="text-sm font-semibold text-ink">Current session</div>
        <div className="mt-3 grid gap-3 text-sm sm:grid-cols-3">
          <div>
            <div className="text-xs text-muted">User</div>
            <div className="font-medium text-ink">{profile?.user.display_name}</div>
          </div>
          <div>
            <div className="text-xs text-muted">Email</div>
            <div className="font-medium text-ink">{profile?.user.email}</div>
          </div>
          <div>
            <div className="text-xs text-muted">Permissions</div>
            <div className="font-medium text-ink">{Object.keys(profile?.permissions || {}).length}</div>
          </div>
        </div>
      </div>
    </AdminShell>
  );
}
