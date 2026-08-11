interface MetricTileProps {
  label: string;
  value: number | null;
  formatValue?: (value: number) => string;
  stale?: boolean;
}

/**
 * DESIGN.md's metric-tile component. A value of zero renders as a literal
 * "0" -- never a blank field (FR2) -- and the tile visually mutes when its
 * source has gone stale, rather than silently showing an old number as
 * current.
 */
export function MetricTile({ label, value, formatValue, stale = false }: MetricTileProps) {
  const display = value === null ? "—" : (formatValue ?? String)(value);

  return (
    <div
      className={`rounded-lg border bg-card p-4 transition-opacity ${stale ? "opacity-50" : "opacity-100"}`}
    >
      <div className="font-mono text-[13px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </div>
      <div className="mt-1 text-[32px] leading-[1.1] font-semibold tabular-nums">{display}</div>
    </div>
  );
}
