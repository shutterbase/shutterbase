import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";
import { ProjectsResponse } from "src/types/pocketbase";

export const useUserStore = defineStore("user", {
  state: () => ({
    activeProjectId: useStorage("activeProjectId", ""),
    activeProject: useStorage("activeProject", {} as ProjectsResponse),
  }),
  getters: {},
  actions: {
    clearActiveProject() {
      this.activeProjectId = "";
      this.activeProject = {} as ProjectsResponse;
    },
  },
});
