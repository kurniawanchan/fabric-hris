import { Badge } from "@/components/ui/badge";

/**
 * DESIGN.md's mode-indicator component: a persistent, always-visible pill
 * reading "Live" or "Idle" only -- no third state. The recorded fallback is a
 * separate video played entirely outside this product (EXPERIENCE.md's State
 * Patterns) -- this component has no way to represent it and must not grow
 * one.
 */
export function ModeIndicator({ isLive }: { isLive: boolean }) {
  return (
    <Badge
      variant={isLive ? "default" : "secondary"}
      className={isLive ? "bg-status-healthy text-status-healthy-foreground" : undefined}
    >
      {isLive ? "Live" : "Idle"}
    </Badge>
  );
}
