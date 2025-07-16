/**
 * This file was @generated using pocketbase-typegen
 */

import type PocketBase from "pocketbase";
import type { RecordService } from "pocketbase";

export enum Collections {
  Cameras = "cameras",
  ImageTagAssignments = "image_tag_assignments",
  ImageTags = "image_tags",
  Images = "images",
  Inferences = "inferences",
  ProjectAssignments = "project_assignments",
  Projects = "projects",
  Roles = "roles",
  TimeOffsets = "time_offsets",
  Uploads = "uploads",
  Users = "users",
}

// Alias types for improved usability
export type IsoDateString = string;
export type RecordIdString = string;
export type HTMLString = string;

// System fields
export type BaseSystemFields<T = never> = {
  id: RecordIdString;
  created: IsoDateString;
  updated: IsoDateString;
  collectionId: string;
  collectionName: Collections;
  expand?: T;
};

export type AuthSystemFields<T = never> = {
  email: string;
  emailVisibility: boolean;
  username: string;
  verified: boolean;
} & BaseSystemFields<T>;

// Record types for each collection

export type CamerasRecord = {
  name: string;
  user: RecordIdString;
};

export enum ImageTagAssignmentsTypeOptions {
  "manual" = "manual",
  "inferred" = "inferred",
  "default" = "default",
}
export type ImageTagAssignmentsRecord = {
  image: RecordIdString;
  imageTag: RecordIdString;
  type?: ImageTagAssignmentsTypeOptions;
};

export enum ImageTagsTypeOptions {
  "template" = "template",
  "default" = "default",
  "manual" = "manual",
  "custom" = "custom",
}
export type ImageTagsRecord = {
  description: string;
  isAlbum?: boolean;
  name: string;
  project: RecordIdString;
  type: ImageTagsTypeOptions;
};

export type ImagesRecord<TdownloadUrls = unknown, TexifData = unknown> = {
  camera: RecordIdString;
  capturedAt?: IsoDateString;
  capturedAtCorrected?: IsoDateString;
  computedFileName?: string;
  downloadUrls?: null | TdownloadUrls;
  exifData?: null | TexifData;
  fileName: string;
  imageTagAssignments?: RecordIdString[];
  project: RecordIdString;
  size: number;
  width?: number;
  height?: number;
  storageId: string;
  upload: RecordIdString;
  user: RecordIdString;
};

export type InferencesRecord = {
  completitionTokens?: number;
  image: RecordIdString;
  promptTokens?: number;
  result?: string;
  success?: boolean;
};

export type ProjectAssignmentsRecord = {
  project: RecordIdString;
  role: RecordIdString;
  user: RecordIdString;
};

export type ProjectsRecord = {
  aiSystemMessage?: string;
  copyright: string;
  copyrightReference: string;
  description: string;
  locationCity: string;
  locationCode: string;
  locationName: string;
  name: string;
};

export type RolesRecord = {
  description: string;
  key: string;
};

export type TimeOffsetsRecord = {
  camera: RecordIdString;
  cameraTime: IsoDateString;
  serverTime: IsoDateString;
  timeOffset?: number;
};

export type UploadsRecord = {
  camera: RecordIdString;
  name: string;
  project: RecordIdString;
  user: RecordIdString;
};

export type UsersRecord = {
  active?: boolean;
  activeProject?: RecordIdString;
  avatar?: string;
  copyrightTag?: string;
  firstName: string;
  lastName: string;
  projectAssignments?: RecordIdString[];
  role?: RecordIdString;
};

// Response types include system fields and match responses from the PocketBase API
export type CamerasResponse<Texpand = unknown> = Required<CamerasRecord> & BaseSystemFields<Texpand>;
export type ImageTagAssignmentsResponse<Texpand = unknown> = Required<ImageTagAssignmentsRecord> & BaseSystemFields<Texpand>;
export type ImageTagsResponse<Texpand = unknown> = Required<ImageTagsRecord> & BaseSystemFields<Texpand>;
export type ImagesResponse<TdownloadUrls = unknown, TexifData = unknown, Texpand = unknown> = Required<ImagesRecord<TdownloadUrls, TexifData>> & BaseSystemFields<Texpand>;
export type InferencesResponse<Texpand = unknown> = Required<InferencesRecord> & BaseSystemFields<Texpand>;
export type ProjectAssignmentsResponse<Texpand = unknown> = Required<ProjectAssignmentsRecord> & BaseSystemFields<Texpand>;
export type ProjectsResponse<Texpand = unknown> = Required<ProjectsRecord> & BaseSystemFields<Texpand>;
export type RolesResponse<Texpand = unknown> = Required<RolesRecord> & BaseSystemFields<Texpand>;
export type TimeOffsetsResponse<Texpand = unknown> = Required<TimeOffsetsRecord> & BaseSystemFields<Texpand>;
export type UploadsResponse<Texpand = unknown> = Required<UploadsRecord> & BaseSystemFields<Texpand>;
export type UsersResponse<Texpand = unknown> = Required<UsersRecord> & AuthSystemFields<Texpand>;

// Types containing all Records and Responses, useful for creating typing helper functions

export type CollectionRecords = {
  cameras: CamerasRecord;
  image_tag_assignments: ImageTagAssignmentsRecord;
  image_tags: ImageTagsRecord;
  images: ImagesRecord;
  inferences: InferencesRecord;
  project_assignments: ProjectAssignmentsRecord;
  projects: ProjectsRecord;
  roles: RolesRecord;
  time_offsets: TimeOffsetsRecord;
  uploads: UploadsRecord;
  users: UsersRecord;
};

export type CollectionResponses = {
  cameras: CamerasResponse;
  image_tag_assignments: ImageTagAssignmentsResponse;
  image_tags: ImageTagsResponse;
  images: ImagesResponse;
  inferences: InferencesResponse;
  project_assignments: ProjectAssignmentsResponse;
  projects: ProjectsResponse;
  roles: RolesResponse;
  time_offsets: TimeOffsetsResponse;
  uploads: UploadsResponse;
  users: UsersResponse;
};

// Type for usage with type asserted PocketBase instance
// https://github.com/pocketbase/js-sdk#specify-typescript-definitions

export type TypedPocketBase = PocketBase & {
  collection(idOrName: "cameras"): RecordService<CamerasResponse>;
  collection(idOrName: "image_tag_assignments"): RecordService<ImageTagAssignmentsResponse>;
  collection(idOrName: "image_tags"): RecordService<ImageTagsResponse>;
  collection(idOrName: "images"): RecordService<ImagesResponse>;
  collection(idOrName: "inferences"): RecordService<InferencesResponse>;
  collection(idOrName: "project_assignments"): RecordService<ProjectAssignmentsResponse>;
  collection(idOrName: "projects"): RecordService<ProjectsResponse>;
  collection(idOrName: "roles"): RecordService<RolesResponse>;
  collection(idOrName: "time_offsets"): RecordService<TimeOffsetsResponse>;
  collection(idOrName: "uploads"): RecordService<UploadsResponse>;
  collection(idOrName: "users"): RecordService<UsersResponse>;
};
