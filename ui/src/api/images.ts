import { http } from "src/boot/axios";
import { Image, ListResponse } from "src/types/api";

export interface ImageListParams {
  projectId: string; // required (§4.3)
  uploadId?: string;
  cameraId?: string;
  userId?: string;
  search?: string;
  tagId?: string[]; // repeated, AND-combined server-side
  orientation?: "portrait" | "landscape";
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface ImageCreate {
  fileName: string;
  storageId: string;
  size: number;
  width?: number;
  height?: number;
  capturedAt?: string;
  exifData?: Record<string, any>;
  cameraId: string;
  uploadId: string;
  projectId: string;
}

export interface ImageUpdate {
  fileName?: string;
  capturedAt?: string;
  exifData?: Record<string, any>;
  cameraId?: string;
  uploadId?: string;
}

export async function list(params: ImageListParams): Promise<ListResponse<Image>> {
  // indexes:null -> repeated `tagId=a&tagId=b` (server reads repeated values)
  const { data } = await http.get<ListResponse<Image>>("/images", { params, paramsSerializer: { indexes: null } });
  return data;
}

export async function get(id: string): Promise<Image> {
  const { data } = await http.get<Image>(`/images/${id}`);
  return data;
}

export async function create(body: ImageCreate): Promise<Image> {
  const { data } = await http.post<Image>("/images", body);
  return data;
}

export async function update(id: string, body: ImageUpdate): Promise<Image> {
  const { data } = await http.put<Image>(`/images/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/images/${id}`);
}
