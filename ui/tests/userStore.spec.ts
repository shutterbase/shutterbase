import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { CurrentUser } from "src/types/api";

const { auth, imageTags, usersApi } = vi.hoisted(() => ({
  auth: {
    me: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    changePassword: vi.fn(),
    impersonate: vi.fn(),
    stopImpersonate: vi.fn(),
  },
  imageTags: { list: vi.fn() },
  usersApi: { setActiveProject: vi.fn() },
}));

vi.mock("src/api", () => ({
  api: { auth, imageTags, users: usersApi },
}));

import { useUserStore } from "src/stores/user-store";

function fakeUser(overrides: Partial<CurrentUser> = {}): CurrentUser {
  return {
    id: "u1",
    username: "max",
    email: "max@x.de",
    verified: true,
    active: true,
    firstName: "Max",
    lastName: "P",
    copyrightTag: "MP",
    forcePasswordChange: false,
    totpEnabled: false,
    role: { id: "r", key: "admin", description: "Admin" },
    activeProject: null,
    projectAssignments: [],
    createdAt: "",
    updatedAt: "",
    ...overrides,
  };
}

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  vi.clearAllMocks();
});

describe("user store (cookie session)", () => {
  it("is unauthenticated before any load", () => {
    const store = useUserStore();
    expect(store.isAuthenticated).toBe(false);
  });

  it("login() sets the effective user and active project", async () => {
    auth.login.mockResolvedValue(fakeUser({ activeProject: { id: "p1", name: "Cup" } }));
    const store = useUserStore();
    const user = await store.login("max@x.de", "pw");
    expect(auth.login).toHaveBeenCalledWith({ identifier: "max@x.de", password: "pw" });
    expect(store.isAuthenticated).toBe(true);
    expect(store.user?.id).toBe("u1");
    expect(store.activeProjectId).toBe("p1");
    expect(user.id).toBe("u1");
  });

  it("load() pulls /users/me into the store", async () => {
    auth.me.mockResolvedValue(fakeUser());
    const store = useUserStore();
    await store.load();
    expect(auth.me).toHaveBeenCalled();
    expect(store.user?.username).toBe("max");
    expect(store.isAdmin()).toBe(true);
  });

  it("loadUser() swallows a 401 and stays unauthenticated", async () => {
    auth.me.mockRejectedValue({ response: { status: 401 } });
    const store = useUserStore();
    await store.loadUser();
    expect(store.isAuthenticated).toBe(false);
  });

  it("logout() clears the user even though the cookie call resolves", async () => {
    auth.login.mockResolvedValue(fakeUser());
    auth.logout.mockResolvedValue(undefined);
    const store = useUserStore();
    await store.login("max@x.de", "pw");
    await store.logout();
    expect(auth.logout).toHaveBeenCalled();
    expect(store.user).toBeNull();
    expect(store.isAuthenticated).toBe(false);
  });

  it("exposes the impersonating block when present", async () => {
    auth.me.mockResolvedValue(fakeUser({ impersonating: { realUserId: "admin1", realUserName: "Boss" } }));
    const store = useUserStore();
    await store.load();
    expect(store.isImpersonating).toBe(true);
    expect(store.impersonating?.realUserName).toBe("Boss");
  });

  it("resolves project roles from assignments", async () => {
    auth.me.mockResolvedValue(
      fakeUser({
        role: { id: "r", key: "user", description: "User" },
        projectAssignments: [
          { id: "a1", project: { id: "p1", name: "Cup" }, user: { id: "u1", firstName: "Max", lastName: "P" }, role: { id: "re", key: "projectEditor", description: "" }, createdAt: "", updatedAt: "" },
        ],
      })
    );
    const store = useUserStore();
    await store.load();
    store.activeProjectId = "p1";
    expect(store.isProjectEditor()).toBe(true);
    expect(store.isProjectAdmin()).toBe(false);
    expect(store.isProjectEditorOrHigher()).toBe(true);
  });
});
