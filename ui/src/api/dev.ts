// DEV quick-actions API seam (REWRITE-SPEC "Local dev quick actions").
//
// DEV-ONLY: every endpoint here lives under /api/v1/dev/*, which the backend
// registers ONLY when DEV=true (404 otherwise). This module is imported solely
// by DevPanel.vue, which renders only in dev builds (import.meta.env.DEV), so a
// production bundle tree-shakes it away entirely.
import { http } from "src/boot/axios";

export interface DevLoginBody {
  userId?: string;
  role?: string; // seeded username == role key: admin|user|projectAdmin|projectEditor|projectViewer
}

export interface DevTimeOffsetBody {
  cameraId: string;
  driftSeconds: number;
  stale?: boolean;
}

export interface DevImagesBody {
  uploadId: string;
  projectId?: string;
  count: number;
}

export interface DevClockBody {
  at?: string; // ISO instant to freeze to; omit (or reset) to go live
  reset?: boolean;
}

export async function login(body: DevLoginBody): Promise<Record<string, unknown>> {
  const { data } = await http.post("/dev/login", body);
  return data;
}

export async function impersonate(userId: string): Promise<Record<string, unknown>> {
  const { data } = await http.post(`/dev/impersonate/${userId}`);
  return data;
}

export async function roleToggle(role?: string): Promise<Record<string, unknown>> {
  const { data } = await http.post("/dev/role", { role });
  return data;
}

export async function timeOffset(body: DevTimeOffsetBody): Promise<Record<string, unknown>> {
  const { data } = await http.post("/dev/time-offset", body);
  return data;
}

export async function images(body: DevImagesBody): Promise<{ created: number; imageIds: string[] }> {
  const { data } = await http.post("/dev/images", body);
  return data;
}

export async function infer(imageId: string, tags?: string[]): Promise<Record<string, unknown>> {
  const { data } = await http.post(`/dev/infer/${imageId}`, tags ? { tags } : {});
  return data;
}

export async function syncTags(): Promise<{ synced: number }> {
  const { data } = await http.post("/dev/sync-tags");
  return data;
}

export async function defaultTags(projectId: string): Promise<{ processed: number }> {
  const { data } = await http.post("/dev/default-tags", { projectId });
  return data;
}

export async function reseed(): Promise<Record<string, unknown>> {
  const { data } = await http.post("/dev/reseed");
  return data;
}

export async function clock(body: DevClockBody): Promise<{ now: string; frozen: boolean }> {
  const { data } = await http.post("/dev/clock", body);
  return data;
}

export async function apiKey(name?: string): Promise<{ token: string; keyId: string }> {
  const { data } = await http.post("/dev/api-key", { name });
  return data;
}
