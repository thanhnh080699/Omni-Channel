import clsx from "clsx";

export function StatusPill({ value }: { value: string | boolean }) {
  const text = typeof value === "boolean" ? (value ? "enabled" : "disabled") : value || "unknown";
  const normalized = text
    .toLowerCase()
    .replace(/đ/g, "d")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "");
  const success = ["active", "enabled", "open", "healthy", "connected", "da ket noi"].includes(normalized);
  const warning = ["connecting", "qr", "pending", "cho quet qr", "dang ket noi", "dang cho", "not_configured", "cho cau hinh"].includes(normalized);
  const danger = ["disabled", "failed", "blocked", "error", "loi"].includes(normalized);
  return (
    <span
      className={clsx(
        "status-pill",
        success && "border-emerald-200 bg-[var(--app-success-soft-bg)] text-[var(--app-success-soft-fg)]",
        danger && "border-red-200 bg-[var(--app-danger-soft-bg)] text-[var(--app-danger-soft-fg)]",
        warning && "border-amber-200 bg-[var(--app-warning-soft-bg)] text-[var(--app-warning-soft-fg)]",
        !success && !danger && !warning && "border-slate-200 bg-[var(--app-surface-muted)] text-[var(--app-muted-strong)]",
      )}
    >
      {text}
    </span>
  );
}
