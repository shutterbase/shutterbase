// AI detection: queue status/positions, manual reruns, and the proxied
// AI-server lookups (faces / person search / similar images). The server
// resolves AI-server refs to regular Image DTOs — the browser never talks to
// the AI server itself.
import { http } from "src/boot/axios";
import { AiStatus, Image } from "src/types/api";

export interface AiImageStatus {
  imageId: string;
  status?: AiStatus;
  position?: number; // 1-based global queue position; only when pending
}

export interface AiQueueStatus {
  items: AiImageStatus[];
  queueTotal: number;
}

export interface AiUploadStatus {
  pending: number;
  processing: number;
  done: number;
  error: number;
  ahead: number; // foreign pending images queued before this upload's oldest
}

export interface AiFace {
  x: number;
  y: number;
  w: number;
  h: number;
  personRef?: string;
}

export interface AiPersonImage {
  image: Image;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface AiPersonImagesPage {
  items: AiPersonImage[];
  total: number;
  page: number;
  pageSize: number;
}

export interface AiSimilarImage {
  image: Image;
  similarity: number;
}

export interface AiSimilarPage {
  items: AiSimilarImage[];
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export async function queueStatus(projectId: string, imageIds: string[]): Promise<AiQueueStatus> {
  const { data } = await http.get<AiQueueStatus>(`/projects/${projectId}/ai/status`, {
    params: { imageId: imageIds },
    paramsSerializer: { indexes: null },
  });
  return data;
}

export async function uploadStatus(uploadId: string): Promise<AiUploadStatus> {
  const { data } = await http.get<AiUploadStatus>(`/uploads/${uploadId}/ai`);
  return data;
}

export async function rerunImage(imageId: string): Promise<void> {
  await http.post(`/images/${imageId}/ai/rerun`);
}

export async function rerunUpload(uploadId: string, failedOnly = false): Promise<number> {
  const { data } = await http.post<{ queued: number }>(`/uploads/${uploadId}/ai/rerun`, null, {
    params: failedOnly ? { failedOnly: "true" } : {},
  });
  return data.queued;
}

export async function rerunBatch(projectId: string, imageIds: string[]): Promise<number> {
  const { data } = await http.post<{ queued: number }>(`/projects/${projectId}/ai/rerun`, { imageIds });
  return data.queued;
}

export async function faces(imageId: string): Promise<AiFace[]> {
  const { data } = await http.get<{ faces: AiFace[] }>(`/images/${imageId}/ai/faces`);
  return data.faces;
}

export async function personImages(projectId: string, personRef: string, page: number, pageSize = 20): Promise<AiPersonImagesPage> {
  const { data } = await http.get<AiPersonImagesPage>(`/projects/${projectId}/ai/persons/${encodeURIComponent(personRef)}/images`, { params: { page, pageSize } });
  return data;
}

export async function similar(imageId: string, page: number, pageSize = 20): Promise<AiSimilarPage> {
  const { data } = await http.get<AiSimilarPage>(`/images/${imageId}/ai/similar`, { params: { page, pageSize } });
  return data;
}
