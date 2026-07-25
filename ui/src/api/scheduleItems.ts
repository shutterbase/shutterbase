import { http } from "src/boot/axios";
import { ListResponse, ScheduleItem } from "src/types/api";

export interface ScheduleItemListParams {
  projectId: string;
  from?: string;
  to?: string;
  mine?: boolean;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface ScheduleItemCreate {
  title: string;
  description?: string;
  start: string;
  end: string;
  cardinality?: number;
  projectId: string;
  tagIds?: string[];
}

export interface ScheduleItemUpdate {
  title?: string;
  description?: string;
  start?: string;
  end?: string;
  cardinality?: number;
  tagIds?: string[];
}

export async function list(params: ScheduleItemListParams): Promise<ListResponse<ScheduleItem>> {
  const { data } = await http.get<ListResponse<ScheduleItem>>("/schedule-items", { params });
  return data;
}

export async function get(id: string): Promise<ScheduleItem> {
  const { data } = await http.get<ScheduleItem>(`/schedule-items/${id}`);
  return data;
}

export async function create(body: ScheduleItemCreate): Promise<ScheduleItem> {
  const { data } = await http.post<ScheduleItem>("/schedule-items", body);
  return data;
}

export async function update(id: string, body: ScheduleItemUpdate): Promise<ScheduleItem> {
  const { data } = await http.put<ScheduleItem>(`/schedule-items/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/schedule-items/${id}`);
}

export async function assign(id: string, userId: string): Promise<ScheduleItem> {
  const { data } = await http.put<ScheduleItem>(`/schedule-items/${id}/assignees/${userId}`);
  return data;
}

export async function unassign(id: string, userId: string): Promise<ScheduleItem> {
  const { data } = await http.delete<ScheduleItem>(`/schedule-items/${id}/assignees/${userId}`);
  return data;
}
