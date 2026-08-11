import { useEffect, useRef, useState } from "react";
import { useLogLines } from "@/hooks/useLogLines";

const NEAR_BOTTOM_PX = 24;

/**
 * DESIGN.md's log-line component + EXPERIENCE.md's auto-scroll convention:
 * new lines append at the bottom and auto-scroll, unless the operator has
 * manually scrolled up to read history -- in which case auto-scroll pauses
 * until they scroll back to bottom.
 */
export function Notifications() {
  const { lines, logStale } = useLogLines();
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  useEffect(() => {
    if (!autoScroll || !containerRef.current) return;
    containerRef.current.scrollTop = containerRef.current.scrollHeight;
  }, [lines, autoScroll]);

  function handleScroll() {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= NEAR_BOTTOM_PX;
    setAutoScroll(atBottom);
  }

  if (lines.length === 0) {
    return <p className="text-muted-foreground">No output yet.</p>;
  }

  return (
    <div>
      {logStale && (
        <p className="mb-2 font-mono text-[12px] tracking-wide text-muted-foreground uppercase">
          Stale — no output recently
        </p>
      )}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className={`h-96 overflow-y-auto rounded-lg border bg-card p-3 font-mono text-[13px] leading-relaxed transition-opacity ${logStale ? "opacity-50" : "opacity-100"}`}
      >
        {lines.map((entry, i) => (
          <div key={i} className="whitespace-pre-wrap">
            {entry.line}
          </div>
        ))}
      </div>
    </div>
  );
}
