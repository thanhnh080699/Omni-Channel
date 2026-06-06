import clsx from "clsx";

type Column<T> = {
  key: string;
  label: string;
  className?: string;
  render: (row: T) => React.ReactNode;
};

export function DataTable<T>({ columns, data, empty }: { columns: Column<T>[]; data: T[]; empty: string }) {
  return (
    <div className="panel overflow-hidden">
      <div className="overflow-x-auto">
        <table className="min-w-full table-fixed divide-y divide-line text-left text-sm">
          <thead className="text-xs uppercase tracking-normal" style={{ background: "var(--app-surface-muted)", color: "var(--app-muted)" }}>
            <tr>
              {columns.map((column) => (
                <th key={column.key} className={clsx("px-4 py-3 font-semibold", column.className)}>
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
            {data.length === 0 ? (
              <tr>
                <td className="px-4 py-8 text-center text-sm" style={{ color: "var(--app-muted)" }} colSpan={columns.length}>
                  {empty}
                </td>
              </tr>
            ) : (
              data.map((row, index) => (
                <tr key={index} className="align-top transition-colors hover:bg-[var(--app-surface-hover)]">
                  {columns.map((column) => (
                    <td key={column.key} className={clsx("px-4 py-3", column.className)}>
                      {column.render(row)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
