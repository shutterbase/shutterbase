import { http } from "src/boot/axios";
import { Upload, ListResponse, UploadState } from "src/types/api";

export interface UploadListParams {
  projectId?: string;
  userId?: string;
  state?: UploadState;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface UploadCreate {
  name: string;
  projectId: string;
  cameraId: string;
  userId?: string;
}

export interface UploadUpdate {
  name?: string;
  state?: UploadState;
}

export async function list(params: UploadListParams = {}): Promise<ListResponse<Upload>> {
  const { data } = await http.get<ListResponse<Upload>>("/uploads", { params });
  return data;
}

export async function get(id: string): Promise<Upload> {
  const { data } = await http.get<Upload>(`/uploads/${id}`);
  return data;
}

export async function create(body: UploadCreate): Promise<Upload> {
  const { data } = await http.post<Upload>("/uploads", body);
  return data;
}

export async function update(id: string, body: UploadUpdate): Promise<Upload> {
  const { data } = await http.put<Upload>(`/uploads/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/uploads/${id}`);
}
