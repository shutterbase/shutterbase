import { http } from "src/boot/axios";
import { ProjectAssignment, ListResponse } from "src/types/api";

export interface ProjectAssignmentListParams {
  projectId?: string;
  userId?: string;
  limit?: number;
  offset?: number;
}

export interface ProjectAssignmentCreate {
  projectId: string;
  userId: string;
  roleId: string;
}

export async function list(params: ProjectAssignmentListParams = {}): Promise<ListResponse<ProjectAssignment>> {
  const { data } = await http.get<ListResponse<ProjectAssignment>>("/project-assignments", { params });
  return data;
}

export async function get(id: string): Promise<ProjectAssignment> {
  const { data } = await http.get<ProjectAssignment>(`/project-assignments/${id}`);
  return data;
}

export async function create(body: ProjectAssignmentCreate): Promise<ProjectAssignment> {
  const { data } = await http.post<ProjectAssignment>("/project-assignments", body);
  return data;
}

export async function update(id: string, roleId: string): Promise<ProjectAssignment> {
  const { data } = await http.put<ProjectAssignment>(`/project-assignments/${id}`, { roleId });
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/project-assignments/${id}`);
}
