import { http } from "src/boot/axios";

export interface TagStatistic {
  id: string;
  name: string;
  description: string;
  type: string;
  count: number;
}

export interface ProjectStatistics {
  tags: TagStatistic[];
}

export async function project(projectId: string): Promise<ProjectStatistics> {
  const { data } = await http.get<ProjectStatistics>(`/statistics/${projectId}`);
  return data;
}

export async function syncImageTags(): Promise<{ synced: number }> {
  const { data } = await http.get<{ synced: number }>("/sync-image-tags");
  return data;
}

export async function uploadUrl(name: string): Promise<{ url: string }> {
  const { data } = await http.get<{ url: string }>("/upload-url", { params: { name } });
  return data;
}
