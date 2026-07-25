import { http } from "src/boot/axios";
import { Upload, ListResponse, UploadState, TimelineTrack } from "src/types/api";

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

// Upload with the apply diff the timeline endpoint reports back.
export type UploadWithApplied = Upload & { applied?: { created: number; deleted: number } };

// Persist the tagging-timeline editor state; the server reconciles all
// "scheduled" tag assignments against it atomically.
export async function applyTimeline(id: string, tracks: TimelineTrack[]): Promise<UploadWithApplied> {
  const { data } = await http.put<UploadWithApplied>(`/uploads/${id}/timeline`, { tracks });
  return data;
}
