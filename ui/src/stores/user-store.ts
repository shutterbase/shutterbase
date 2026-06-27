import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";
import { api } from "src/api";
import type { ChangePasswordBody } from "src/api/auth";
import { CurrentUser, EmbeddedProject, ImageTag, Impersonating } from "src/types/api";
import { SORT_ORDER } from "src/components/image/sortOrder";

const PROJECT_TAG_FETCH_INTERVAL = 1000 * 30;
let projectTagInterval: ReturnType<typeof setInterval> | undefined;

const USER_FETCH_INTERVAL = 1000 * 60;
let userInterval: ReturnType<typeof setInterval> | undefined;

const MAX_TAG_STACK_SIZE = 10;

export const useUserStore = defineStore("user", {
  state: () => ({
    activeProjectId: useStorage("activeProjectId", ""),
    activeProject: useStorage("activeProject", {} as EmbeddedProject),
    preferredImageSortOrder: useStorage("preferredImageSortOrder", SORT_ORDER.LATEST_FIRST),
    projectTags: useStorage("projectTags", [] as ImageTag[]),
    tagStack: useStorage("tagStack", [] as ImageTag[]),
    // user lives in memory only — the cookie session is the source of truth.
    user: null as CurrentUser | null,
  }),
  getters: {
    isAuthenticated: (state): boolean => !!state.user?.id,
    isImpersonating: (state): boolean => !!state.user?.impersonating,
    impersonating: (state): Impersonating | undefined => state.user?.impersonating,
  },
  actions: {
    setProject(project: EmbeddedProject) {
      this.activeProjectId = project.id;
      this.activeProject = { id: project.id, name: project.name };
      this.projectTags = [];
      this.tagStack = [];
      this.loadProjectTags();
    },
    clearActiveProject() {
      this.activeProjectId = "";
      this.activeProject = {} as EmbeddedProject;
      this.projectTags = [];
    },
    async loadProjectTags() {
      if (!this.activeProjectId) return;
      const response = await api.imageTags.list({ projectId: this.activeProjectId, limit: 500 });
      this.projectTags = response.items;
    },
    startProjectTagFetching() {
      if (projectTagInterval) return;
      projectTagInterval = setInterval(() => this.loadProjectTags(), PROJECT_TAG_FETCH_INTERVAL);
    },
    stopProjectTagFetching() {
      if (projectTagInterval) clearInterval(projectTagInterval);
      projectTagInterval = undefined;
    },
    addTagToStack(tag: ImageTag) {
      this.tagStack = this.tagStack.filter((t) => t.id !== tag.id);
      this.tagStack.push(tag);
      if (this.tagStack.length > MAX_TAG_STACK_SIZE) {
        this.tagStack.shift();
      }
    },
    // GET /users/me — sets the effective user (role + assignments + impersonating).
    async load(): Promise<CurrentUser> {
      const user = await api.auth.me();
      this.user = user;
      if (user.activeProject) {
        this.activeProjectId = user.activeProject.id;
        this.activeProject = user.activeProject;
      }
      return user;
    },
    // kept for backwards compatibility with existing callers
    async loadUser(): Promise<void> {
      try {
        await this.load();
      } catch {
        this.user = null;
      }
    },
    async login(identifier: string, password: string): Promise<CurrentUser> {
      const user = await api.auth.login({ identifier, password });
      this.user = user;
      if (user.activeProject) {
        this.activeProjectId = user.activeProject.id;
        this.activeProject = user.activeProject;
      }
      return user;
    },
    async logout(): Promise<void> {
      try {
        await api.auth.logout();
      } finally {
        this.user = null;
        this.clearActiveProject();
        this.tagStack = [];
      }
    },
    async changePassword(body: ChangePasswordBody): Promise<CurrentUser> {
      const user = await api.auth.changePassword(body);
      this.user = user;
      return user;
    },
    async impersonate(userId: string): Promise<CurrentUser> {
      const user = await api.auth.impersonate(userId);
      this.user = user;
      return user;
    },
    async stopImpersonate(): Promise<CurrentUser> {
      const user = await api.auth.stopImpersonate();
      this.user = user;
      return user;
    },
    startUserFetching() {
      if (userInterval) return;
      userInterval = setInterval(() => this.loadUser(), USER_FETCH_INTERVAL);
    },
    stopUserFetching() {
      if (userInterval) clearInterval(userInterval);
      userInterval = undefined;
    },
    hasUserProjectRole(roleKey: string, projectId?: string) {
      if (!projectId) projectId = this.activeProjectId;
      if (!projectId) return false;
      for (const assignment of this.user?.projectAssignments || []) {
        if (assignment.project?.id === projectId && assignment.role?.key === roleKey) {
          return true;
        }
      }
      return false;
    },
    isAdmin() {
      return this.user?.role?.key === "admin";
    },
    isProjectAdmin() {
      return this.hasUserProjectRole("projectAdmin");
    },
    isProjectEditor() {
      return this.hasUserProjectRole("projectEditor");
    },
    isProjectAdminOrHigher() {
      return this.isProjectAdmin() || this.isAdmin();
    },
    isProjectEditorOrHigher() {
      return this.isProjectEditor() || this.isProjectAdminOrHigher();
    },
  },
});
