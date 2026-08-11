import { useEffect, useState } from "react";
import { fetchConfig } from "@/lib/api";

interface ConfigViewProps {
  scenario: string;
}

/**
 * FR8/AD-4: the active scenario's benchconfig YAML, byte-for-byte, as
 * plain non-editable text -- proof the demo isn't rigged (UX-DR6). A <pre>
 * is used rather than any input/textarea: it carries no edit affordance by
 * construction -- no cursor, no focus ring -- rather than relying on a
 * readOnly attribute that a live edit could bypass.
 */
export function ConfigView({ scenario }: ConfigViewProps) {
  const [yaml, setYaml] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setYaml(null);
    setError(null);
    void fetchConfig()
      .then((text) => {
        if (!cancelled) setYaml(text);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [scenario]);

  if (error) {
    return <p className="text-status-down">{error}</p>;
  }

  if (yaml === null) {
    return <p className="text-muted-foreground">Loading configuration…</p>;
  }

  return (
    <pre className="max-h-[32rem] cursor-default overflow-auto rounded-lg border bg-card p-4 font-mono text-[13px] leading-relaxed whitespace-pre">
      {yaml}
    </pre>
  );
}
