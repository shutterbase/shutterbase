import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";
import { ProjectsResponse } from "src/types/pocketbase";
import { SORT_ORDER } from "src/components/image/ImagesHeader.vue";

export const useUserStore = defineStore("user", {
  state: () => ({
    activeProjectId: useStorage("activeProjectId", ""),
    activeProject: useStorage("activeProject", {} as ProjectsResponse),
    preferredImageSortOrder: useStorage("preferredImageSortOrder", SORT_ORDER.LATEST_FIRST),
  }),
  getters: {},
  actions: {
    clearActiveProject() {
      this.activeProjectId = "";
      this.activeProject = {} as ProjectsResponse;
    },
  },
});
