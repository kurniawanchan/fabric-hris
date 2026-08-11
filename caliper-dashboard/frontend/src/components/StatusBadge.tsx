import type { NodeStatusEntry } from "@/hooks/useNodeStatus";

const LABELS: Record<NodeStatusEntry["status"], string> = {
  healthy: "Healthy",
  slow: "Slow",
  unreachable: "Unreachable",
};

const COLORS: Record<NodeStatusEntry["status"], string> = {
  healthy: "bg-status-healthy text-status-healthy-foreground",
  slow: "bg-status-warning text-status-warning-foreground",
  unreachable: "bg-status-down text-status-down-foreground",
};

const ICONS: Record<NodeStatusEntry["status"], string> = {
  healthy: "●",
  slow: "◐",
  unreachable: "✕",
};

/**
 * DESIGN.md's status-badge component. Always icon *and* color *and* text
 * label together -- never color alone (Accessibility Floor).
 */
export function StatusBadge({ status }: { status: NodeStatusEntry["status"] | undefined }) {
  if (!status) {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
        <span aria-hidden>…</span> Unknown
      </span>
    );
  }
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${COLORS[status]}`}
    >
      <span aria-hidden>{ICONS[status]}</span> {LABELS[status]}
    </span>
  );
}
