// Central data seam: components call api.<resource>.<method>, never axios directly.
import * as ai from "./ai";
import * as images from "./images";
import * as imageTags from "./imageTags";
import * as imageTagAssignments from "./imageTagAssignments";
import * as projects from "./projects";
import * as projectAssignments from "./projectAssignments";
import * as cameras from "./cameras";
import * as uploads from "./uploads";
import * as scheduleItems from "./scheduleItems";
import * as timeOffsets from "./timeOffsets";
import * as roles from "./roles";
import * as users from "./users";
import * as auth from "./auth";
import * as statistics from "./statistics";
import * as apiKeys from "./apiKeys";
import * as downloadConfigs from "./downloadConfigs";

export const api = {
  ai,
  images,
  imageTags,
  imageTagAssignments,
  projects,
  projectAssignments,
  cameras,
  uploads,
  scheduleItems,
  timeOffsets,
  roles,
  users,
  auth,
  statistics,
  apiKeys,
  downloadConfigs,
};

export default api;
