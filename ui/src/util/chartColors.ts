// Chart palette accessors. The actual hex values live as CSS custom properties
// in css/tailwind.css (:root / html.dark) — validated with the dataviz palette
// checks against both surfaces — so components get the correct mode for free.

export const CATEGORICAL = ["var(--chart-cat-1)", "var(--chart-cat-2)", "var(--chart-cat-3)", "var(--chart-cat-4)", "var(--chart-cat-5)", "var(--chart-cat-6)"];

export const OTHER_COLOR = "var(--chart-other)";
export const MANUAL_COLOR = "var(--chart-manual)";
export const AI_COLOR = "var(--chart-ai)";

const HEAT_STEPS = 5;

// heatColor buckets a cell value onto the 5-step sequential ramp; zero cells
// get no encoding (the component renders them as muted surface).
export function heatColor(value: number, max: number): string {
  if (value <= 0 || max <= 0) return "";
  const step = Math.min(HEAT_STEPS, Math.max(1, Math.ceil((value / max) * HEAT_STEPS)));
  return `var(--chart-heat-${step})`;
}
