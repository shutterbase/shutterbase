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
  count?: number; // how often this face's person was detected in the project
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

// Re-queues every dead-lettered (aiStatus=error) image of the project.
export async function rerunFailed(projectId: string): Promise<number> {
  const { data } = await http.post<{ queued: number }>(`/projects/${projectId}/ai/rerun-failed`);
  return data.queued;
}

// Re-queues EVERY image of the project (full recompute) — confirm before calling.
export async function rerunAll(projectId: string): Promise<number> {
  const { data } = await http.post<{ queued: number }>(`/projects/${projectId}/ai/rerun-all`);
  return data.queued;
}

// Re-queues EVERY image for a vision-only car-number re-read against the AI
// server's currently configured model. Faces, similarity and descriptions are
// kept — much cheaper than rerunAll. Confirm before calling.
export async function rerunNumbers(projectId: string): Promise<number> {
  const { data } = await http.post<{ queued: number }>(`/projects/${projectId}/ai/rerun-numbers`);
  return data.queued;
}

// Rebuilds all person clusters from stored face embeddings (no inference, no
// credits). Fire-and-forget on the server: 202 means started, not finished.
// Discards all merge entries — confirm before calling.
export async function recluster(projectId: string): Promise<void> {
  await http.post(`/projects/${projectId}/ai/recluster`);
}

// Stored raw detection payload of the image's last AI run (the AI server's
// full detail — model reads, evidence axes, notes). 404 when none is stored.
export async function result(imageId: string): Promise<{ raw: unknown; inferredAt?: string }> {
  const { data } = await http.get<{ raw: unknown; inferredAt?: string }>(`/images/${imageId}/ai/result`);
  return data;
}

export async function faces(imageId: string): Promise<AiFace[]> {
  const { data } = await http.get<{ faces: AiFace[] }>(`/images/${imageId}/ai/faces`);
  return data.faces;
}

export async function similar(imageId: string, page: number, pageSize = 20): Promise<AiSimilarPage> {
  const { data } = await http.get<AiSimilarPage>(`/images/${imageId}/ai/similar`, { params: { page, pageSize } });
  return data;
}

// Copies the AI server's stored descriptions onto this project's images
// (backfill for images analyzed before descriptions were persisted here).
export async function syncDescriptions(projectId: string): Promise<{ updated: number; skipped: number }> {
  const { data } = await http.post<{ updated: number; skipped: number }>(`/projects/${projectId}/ai/sync-descriptions`);
  return data;
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
  hasMore: boolean;
}

// raw=true skips merge-group resolution and returns one cluster's own faces.
export async function personImages(projectId: string, personRef: string, page = 0, pageSize = 20, raw = false): Promise<AiPersonImagesPage> {
  const { data } = await http.get<AiPersonImagesPage>(`/projects/${projectId}/ai/persons/${personRef}/images`, {
    params: { page, pageSize, ...(raw ? { raw: "true" } : {}) },
  });
  return data;
}

// --- People overview + merge review (GLOBAL — persons span projects) ---
// The server scopes everything to the caller's viewable projects; merge
// mutations additionally require projectAdmin on at least one project (403).

export interface AiRankedPerson {
  personRef: string;
  count: number;
  name?: string;
  sample?: AiPersonImage;
}

export interface AiRankedPersonsPage {
  items: AiRankedPerson[];
  total: number;
  page: number;
  pageSize: number;
}

export async function rankedPersons(page = 0, pageSize = 24): Promise<AiRankedPersonsPage> {
  const { data } = await http.get<AiRankedPersonsPage>(`/ai/persons`, { params: { page, pageSize } });
  return data;
}

// A person's appearances across every viewable project (People page samples).
export async function personImagesGlobal(personRef: string, page = 0, pageSize = 20, raw = false): Promise<AiPersonImagesPage> {
  const { data } = await http.get<AiPersonImagesPage>(`/ai/persons/${personRef}/images`, {
    params: { page, pageSize, ...(raw ? { raw: "true" } : {}) },
  });
  return data;
}

// Person names are user-given labels for face clusters, stored in shutterbase
// (not the AI server) and shared by every member of a merge group.
export async function personNames(refs: string[]): Promise<Record<string, string>> {
  const { data } = await http.get<{ names: Record<string, string> }>(`/ai/persons/names`, {
    params: { ref: refs },
    paramsSerializer: { indexes: null },
  });
  return data.names;
}

// Empty name clears it. Requires projectAdmin on ≥1 project (403 otherwise).
export async function setPersonName(personRef: string, name: string): Promise<void> {
  await http.put(`/ai/persons/${personRef}/name`, { name });
}

export interface AiMergeCandidate {
  personA: string;
  personB: string;
  sim: number;
}

export interface AiMergeCandidates {
  candidate?: AiMergeCandidate;
  remaining: number;
}

// person (optional) narrows the queue to pairs involving that person.
export async function mergeNext(skip = 0, person = ""): Promise<AiMergeCandidates> {
  const { data } = await http.get<AiMergeCandidates>(`/ai/merge/next`, { params: { skip, ...(person ? { person } : {}) } });
  return data;
}

// verdict "same" records a reversible merge entry; deleteMerge splits it again.
export async function mergeDecide(personA: string, personB: string, verdict: "same" | "different"): Promise<void> {
  await http.post(`/ai/merge/decide`, { personA, personB, verdict });
}

export interface AiMerge {
  personA: string;
  personB: string;
  createdAt: string;
}

export async function merges(): Promise<AiMerge[]> {
  const { data } = await http.get<{ items: AiMerge[] }>(`/ai/merge`);
  return data.items;
}

export async function deleteMerge(personA: string, personB: string): Promise<void> {
  await http.delete(`/ai/merge`, { params: { personA, personB } });
}
