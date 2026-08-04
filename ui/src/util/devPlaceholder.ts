// Dev-only stand-in for missing S3 thumbnails: in dev there is often no object
// store behind the presigned URLs, so images 404. A deterministic colored SVG
// per image id (varied aspect ratios) keeps gallery/strip layouts reviewable —
// no external network call and no image-id leaked off-box. Returns null in prod.
// ponytail: drop once S3 dev fixtures exist.
export function devPlaceholder(id: string): string | null {
  if (!import.meta.env.DEV) return null;
  const ratios = [
    [4, 3],
    [3, 4],
    [1, 1],
    [3, 2],
    [2, 3],
  ];
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) | 0;
  h = Math.abs(h);
  const [w, hh] = ratios[h % ratios.length];
  const hue = h % 360;
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${w * 220}" height="${hh * 220}"><rect width="100%" height="100%" fill="hsl(${hue} 28% 22%)"/></svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}
