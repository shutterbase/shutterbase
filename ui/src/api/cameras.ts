import { http } from "src/boot/axios";
import { Camera, ListResponse } from "src/types/api";

export interface CameraListParams {
  userId?: string;
  search?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface CameraCreate {
  name: string;
  userId?: string;
}

export interface CameraUpdate {
  name?: string;
}

export async function list(params: CameraListParams = {}): Promise<ListResponse<Camera>> {
  const { data } = await http.get<ListResponse<Camera>>("/cameras", { params });
  return data;
}

export async function get(id: string): Promise<Camera> {
  const { data } = await http.get<Camera>(`/cameras/${id}`);
  return data;
}

export async function create(body: CameraCreate): Promise<Camera> {
  const { data } = await http.post<Camera>("/cameras", body);
  return data;
}

export async function update(id: string, body: CameraUpdate): Promise<Camera> {
  const { data } = await http.put<Camera>(`/cameras/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/cameras/${id}`);
}
