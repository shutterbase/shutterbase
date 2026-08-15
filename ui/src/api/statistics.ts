import { http } from "src/boot/axios";

export interface TagStatistic {
  id: string;
  name: string;
  displayName?: string;
  description: string;
  type: string;
  count: number;
}

export interface StatTotals {
  images: number;
  photographers: number;
  manualTags: number;
  aiTags: number;
  storageBytes: number;
  unbucketedImages: number; // legacy images without capturedAtCorrected — absent from day buckets
}

export interface StatDay {
  date: string; // "2026-08-11", event timezone
  total: number;
  byUser: Record<string, number>;
  byHour: number[]; // 24 entries
}

export interface StatAssignmentDay {
  date: string;
  manual: number;
  ai: number;
}

export interface StatPhotographer {
  id: string;
  firstName: string;
  lastName: string;
  copyrightTag: string;
  imageCount: number;
}

export interface StatAiStatus {
  done: number;
  inFlight: number;
  error: number;
  notQueued: number;
}

export interface StatUploadStates {
  open: number;
  ready: number;
  reviewed: number;
}

export interface ProjectStatistics {
  totals: StatTotals;
  days: StatDay[]; // sorted asc, sparse (missing days carry no images)
  assignmentsPerDay: StatAssignmentDay[];
  photographers: StatPhotographer[]; // sorted by imageCount desc
  aiStatus: StatAiStatus;
  uploadStates: StatUploadStates;
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
