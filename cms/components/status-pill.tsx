import clsx from "clsx";

export function StatusPill({ value }: { value: string | boolean }) {
  const text = typeof value === "boolean" ? (value ? "enabled" : "disabled") : value || "unknown";
  const normalized = text.toLowerCase();
  return (
    <span
      className={clsx(
        "status-pill",
        ["active", "enabled", "open", "healthy"].includes(normalized) && "border-emerald-200 bg-[var(--app-success-soft-bg)] text-[var(--app-success-soft-fg)]",
        ["disabled", "failed", "blocked"].includes(normalized) && "border-red-200 bg-[var(--app-danger-soft-bg)] text-[var(--app-danger-soft-fg)]",
        !["active", "enabled", "open", "healthy", "disabled", "failed", "blocked"].includes(normalized) &&
          "border-amber-200 bg-[var(--app-warning-soft-bg)] text-[var(--app-warning-soft-fg)]",
      )}
    >
      {text}
    </span>
  );
}
