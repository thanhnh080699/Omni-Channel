"use client";

import { FormEvent, useEffect, useState } from "react";
import { Lock, Save, User } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { FormField } from "@/components/form-field";
import { Modal } from "@/components/modal";

export function ProfileModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { token, profile, refreshProfile } = useAuth();
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (profile?.user && open) {
      setDisplayName(profile.user.display_name || "");
      setEmail(profile.user.email || "");
      setMessage("");
      setError("");
    }
  }, [profile?.user, open]);

  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    setError("");
    setMessage("");
    try {
      await api.updateProfile(token, { display_name: displayName, email });
      await refreshProfile();
      setMessage("Profile updated.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update profile");
    }
  }

  async function changePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    if (newPassword !== confirmPassword) {
      setError("Password confirmation does not match.");
      return;
    }
    setError("");
    setMessage("");
    try {
      await api.changePassword(token, { current_password: currentPassword, new_password: newPassword });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setMessage("Password changed.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not change password");
    }
  }

  return (
    <Modal title="Profile settings" open={open} onClose={onClose}>
      <div className="grid gap-6">
        <form className="space-y-3" onSubmit={saveProfile}>
          <div className="flex items-center gap-2 border-b pb-2" style={{ borderColor: "var(--app-border)" }}>
            <User size={18} className="text-sky-600" />
            <h3 className="font-semibold">Personal information</h3>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField label="Display name">
              <input className="field" value={displayName} onChange={(event) => setDisplayName(event.target.value)} required />
            </FormField>
            <FormField label="Email">
              <input className="field" type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
            </FormField>
          </div>
          <div className="flex justify-end">
            <button className="btn btn-primary">
              <Save size={16} /> Save changes
            </button>
          </div>
        </form>

        <form className="space-y-3" onSubmit={changePassword}>
          <div className="flex items-center gap-2 border-b pb-2" style={{ borderColor: "var(--app-border)" }}>
            <Lock size={18} className="text-sky-600" />
            <h3 className="font-semibold">Change password</h3>
          </div>
          <FormField label="Current password">
            <input className="field" type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} required />
          </FormField>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField label="New password">
              <input className="field" type="password" minLength={8} value={newPassword} onChange={(event) => setNewPassword(event.target.value)} required />
            </FormField>
            <FormField label="Confirm new password">
              <input className="field" type="password" minLength={8} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} required />
            </FormField>
          </div>
          <div className="flex justify-end">
            <button className="btn">
              <Lock size={16} /> Change password
            </button>
          </div>
        </form>

        {message ? <div className="rounded-md bg-emerald-50 p-3 text-sm text-emerald-700">{message}</div> : null}
        {error ? <div className="rounded-md bg-red-50 p-3 text-sm text-danger">{error}</div> : null}
      </div>
    </Modal>
  );
}
