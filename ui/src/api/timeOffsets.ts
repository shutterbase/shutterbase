import { http } from "src/boot/axios";
import { TimeOffset, ListResponse } from "src/types/api";

export interface TimeOffsetListParams {
  cameraId?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface TimeOffsetCreate {
  cameraId: string;
  serverTime: string;
  cameraTime: string;
}

export async function list(params: TimeOffsetListParams = {}): Promise<ListResponse<TimeOffset>> {
  const { data } = await http.get<ListResponse<TimeOffset>>("/time-offsets", { params });
  return data;
}

export async function get(id: string): Promise<TimeOffset> {
  const { data } = await http.get<TimeOffset>(`/time-offsets/${id}`);
  return data;
}

export async function create(body: TimeOffsetCreate): Promise<TimeOffset> {
  const { data } = await http.post<TimeOffset>("/time-offsets", body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/time-offsets/${id}`);
}
