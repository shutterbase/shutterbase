// Compatibility shim mapping the old PB "expand"-shaped aliases onto the REST
// contract types (src/types/api). New code should import from src/types/api.
import type { Image, ImageTagAssignment, Project, ImageTag, CurrentUser, ProjectAssignment } from "src/types/api";

export type DownloadUrls = {
  256: string;
  512: string;
  1024: string;
  2048: string;
  original: string;
};

// Image now carries embedded user/camera/project/upload and a `tags[]` array of
// assignments — no PB `expand`.
export type ImageWithTagsType = Image;
export type ImageTagAssignmentType = ImageTagAssignment;
export type ProjectWithTagsType = Project & { tags?: ImageTag[] };
export type UserWithProjectAssignmentsType = CurrentUser;
export type ProjectAssignmentWithRoleType = ProjectAssignment;
