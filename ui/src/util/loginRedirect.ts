// Post-login target from the ?redirect= query. Only same-app absolute paths
// survive — full URLs and protocol-relative "//host" are open-redirect vectors
// and fall back to the start page.
export function sanitizeRedirect(value: unknown): string {
  if (typeof value !== "string" || !value.startsWith("/") || value.startsWith("//")) return "/";
  return value;
}
