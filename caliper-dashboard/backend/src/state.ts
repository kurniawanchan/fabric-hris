/**
 * AD-5: the backend's exactly-two mutable singletons -- selected scenario and
 * is-live -- independent axes, never derived from one another. Selecting a
 * scenario never by itself changes isLive; isLive never implies any
 * particular scenario is selected. No session/multi-client model exists --
 * there is never more than one operator or one browser window.
 *
 * isLive's trigger (first successfully parsed data record from a watched
 * source, never mere file-existence/open) is set by the watchers/poller in a
 * later story -- this module only exposes the mechanism, it doesn't decide
 * when to flip it.
 */

export interface DashboardState {
  scenario: string;
  isLive: boolean;
}

let scenario: string;
let isLive = false;

export function initScenario(defaultScenario: string): void {
  scenario = defaultScenario;
}

export function getState(): DashboardState {
  return { scenario, isLive };
}

export function setScenario(next: string): void {
  scenario = next;
  // Deliberately does NOT touch isLive (AD-5: independent axes).
}

export function setIsLive(next: boolean): void {
  isLive = next;
}
