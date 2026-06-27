// Central data seam: components call api.<resource>.<method>, never axios directly.
import * as images from "./images";
import * as imageTags from "./imageTags";
import * as imageTagAssignments from "./imageTagAssignments";
import * as projects from "./projects";
import * as projectAssignments from "./projectAssignments";
import * as cameras from "./cameras";
import * as uploads from "./uploads";
import * as timeOffsets from "./timeOffsets";
import * as roles from "./roles";
import * as users from "./users";
import * as auth from "./auth";
import * as statistics from "./statistics";

export const api = {
  images,
  imageTags,
  imageTagAssignments,
  projects,
  projectAssignments,
  cameras,
  uploads,
  timeOffsets,
  roles,
  users,
  auth,
  statistics,
};

export default api;
