import { http } from "src/boot/axios";
import { DownloadConfig, ListResponse } from "src/types/api";

export interface DownloadConfigCreate {
  name: string;
  projectId: string;
  whitelistTagIds?: string[];
  blacklistTagIds?: string[];
  blockedImageIds?: string[];
  deltaSubfolder?: boolean;
  groupByDate?: boolean;
}

export interface DownloadConfigUpdate {
  name?: string;
  whitelistTagIds?: string[];
  blacklistTagIds?: string[];
  blockedImageIds?: string[];
  deltaSubfolder?: boolean;
  groupByDate?: boolean;
  lastDownloadAt?: string;
}

export async function list(projectId: string): Promise<ListResponse<DownloadConfig>> {
  const { data } = await http.get<ListResponse<DownloadConfig>>("/download-configs", { params: { projectId } });
  return data;
}

export async function create(body: DownloadConfigCreate): Promise<DownloadConfig> {
  const { data } = await http.post<DownloadConfig>("/download-configs", body);
  return data;
}

export async function update(id: string, body: DownloadConfigUpdate): Promise<DownloadConfig> {
  const { data } = await http.put<DownloadConfig>(`/download-configs/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/download-configs/${id}`);
}
