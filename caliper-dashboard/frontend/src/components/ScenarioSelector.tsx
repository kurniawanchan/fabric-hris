import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface ScenarioSelectorProps {
  scenarios: string[];
  selected: string;
  onChange: (scenario: string) => void;
}

/**
 * EXPERIENCE.md's persistent benchmark-scenario selector. Changing it never
 * restarts or affects an in-progress live run (AD-5) -- it only reflects
 * which scenario is active; the terminal, not this dropdown, controls the
 * Caliper process itself (UJ-1).
 */
export function ScenarioSelector({ scenarios, selected, onChange }: ScenarioSelectorProps) {
  return (
    <Select value={selected} onValueChange={onChange}>
      <SelectTrigger className="w-56" aria-label="Benchmark scenario">
        <SelectValue placeholder="Select a scenario" />
      </SelectTrigger>
      <SelectContent>
        {scenarios.map((scenario) => (
          <SelectItem key={scenario} value={scenario}>
            {scenario}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
