import { http } from "src/boot/axios";
import { User, ListResponse } from "src/types/api";

export interface UserListParams {
  search?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface UserCreate {
  username: string;
  email: string;
  password: string;
  firstName: string;
  lastName: string;
  copyrightTag?: string;
  active?: boolean;
  roleId: string;
  forcePasswordChange?: boolean;
}

export interface UserUpdate {
  firstName?: string;
  lastName?: string;
  copyrightTag?: string;
  email?: string;
  password?: string;
  // admin-only:
  active?: boolean;
  roleId?: string;
  forcePasswordChange?: boolean;
  activeProjectId?: string;
}

export async function list(params: UserListParams = {}): Promise<ListResponse<User>> {
  const { data } = await http.get<ListResponse<User>>("/users", { params });
  return data;
}

export async function get(id: string): Promise<User> {
  const { data } = await http.get<User>(`/users/${id}`);
  return data;
}

export async function create(body: UserCreate): Promise<User> {
  const { data } = await http.post<User>("/users", body);
  return data;
}

export async function update(id: string, body: UserUpdate): Promise<User> {
  const { data } = await http.put<User>(`/users/${id}`, body);
  return data;
}

export async function setActiveProject(projectId: string): Promise<void> {
  await http.patch("/users/me/active-project", { projectId });
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/users/${id}`);
}
