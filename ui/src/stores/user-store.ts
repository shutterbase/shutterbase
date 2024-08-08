import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";
import { ImageTagsResponse, ProjectsResponse, UsersResponse } from "src/types/pocketbase";
import { UserWithProjectAssignmentsType } from "src/types/custom";
import { SORT_ORDER } from "src/components/image/ImagesHeader.vue";
import pb from "src/boot/pocketbase";

const PROJECT_TAG_FETCH_INTERVAL = 1000 * 30;
let projectTagInterval: NodeJS.Timeout;

const USER_FETCH_INTERVAL = 1000 * 60;
let userInterval: NodeJS.Timeout;

const MAX_TAG_STACK_SIZE = 10;

export const useUserStore = defineStore("user", {
  state: () => ({
    activeProjectId: useStorage("activeProjectId", ""),
    activeProject: useStorage("activeProject", {} as ProjectsResponse),
    preferredImageSortOrder: useStorage("preferredImageSortOrder", SORT_ORDER.LATEST_FIRST),
    projectTags: useStorage("projectTags", [] as ImageTagsResponse[]),
    tagStack: useStorage("tagStack", [] as ImageTagsResponse[]),
    user: useStorage("user", {} as UserWithProjectAssignmentsType),
  }),
  getters: {},
  actions: {
    setProject(project: ProjectsResponse) {
      this.activeProjectId = project.id;
      this.activeProject = project;
      this.projectTags = [];
      this.tagStack = [];
      this.loadProjectTags();
    },
    clearActiveProject() {
      this.activeProjectId = "";
      this.activeProject = {} as ProjectsResponse;
      this.projectTags = [];
    },
    async loadProjectTags() {
      if (!this.activeProjectId) return;
      const response = await pb.collection<ImageTagsResponse>("image_tags").getList(1, 10000, {
        filter: `(project='${this.activeProjectId}')`,
      });
      this.projectTags = response.items;
    },
    startProjectTagFetching() {
      if (projectTagInterval) return;
      projectTagInterval = setInterval(this.loadProjectTags, PROJECT_TAG_FETCH_INTERVAL);
    },
    stopProjectTagFetching() {
      if (projectTagInterval) clearInterval(projectTagInterval);
    },
    addTagToStack(tag: ImageTagsResponse) {
      this.tagStack = this.tagStack.filter((t) => t.id !== tag.id);
      this.tagStack.push(tag);
      if (this.tagStack.length > MAX_TAG_STACK_SIZE) {
        this.tagStack.shift();
      }
    },
    async loadUser() {
      if (!pb.authStore.isValid || pb.authStore.model === null) return;
      const user = await pb.collection<UserWithProjectAssignmentsType>("users").getOne(pb.authStore.model.id, { expand: "role, projectAssignments, projectAssignments.role" });
      if (user) {
        this.user = user;
      }
    },
    startUserFetching() {
      if (userInterval) return;
      userInterval = setInterval(this.loadUser, USER_FETCH_INTERVAL);
    },
    stopUserFetching() {
      if (userInterval) clearInterval(userInterval);
    },
    hasUserProjectRole(roleKey: string, projectId?: string) {
      if (!projectId) projectId = this.activeProjectId;
      if (!projectId) return false;
      if (!this.user.projectAssignments) return false;
      for (const assignment of this.user.expand?.projectAssignments || []) {
        if (assignment.project === projectId && assignment.expand?.role?.key === roleKey) {
          return true;
        }
      }
      return false;
    },
    isAdmin() {
      const roleKey = this.user?.expand?.role?.key;
      if (!roleKey) return false;
      return roleKey === "admin";
    },
    isProjectAdmin() {
      const hasProjectRole = this.hasUserProjectRole("projectAdmin");
      return hasProjectRole;
    },
    isProjectEditor() {
      const hasProjectRole = this.hasUserProjectRole("projectEditor");
      return hasProjectRole;
    },
    isProjectAdminOrHigher() {
      const isProjectAdmin = this.isProjectAdmin();
      const isAdmin = this.isAdmin();
      return isProjectAdmin || isAdmin;
    },
    isProjectEditorOrHigher() {
      const isProjectEditor = this.isProjectEditor();
      const isProjectAdminOrHigher = this.isProjectAdminOrHigher();
      return isProjectEditor || isProjectAdminOrHigher;
    },
  },
});
