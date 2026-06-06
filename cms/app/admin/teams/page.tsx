"use client";

import { Plus } from "lucide-react";
import { FormEvent, useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { DataTable } from "@/components/data-table";
import { FormField } from "@/components/form-field";
import { Modal } from "@/components/modal";
import { PageHeader } from "@/components/page-header";
import { StatusPill } from "@/components/status-pill";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Team, User } from "@/lib/types";

const emptyTeam = { name: "", parent_team_id: "", manager_user_ids: [] as string[], status: "active" };

export default function TeamsPage() {
  const { token } = useAuth();
  const [teams, setTeams] = useState<Team[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [editing, setEditing] = useState<Team | null>(null);
  const [form, setForm] = useState(emptyTeam);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    if (!token) return;
    const [teamResult, userResult] = await Promise.all([api.teams(token), api.users(token)]);
    setTeams(teamResult.data || []);
    setUsers(userResult.data || []);
  }

  useEffect(() => {
    void load();
  }, [token]);

  function edit(team?: Team) {
    setError("");
    setEditing(team || null);
    setForm(
      team
        ? {
            name: team.name,
            parent_team_id: team.parent_team_id || "",
            manager_user_ids: team.manager_user_ids || [],
            status: team.status,
          }
        : emptyTeam,
    );
    setOpen(true);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    try {
      if (editing) {
        await api.updateTeam(token, editing.id, form);
      } else {
        await api.createTeam(token, form);
      }
      setOpen(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save team");
    }
  }

  async function disable(id: string) {
    if (!token) return;
    await api.deleteTeam(token, id);
    await load();
  }

  return (
    <AdminShell>
      <PageHeader
        title="Teams"
        description="Organize staff visibility and manager access boundaries."
        actions={
          <button className="btn btn-primary" onClick={() => edit()}>
            <Plus size={16} /> New team
          </button>
        }
      />
      <DataTable
        data={teams}
        empty="No teams found."
        columns={[
          { key: "name", label: "Team", render: (row) => <div className="font-medium text-ink">{row.name}</div> },
          { key: "status", label: "Status", render: (row) => <StatusPill value={row.status} /> },
          {
            key: "managers",
            label: "Managers",
            render: (row) => (
              <div className="text-sm text-muted">
                {row.manager_user_ids.map((id) => users.find((user) => user.id === id)?.display_name || id).join(", ") || "-"}
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
      <Modal title={editing ? "Edit team" : "Create team"} open={open} onClose={() => setOpen(false)}>
        <form className="space-y-3" onSubmit={save}>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField label="Name">
              <input className="field" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
            </FormField>
            <FormField label="Status">
              <select className="field" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
            </FormField>
          </div>
          <FormField label="Parent team">
            <select
              className="field"
              value={form.parent_team_id}
              onChange={(event) => setForm({ ...form, parent_team_id: event.target.value })}
            >
              <option value="">No parent</option>
              {teams
                .filter((team) => team.id !== editing?.id)
                .map((team) => (
                  <option key={team.id} value={team.id}>
                    {team.name}
                  </option>
                ))}
            </select>
          </FormField>
          <FormField label="Managers">
            <select
              className="field h-32"
              multiple
              value={form.manager_user_ids}
              onChange={(event) =>
                setForm({ ...form, manager_user_ids: Array.from(event.target.selectedOptions).map((option) => option.value) })
              }
            >
              {users.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.display_name} ({user.email})
                </option>
              ))}
            </select>
          </FormField>
          {error ? <div className="rounded-md bg-red-50 p-3 text-sm text-danger">{error}</div> : null}
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
