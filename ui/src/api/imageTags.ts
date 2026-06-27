import { http } from "src/boot/axios";
import { ImageTag, ListResponse } from "src/types/api";

export interface ImageTagListParams {
  projectId: string; // required
  search?: string;
  type?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface ImageTagCreate {
  name: string;
  description?: string;
  isAlbum?: boolean;
  type: string;
  projectId: string;
}

export interface ImageTagUpdate {
  name?: string;
  description?: string;
  isAlbum?: boolean;
  type?: string;
}

export async function list(params: ImageTagListParams): Promise<ListResponse<ImageTag>> {
  const { data } = await http.get<ListResponse<ImageTag>>("/image-tags", { params });
  return data;
}

export async function get(id: string): Promise<ImageTag> {
  const { data } = await http.get<ImageTag>(`/image-tags/${id}`);
  return data;
}

export async function create(body: ImageTagCreate): Promise<ImageTag> {
  const { data } = await http.post<ImageTag>("/image-tags", body);
  return data;
}

export async function update(id: string, body: ImageTagUpdate): Promise<ImageTag> {
  const { data } = await http.put<ImageTag>(`/image-tags/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/image-tags/${id}`);
}
