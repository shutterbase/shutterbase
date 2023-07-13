import { defineStore } from "pinia";
import { loadOwnUser, User } from "src/api/user";
import { useStorage } from "@vueuse/core";
import { useLoginStore } from "./login-store";
import { Ref, ref } from "vue";

const USER_REFRESH_INTERVAL_TIME = 1000 * 60; // 1 minute

// TODO: switch to SetupStores https://pinia.vuejs.org/core-concepts/#setup-stores
// TODO: refactor to `storeToRefs(...)` https://pinia.vuejs.org/core-concepts/#using-the-store

export const useUserStore = defineStore("user", () => {
  const ownUserJson: Ref<string> = useStorage("user/ownUserJson", "" as string);
  const ownUserRefreshInterval: Ref<NodeJS.Timeout | undefined> = ref(undefined) as Ref<NodeJS.Timeout | undefined>;
  const ownUser = (): User | null => {
    if (ownUserJson.value === "") return null;
    return JSON.parse(ownUserJson.value);
  };

  const isAdmin = (): boolean => {
    return ownUser()?.edges?.role?.key === "admin" || false;
  };

  const setUser = (user: User | null) => {
    if (!user) {
      ownUserJson.value = "";
      return;
    }
    console.log(user);
    ownUserJson.value = JSON.stringify(user);
  };

  const refreshUser = async () => {
    if (!useLoginStore().isLoggedIn) return;
    const loadOwnUserResult = await loadOwnUser();
    if (loadOwnUserResult.response.status !== 200) {
      console.log(`Error loading own user into store: ${loadOwnUserResult.response.code}`);
      if (loadOwnUserResult.response.code === "UNAUTHORIZED" || loadOwnUserResult.response.code === "TOKEN_INVALID") {
        useLoginStore().setLoggedOut();
      }
      return;
    }
    setUser(loadOwnUserResult.item || null);
  };
  const startUserRefresh = () => {
    if (!useLoginStore().isLoggedIn) return;
    if (ownUserRefreshInterval.value !== null) {
      // @ts-ignore
      clearInterval(ownUserRefreshInterval.value);
    }
    ownUserRefreshInterval.value = setInterval(refreshUser, USER_REFRESH_INTERVAL_TIME);
  };

  return {
    ownUser,
    ownUserJson,
    refreshUser,
    isAdmin,
    setUser,
    startUserRefresh,
  };
});
