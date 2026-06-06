"use client";

import { X } from "lucide-react";

export function Modal({
  title,
  open,
  onClose,
  children,
}: {
  title: string;
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
}) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: "var(--app-overlay)" }}>
      <div className="menu-motion max-h-[90vh] w-full max-w-2xl overflow-hidden rounded-md shadow-xl" style={{ background: "var(--app-surface)" }}>
        <div className="flex h-14 items-center justify-between border-b px-4" style={{ borderColor: "var(--app-border)" }}>
          <h2 className="text-sm font-semibold">{title}</h2>
          <button className="btn h-8 w-8 px-0" onClick={onClose} title="Close">
            <X size={16} />
          </button>
        </div>
        <div className="max-h-[calc(90vh-56px)] overflow-y-auto p-4">{children}</div>
      </div>
    </div>
  );
}
