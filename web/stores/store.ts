import { defineStore } from "pinia";
import { refreshToken, ResponseCode as AuthorizationResponseCode } from "~/api/authorization";
import { emitter } from "~/boot/mitt";
import { User, loadOwnUser } from "~/api/user";

// Interval determines how often the token refresh is checked
// If the token is about to expire, it will be refreshed
const TOKEN_CHECK_INTERVAL_TIME = 1000 * 5; // 5 seconds
// Threshold determines how early the token refresh is triggered
const TOKEN_REFRESH_THRESHOLD = 1000 * 60 * 2; // 2 Minutes

// Interval determines how often the user is refreshed
const USER_REFRESH_INTERVAL_TIME = 1000 * 60; // 1 minute

export const useStore = defineStore(
  "store",
  () => {
    // interval for checking token refresh
    const tokenRefreshInterval: Ref<NodeJS.Timeout | null> = ref(null);

    // expiration time of the refresh token cookie
    const refreshTokenExpiration: Ref<number> = ref(0);

    // expiration time of the auth token cookie
    const authTokenExpiration: Ref<number> = ref(0);

    // whether the user is logged in
    const loggedIn: Ref<boolean> = ref(false);

    const ownUserJson: Ref<string> = ref("");
    const ownUserRefreshInterval: Ref<NodeJS.Timeout | undefined> = ref(undefined) as Ref<NodeJS.Timeout | undefined>;

    function getOwnUser(): User | null {
      if (ownUserJson.value === "") return null;
      return JSON.parse(ownUserJson.value);
    }

    function isAdmin(): boolean {
      return getOwnUser()?.edges?.role?.key === "admin" || false;
    }

    function setUser(user: User | null): void {
      if (!user) {
        ownUserJson.value = "";
        return;
      }
      ownUserJson.value = JSON.stringify(user);
    }

    async function refreshOwnUser(): Promise<void> {
      if (!isLoggedIn()) return;
      const loadOwnUserResult = await loadOwnUser();
      if (!loadOwnUserResult.response.ok) {
        console.log(`Error loading own user into store: ${loadOwnUserResult.response.message}`);
        if (loadOwnUserResult.response.code === "UNAUTHORIZED" || loadOwnUserResult.response.code === "TOKEN_INVALID") {
          setLoggedOut();
        }
        return;
      }
      setUser(loadOwnUserResult.item || null);
    }

    async function startOwnUserRefresh(): Promise<void> {
      if (!isLoggedIn()) return;
      if (ownUserRefreshInterval.value !== null) {
        clearInterval(ownUserRefreshInterval.value);
      }
      await refreshOwnUser();
      ownUserRefreshInterval.value = setInterval(refreshOwnUser, USER_REFRESH_INTERVAL_TIME);
    }

    function isLoggedIn(): boolean {
      return loggedIn.value;
    }

    function resetTokenRefreshInterval() {
      if (tokenRefreshInterval.value !== null) {
        clearInterval(tokenRefreshInterval.value);
        tokenRefreshInterval.value = null;
      }
    }

    async function startTokenRefresh() {
      resetTokenRefreshInterval();
      await checkTokenRefresh();
      tokenRefreshInterval.value = setInterval(checkTokenRefresh, TOKEN_CHECK_INTERVAL_TIME);
    }

    async function checkTokenRefresh() {
      if (!loggedIn.value) {
        // not logged in, still we are here...
        // we takled about this
        // no interval if we are not logged in
        // yet, every time you call this function
        // I am putting an end to it now, once and for all
        resetTokenRefreshInterval();
        return;
      }
      if (refreshTokenExpiration.value === 0 || authTokenExpiration.value === 0 || new Date().getTime() > refreshTokenExpiration.value) {
        // refresh token expired or auth token exipry is not set correctly
        resetTokenRefreshInterval();
        setLoggedOut();
        return;
      }

      if (
        refreshTokenExpiration.value - new Date().getTime() < TOKEN_REFRESH_THRESHOLD ||
        authTokenExpiration.value - new Date().getTime() < TOKEN_REFRESH_THRESHOLD ||
        refreshTokenExpiration.value === 0 ||
        authTokenExpiration.value === 0 ||
        isNaN(refreshTokenExpiration.value) ||
        isNaN(authTokenExpiration.value)
      ) {
        const response = await refreshToken();
        const authErrors = [
          AuthorizationResponseCode.INVALID_TOKEN,
          AuthorizationResponseCode.MISSING_TOKEN,
          AuthorizationResponseCode.INVALID_REFRESH_TOKEN,
          AuthorizationResponseCode.EXPIRED_TOKEN,
        ];
        if (authErrors.includes(response.code)) {
          setLoggedOut();
          return;
        }
      }
    }

    async function startIntervals() {
      await startTokenRefresh();
      await startOwnUserRefresh();
    }

    async function setLoggedOut() {
      loggedIn.value = false;

      setUser(null);
      resetTokenRefreshInterval();

      authTokenExpiration.value = 0;
      refreshTokenExpiration.value = 0;
      emitter.emit("logout");
    }

    async function setLoggedIn() {
      loggedIn.value = true;
      startTokenRefresh();

      await refreshOwnUser();
      emitter.emit("login");
    }

    return {
      getOwnUser,
      ownUserJson,
      isAdmin,
      startIntervals,
      authTokenExpiration,
      refreshTokenExpiration,
      loggedIn,
      isLoggedIn,
      setLoggedIn,
      setLoggedOut,
    };
  },
  { persist: true }
);
