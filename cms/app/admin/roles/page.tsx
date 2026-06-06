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
import type { Permission, Role } from "@/lib/types";

const emptyRole = { name: "", code: "", permission_codes: [] as string[] };

export default function RolesPage() {
  const { token } = useAuth();
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [editing, setEditing] = useState<Role | null>(null);
  const [form, setForm] = useState(emptyRole);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    if (!token) return;
    const matrix = await api.permissionMatrix(token);
    setRoles(matrix.roles || []);
    setPermissions(matrix.permissions || []);
  }

  useEffect(() => {
    void load();
  }, [token]);

  function edit(role?: Role) {
    setError("");
    setEditing(role || null);
    setForm(role ? { name: role.name, code: role.code, permission_codes: role.permission_codes || [] } : emptyRole);
    setOpen(true);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    if (editing?.is_system) {
      setError("System roles cannot be edited.");
      return;
    }
    try {
      if (editing) {
        await api.updateRole(token, editing.id, form);
      } else {
        await api.createRole(token, form);
      }
      setOpen(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save role");
    }
  }

  return (
    <AdminShell>
      <PageHeader
        title="Roles"
        description="Define permission bundles used by the backend RBAC checks."
        actions={
          <button className="btn btn-primary" onClick={() => edit()}>
            <Plus size={16} /> New role
          </button>
        }
      />
      <DataTable
        data={roles}
        empty="No roles found."
        columns={[
          {
            key: "role",
            label: "Role",
            render: (row) => (
              <div>
                <div className="font-medium text-ink">{row.name}</div>
                <div className="text-xs text-muted">{row.code}</div>
              </div>
            ),
          },
          { key: "system", label: "Type", render: (row) => <StatusPill value={row.is_system ? "system" : "custom"} /> },
          {
            key: "permissions",
            label: "Permissions",
            render: (row) => <div className="text-sm text-muted">{row.permission_codes.length} permissions</div>,
          },
          {
            key: "actions",
            label: "",
            className: "w-28",
            render: (row) => (
              <button className="btn h-8" onClick={() => edit(row)}>
                {row.is_system ? "View" : "Edit"}
              </button>
            ),
          },
        ]}
      />
      <div className="panel mt-4 p-4">
        <div className="mb-3 text-sm font-semibold text-ink">Permission matrix</div>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
          {permissions.map((permission) => (
            <div key={permission.id} className="rounded-md border border-line p-3">
              <div className="text-sm font-medium text-ink">{permission.code}</div>
              <div className="mt-1 text-xs text-muted">{permission.description}</div>
            </div>
          ))}
        </div>
      </div>
      <Modal title={editing ? "Role details" : "Create role"} open={open} onClose={() => setOpen(false)}>
        <form className="space-y-3" onSubmit={save}>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField label="Name">
              <input
                className="field"
                value={form.name}
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                disabled={editing?.is_system}
                required
              />
            </FormField>
            <FormField label="Code">
              <input
                className="field"
                value={form.code}
                onChange={(event) => setForm({ ...form, code: event.target.value })}
                disabled={editing?.is_system}
                required
              />
            </FormField>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            {permissions.map((permission) => (
              <label key={permission.code} className="flex items-start gap-2 rounded-md border border-line p-3 text-sm">
                <input
                  className="mt-1"
                  type="checkbox"
                  checked={form.permission_codes.includes(permission.code)}
                  disabled={editing?.is_system}
                  onChange={(event) => {
                    const next = event.target.checked
                      ? [...form.permission_codes, permission.code]
                      : form.permission_codes.filter((code) => code !== permission.code);
                    setForm({ ...form, permission_codes: next });
                  }}
                />
                <span>
                  <span className="block font-medium text-ink">{permission.code}</span>
                  <span className="block text-xs text-muted">{permission.description}</span>
                </span>
              </label>
            ))}
          </div>
          {error ? <div className="rounded-md bg-red-50 p-3 text-sm text-danger">{error}</div> : null}
          <div className="flex justify-end gap-2">
            <button className="btn" type="button" onClick={() => setOpen(false)}>
              Close
            </button>
            {!editing?.is_system ? <button className="btn btn-primary">Save</button> : null}
          </div>
        </form>
      </Modal>
    </AdminShell>
  );
}
