// Pure logic of the API-key page — SFC-free for vitest, same pattern as
// uploadReview.ts/uploadTimeline.ts. The server is the authority (§4.13:
// "admin or self" on create, list and revoke); this mirrors it so the UI never
// offers an action that would come back 403.

import { ApiKey } from "src/types/api";

// canManageApiKeys: whose keys may `viewer` mint and revoke — their own, or
// anyone's if they are a platform admin. Mirrors createApiKey/revokeApiKey.
export function canManageApiKeys(viewer: { id: string; isAdmin: boolean } | null, targetUserId: string): boolean {
  if (!viewer || !targetUserId) return false;
  return viewer.isAdmin || viewer.id === targetUserId;
}

// sortApiKeys: active keys first, newest first within each group. A revoked key
// is history — it must not push a live key down the list.
export function sortApiKeys(keys: ApiKey[]): ApiKey[] {
  return [...keys].sort((a, b) => {
    if (a.revoked !== b.revoked) return a.revoked ? 1 : -1;
    return Date.parse(b.createdAt) - Date.parse(a.createdAt);
  });
}

// keyStatus: what the row badge says. "never used" is worth calling out — it is
// the difference between a key that failed to reach its client and one in use.
export function keyStatus(key: ApiKey): "revoked" | "active" | "unused" {
  if (key.revoked) return "revoked";
  return key.lastUsedAt ? "active" : "unused";
}
