export function FormField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium uppercase tracking-normal" style={{ color: "var(--app-muted)" }}>{label}</span>
      {children}
    </label>
  );
}
