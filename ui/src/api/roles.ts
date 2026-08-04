import { http } from "src/boot/axios";
import { Role, ListResponse } from "src/types/api";

export interface RoleListParams {
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export async function list(params: RoleListParams = {}): Promise<ListResponse<Role>> {
  const { data } = await http.get<ListResponse<Role>>("/roles", { params });
  return data;
}

export async function get(id: string): Promise<Role> {
  const { data } = await http.get<Role>(`/roles/${id}`);
  return data;
}
