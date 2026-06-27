// Compatibility shim. The PocketBase SDK and its generated types are gone (S13).
// Old type names are re-pointed at the REST contract types in ./api so that
// type-only imports across the app keep resolving. New code should import from
// "src/types/api" directly.
import type { Camera, ImageTag, ImageTagAssignment, Image, ProjectAssignment, Project, Role, TimeOffset, Upload, User } from "src/types/api";

export type CamerasResponse = Camera;
export type ImageTagAssignmentsResponse = ImageTagAssignment;
export type ImageTagsResponse = ImageTag;
export type ImagesResponse = Image;
export type ProjectAssignmentsResponse = ProjectAssignment;
export type ProjectsResponse = Project;
export type RolesResponse = Role;
export type TimeOffsetsResponse = TimeOffset;
export type UploadsResponse = Upload;
export type UsersResponse = User;

// "Record" types = create/update payload shapes (loose).
export type CamerasRecord = Partial<Camera>;
export type ImageTagsRecord = Partial<ImageTag>;
export type ImagesRecord = Partial<Image>;
export type TimeOffsetsRecord = {
  camera?: string;
  cameraTime: string;
  serverTime: string;
  timeOffset?: number;
};

export enum ImageTagsTypeOptions {
  template = "template",
  default = "default",
  manual = "manual",
  custom = "custom",
}

export enum ImageTagAssignmentsTypeOptions {
  manual = "manual",
  inferred = "inferred",
  default = "default",
}
