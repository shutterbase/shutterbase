// pageWindow returns the 1-based page buttons a pager should render: every
// page when they all fit, otherwise first/last plus a current±1 window with
// "…" marking the gaps.
export function pageWindow(current: number, count: number): (number | "…")[] {
  if (count <= 7) return Array.from({ length: count }, (_, i) => i + 1);
  const out: (number | "…")[] = [1];
  if (current > 3) out.push("…");
  for (let p = Math.max(2, current - 1); p <= Math.min(count - 1, current + 1); p++) out.push(p);
  if (current < count - 2) out.push("…");
  out.push(count);
  return out;
}
