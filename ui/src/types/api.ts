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
}

export interface EmbeddedCamera {
  id: string;
  name: string;
}

export interface EmbeddedUpload {
  id: string;
  name: string;
}

export interface EmbeddedTag {
  id: string;
  name: string;
  type: string;
  isAlbum?: boolean;
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
  createdAt: string;
  updatedAt: string;
}

export interface ImageTag {
  id: string;
  name: string;
  description: string;
  isAlbum: boolean;
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
  locationName: string;
  locationCode: string;
  locationCity: string;
  aiSystemMessage?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Camera {
  id: string;
  name: string;
  user: EmbeddedUser;
  createdAt: string;
  updatedAt: string;
}

export interface Upload {
  id: string;
  name: string;
  project: EmbeddedProject;
  user: EmbeddedUser;
  camera: EmbeddedCamera;
  imageCount?: number;
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
  createdAt: string;
  updatedAt: string;
}

export interface Impersonating {
  realUserId: string;
  realUserName: string;
}

// /users/me — effective user, plus impersonating block when active
export type CurrentUser = User & { impersonating?: Impersonating };
