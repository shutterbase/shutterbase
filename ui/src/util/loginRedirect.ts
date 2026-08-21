// Post-login target from the ?redirect= query. Only same-app absolute paths
// survive — full URLs and protocol-relative "//host" are open-redirect vectors,
// and /logout would sign the user straight out again; all fall back to the
// start page.
export function sanitizeRedirect(value: unknown): string {
  if (typeof value !== "string" || !value.startsWith("/") || value.startsWith("//") || value.startsWith("/logout")) return "/";
  return value;
}
