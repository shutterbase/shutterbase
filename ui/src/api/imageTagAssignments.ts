import { http } from "src/boot/axios";
import { ImageTagAssignment, ListResponse } from "src/types/api";

export interface AssignmentListParams {
  imageId?: string;
  tagId?: string;
  limit?: number;
  offset?: number;
}

export interface AssignmentCreate {
  imageId: string;
  imageTagId: string;
  type: string;
}

export async function list(params: AssignmentListParams): Promise<ListResponse<ImageTagAssignment>> {
  const { data } = await http.get<ListResponse<ImageTagAssignment>>("/image-tag-assignments", { params });
  return data;
}

export async function get(id: string): Promise<ImageTagAssignment> {
  const { data } = await http.get<ImageTagAssignment>(`/image-tag-assignments/${id}`);
  return data;
}

// idempotent: existing (image,tag) -> 200 existing row
export async function create(body: AssignmentCreate): Promise<ImageTagAssignment> {
  const { data } = await http.post<ImageTagAssignment>("/image-tag-assignments", body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/image-tag-assignments/${id}`);
}
