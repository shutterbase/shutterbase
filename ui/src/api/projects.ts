import { http } from "src/boot/axios";
import { Project, ListResponse } from "src/types/api";

export interface ProjectListParams {
  search?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export type ProjectCreate = Omit<Project, "id" | "createdAt" | "updatedAt">;
export type ProjectUpdate = Partial<ProjectCreate>;

export async function list(params: ProjectListParams = {}): Promise<ListResponse<Project>> {
  const { data } = await http.get<ListResponse<Project>>("/projects", { params });
  return data;
}

export async function get(id: string): Promise<Project> {
  const { data } = await http.get<Project>(`/projects/${id}`);
  return data;
}

export async function create(body: ProjectCreate): Promise<Project> {
  const { data } = await http.post<Project>("/projects", body);
  return data;
}

export async function update(id: string, body: ProjectUpdate): Promise<Project> {
  const { data } = await http.put<Project>(`/projects/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/projects/${id}`);
}
