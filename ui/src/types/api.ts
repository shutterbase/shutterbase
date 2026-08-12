// REST contract types — REWRITE-SPEC.md §4 (FROZEN). Replaces the PocketBase SDK types.

export interface ListResponse<T> {
  limit: number;
  offset: number;
  total: number;
  items: T[];
}

export interface ListParams {
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

// Error envelopes (§1). Controllers: {message,code}. go-basicauth routes: {error,message}.
export interface ApiError {
  code?: string;
  message?: string;
  error?: string;
}

export interface Role {
  id: string;
  key: string;
  description: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface EmbeddedUser {
  id: string;
  firstName: string;
  lastName: string;
  copyrightTag?: string;
  email?: string;
}

export interface EmbeddedProject {
  id: string;
  name: string;
  uploadReviewEnabled?: boolean;
}

export interface EmbeddedCamera {
  id: string;
  name: string;
}

export interface EmbeddedUpload {
  id: string;
  name: string;
  state?: UploadState;
}

export interface EmbeddedTag {
  id: string;
  name: string;
  displayName?: string;
  type: string;
  isAlbum?: boolean;
  order?: number | null;
  description?: string;
}

export interface DownloadUrls {
  original: string;
  256: string;
  512: string;
  1024: string;
  2048: string;
}

// image.tags[] element (assignment, denormalized on the image)
export interface ImageTagAssignment {
  id: string;
  type: string; // manual | inferred | default
  image?: { id: string };
  tag: EmbeddedTag;
  createdAt?: string;
  updatedAt?: string;
}

export interface Image {
  id: string;
  fileName: string;
  computedFileName: string;
  exifData: Record<string, any>;
  capturedAt: string;
  capturedAtCorrected: string;
  width?: number;
  height?: number;
  size: number;
  storageId: string;
  user: EmbeddedUser;
  camera: EmbeddedCamera;
  project: EmbeddedProject;
  upload: EmbeddedUpload;
  tags: ImageTagAssignment[];
  imageTags: string[];
  downloadUrls: DownloadUrls;
  inferredAt?: string | null;
  aiStatus?: AiStatus | null;
  aiError?: string;
  createdAt: string;
  updatedAt: string;
}

export type AiStatus = "pending" | "processing" | "done" | "error";

export interface ImageTag {
  id: string;
  name: string;
  displayName?: string; // optional UI label; empty = display the name
  description: string;
  isAlbum: boolean;
  aiEnabled: boolean; // part of the AI vocabulary; false = model may not assign it
  order?: number | null; // positive rank; lower = applied/shown first, unset = last
  type: string; // template | default | manual | custom
  project: EmbeddedProject;
  createdAt: string;
  updatedAt: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  copyright: string;
  copyrightReference: string;
  copyrightTagPrefix?: string;
  locationName: string;
  locationCode: string;
  locationCity: string;
  aiSystemMessage?: string;
  uploadReviewEnabled: boolean;
  // Event period (S15): frames the schedule calendar. Optional.
  startAt?: string | null;
  endAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

// One coverable block of the event schedule (S15). cardinality is the TARGET
// headcount, not a cap — overbooking is allowed (violett). A block may be
// subdivided into shifts (nested, one level; kind=break marks an unclaimable
// pause tile) — claiming then happens on the shifts.
export interface ScheduleItem {
  id: string;
  title: string;
  description: string;
  start: string;
  end: string;
  cardinality: number;
  kind: "item" | "break";
  parentId: string; // "" for top-level blocks
  assignees: EmbeddedUser[];
  tags: EmbeddedTag[];
  shifts?: ScheduleItem[]; // present on top-level items only, start-ordered
  project: EmbeddedProject;
  createdAt: string;
  updatedAt: string;
}

// A personal per-project preset for the in-browser bulk download page.
// whitelistTagIds are AND-applied server-side; blacklistTagIds and
// blockedImageIds are excluded client-side by the runner. lastDownloadAt is
// the start time of the last completed run — the delta window's anchor.
export interface DownloadConfig {
  id: string;
  name: string;
  whitelistTagIds: string[];
  blacklistTagIds: string[];
  blockedImageIds: string[];
  deltaSubfolder: boolean;
  groupByDate: boolean;
  reviewedOnly: boolean;
  folderStructure: "default" | "weekday";
  lastDownloadAt: string | null;
  projectId: string;
  createdAt: string;
  updatedAt: string;
}

// One lane of the upload tagging timeline. Exactly one of scheduleItemId
// (mutually exclusive with its siblings) or tagId (stacks freely) is set.
export interface TimelineTrack {
  scheduleItemId?: string;
  tagId?: string;
  start: string;
  end: string;
  enabled: boolean;
}

export interface Camera {
  id: string;
  name: string;
  user: EmbeddedUser;
  createdAt: string;
  updatedAt: string;
}

// API keys (§4.13). `token` is present ONLY in the create response — the secret
// is stored as an argon2 hash and can never be read back.
export interface ApiKey {
  id: string;
  keyId: string;
  name: string;
  userId: string;
  revoked: boolean;
  lastUsedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ApiKeyWithToken extends ApiKey {
  token: string;
}

// Upload review flow (§4.9): open -> ready (photographer submits) -> reviewed
// (projectAdmin accepts); ready -> open sends it back for rework.
export type UploadState = "open" | "ready" | "reviewed";

// Per-upload tagging metrics. Rates are server-derived from the counts.
export interface UploadMetrics {
  imageCount: number;
  tagCount: number;
  tagsPerImage: number;
  taggingSeconds: number;
  imagesPerSecond: number;
  timeToReadySeconds: number;
  reviewCycles: number;
  errorCount: number;
  rejectedCount: number;
  aiDone: number;
  aiInFlight: number;
  aiError: number;
}

export interface Upload {
  id: string;
  name: string;
  state: UploadState;
  project: EmbeddedProject;
  user: EmbeddedUser;
  camera: EmbeddedCamera;
  imageCount?: number;
  metrics?: UploadMetrics;
  timeline?: TimelineTrack[];
  createdAt: string;
  updatedAt: string;
}

export interface TimeOffset {
  id: string;
  serverTime: string;
  cameraTime: string;
  timeOffset: number;
  camera: EmbeddedCamera;
  upToDate: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectAssignment {
  id: string;
  project: EmbeddedProject;
  user: EmbeddedUser;
  role: Role;
  createdAt: string;
  updatedAt: string;
}

// Per-user hotkey customization. bindings: hotkey action id → key combos
// (overrides the system default; empty list = unbound). tagBindings: key combo
// → image tag name toggled on the current image (null = system defaults).
export interface UserHotkeys {
  bindings?: Record<string, string[]> | null;
  tagBindings?: Record<string, string> | null;
}

export interface User {
  id: string;
  username: string;
  email: string;
  verified: boolean;
  active: boolean;
  firstName: string;
  lastName: string;
  copyrightTag: string;
  forcePasswordChange: boolean;
  totpEnabled: boolean;
  role: Role;
  activeProject: EmbeddedProject | null;
  projectAssignments: ProjectAssignment[];
  hotkeys?: UserHotkeys | null;
  createdAt: string;
  updatedAt: string;
}

export interface Impersonating {
  realUserId: string;
  realUserName: string;
}

// /users/me — effective user, plus impersonating block when active
export type CurrentUser = User & { impersonating?: Impersonating };
