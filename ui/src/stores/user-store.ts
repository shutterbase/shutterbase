import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";
import { ImageTagsResponse, ProjectsResponse } from "src/types/pocketbase";
import { SORT_ORDER } from "src/components/image/ImagesHeader.vue";
import pb from "src/boot/pocketbase";

const PROJECT_TAG_FETCH_INTERVAL = 1000 * 30;
let interval: NodeJS.Timeout;

export const useUserStore = defineStore("user", {
  state: () => ({
    activeProjectId: useStorage("activeProjectId", ""),
    activeProject: useStorage("activeProject", {} as ProjectsResponse),
    preferredImageSortOrder: useStorage("preferredImageSortOrder", SORT_ORDER.LATEST_FIRST),
    projectTags: useStorage("projectTags", [] as ImageTagsResponse[]),
  }),
  getters: {},
  actions: {
    setProject(project: ProjectsResponse) {
      this.activeProjectId = project.id;
      this.activeProject = project;
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
      if (interval) return;
      interval = setInterval(this.loadProjectTags, PROJECT_TAG_FETCH_INTERVAL);
    },
    stopProjectTagFetching() {
      if (interval) clearInterval(interval);
    },
  },
});
