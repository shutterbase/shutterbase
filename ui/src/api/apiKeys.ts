import { http } from "src/boot/axios";
import { ApiKey, ApiKeyWithToken, ListResponse } from "src/types/api";

export interface ApiKeyListParams {
  // Admins may list another user's keys; for everyone else the server ignores
  // this and scopes the result to the caller.
  userId?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface ApiKeyCreate {
  name: string;
  // Omit to mint for yourself. A non-admin may only pass their own id.
  userId?: string;
}

export async function list(params: ApiKeyListParams = {}): Promise<ListResponse<ApiKey>> {
  const { data } = await http.get<ListResponse<ApiKey>>("/api-keys", { params });
  return data;
}

// The response carries `token` exactly once — it is unrecoverable afterwards, so
// the caller must show it immediately.
export async function create(body: ApiKeyCreate): Promise<ApiKeyWithToken> {
  const { data } = await http.post<ApiKeyWithToken>("/api-keys", body);
  return data;
}

// Revoking keeps the row (with revoked=true) rather than deleting it, so the
// audit trail of what once existed survives.
export async function revoke(id: string): Promise<void> {
  await http.delete(`/api-keys/${id}`);
}
