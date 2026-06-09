"use client";

import { Plus, Search, Loader2 } from "lucide-react";
import { FormEvent, useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { DataTable } from "@/components/data-table";
import { FormField } from "@/components/form-field";
import { Modal } from "@/components/modal";
import { PageHeader } from "@/components/page-header";
import { StatusPill } from "@/components/status-pill";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Role, Team, User } from "@/lib/types";

const emptyUser = {
  email: "",
  password: "",
  display_name: "",
  status: "active",
  role_ids: [] as string[],
  team_ids: [] as string[],
};

export default function UsersPage() {
  const { token } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState(emptyUser);
  const [error, setError] = useState("");

  async function load() {
    if (!token) return;
    const [userResult, roleResult, teamResult] = await Promise.all([
      api.users(token, { q }),
      api.roles(token),
      api.teams(token),
    ]);
    setUsers(userResult.data || []);
    setRoles(roleResult.data || []);
    setTeams(teamResult.data || []);
  }

  useEffect(() => {
    void load();
  }, [token]);

  function edit(user?: User) {
    setError("");
    setEditing(user || null);
    setForm(
      user
        ? {
            email: user.email,
            password: "",
            display_name: user.display_name,
            status: user.status,
            role_ids: user.role_ids || [],
            team_ids: user.team_ids || [],
          }
        : emptyUser,
    );
    setOpen(true);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    setError("");
    try {
      if (editing) {
        const body = { ...form, password: form.password || undefined };
        await api.updateUser(token, editing.id, body);
      } else {
        await api.createUser(token, form);
      }
      setOpen(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save user");
    }
  }

  async function disable(id: string) {
    if (!token) return;
    await api.deleteUser(token, id);
    await load();
  }

  return (
    <AdminShell>
      <PageHeader
        title="Users"
        description="Manage agent accounts, roles, and team membership."
        actions={
          <button className="btn btn-primary" onClick={() => edit()}>
            <Plus size={16} /> New user
          </button>
        }
      />
      <div className="mb-3 flex max-w-md gap-2">
        <input className="field" placeholder="Search users" value={q} onChange={(event) => setQ(event.target.value)} />
        <button className="btn" onClick={load} title="Search">
          <Search size={16} />
        </button>
      </div>
      <DataTable
        data={users}
        empty="No users found."
        columns={[
          {
            key: "user",
            label: "User",
            render: (row) => (
              <div>
                <div className="font-medium text-ink">{row.display_name}</div>
                <div className="text-xs text-muted">{row.email}</div>
              </div>
            ),
          },
          { key: "status", label: "Status", render: (row) => <StatusPill value={row.status} /> },
          {
            key: "roles",
            label: "Roles",
            render: (row) => (
              <div className="text-sm text-muted">
                {row.role_ids.map((id) => roles.find((role) => role.id === id)?.name || id).join(", ") || "-"}
              </div>
            ),
          },
          {
            key: "teams",
            label: "Teams",
            render: (row) => (
              <div className="text-sm text-muted">
                {row.team_ids.map((id) => teams.find((team) => team.id === id)?.name || id).join(", ") || "-"}
              </div>
            ),
          },
          {
            key: "actions",
            label: "",
            className: "w-44",
            render: (row) => (
              <div className="flex justify-end gap-2">
                <button className="btn h-8" onClick={() => edit(row)}>
                  Edit
                </button>
                <button className="btn h-8" onClick={() => disable(row.id)}>
                  Disable
                </button>
              </div>
            ),
          },
        ]}
      />
      <Modal title={editing ? "Edit user" : "Create user"} open={open} onClose={() => setOpen(false)}>
        <form className="space-y-3" onSubmit={save}>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField label="Display name">
              <input
                className="field"
                value={form.display_name}
                onChange={(event) => setForm({ ...form, display_name: event.target.value })}
                required
              />
            </FormField>
            <FormField label="Email">
              <input
                className="field"
                value={form.email}
                onChange={(event) => setForm({ ...form, email: event.target.value })}
                type="email"
                required
                disabled={Boolean(editing)}
              />
            </FormField>
            <FormField label="Password">
              <input
                className="field"
                value={form.password}
                onChange={(event) => setForm({ ...form, password: event.target.value })}
                type="password"
                required={!editing}
                minLength={editing ? undefined : 8}
                placeholder={editing ? "Leave blank to keep current password" : ""}
              />
            </FormField>
            <FormField label="Status">
              <select className="field" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
            </FormField>
          </div>
          <FormField label="Roles">
            <select
              className="field h-28"
              multiple
              value={form.role_ids}
              onChange={(event) =>
                setForm({ ...form, role_ids: Array.from(event.target.selectedOptions).map((option) => option.value) })
              }
            >
              {roles.map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Teams">
            <select
              className="field h-28"
              multiple
              value={form.team_ids}
              onChange={(event) =>
                setForm({ ...form, team_ids: Array.from(event.target.selectedOptions).map((option) => option.value) })
              }
            >
              {teams.map((team) => (
                <option key={team.id} value={team.id}>
                  {team.name}
                </option>
              ))}
            </select>
          </FormField>
          {error ? (
            <div className="rounded-md bg-red-50 p-3 text-sm text-danger flex items-center gap-2">
              {error.includes("Mất kết nối") && <Loader2 className="h-4 w-4 animate-spin text-danger" />}
              <span>{error}</span>
            </div>
          ) : null}
          <div className="flex justify-end gap-2">
            <button className="btn" type="button" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button className="btn btn-primary">Save</button>
          </div>
        </form>
      </Modal>
    </AdminShell>
  );
}
