import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";
import { useUserStore } from "./user-store";
import { refreshToken, ResponseCode as AuthorizationResponseCode } from "src/api/authorization";
import { emitter } from "src/boot/mitt";

// Interval determines how often the token refresh is checked
// If the token is about to expire, it will be refreshed
const TOKEN_CHECK_INTERVAL_TIME = 1000 * 5; // 5 seconds
// Threshold determines how early the token refresh is triggered
const TOKEN_REFRESH_THRESHOLD = 1000 * 60 * 2; // 2 Minutes

export const useLoginStore = defineStore("login", {
  state: () => ({
    // interval for checking token refresh
    interval: null as NodeJS.Timeout | null,
    // expiration time of the refresh token cookie
    refreshTokenExpiration: useStorage("login/refreshTokenExpiration", 0),
    // expiration time of the auth token cookie
    authTokenExpiration: useStorage("login/authTokenExpiration", 0),
    // whether the user is logged in
    loggedIn: useStorage("login/loggedIn", false),
  }),
  getters: {
    isLoggedIn(state): boolean {
      return state.loggedIn;
    },
  },
  actions: {
    clearInterval() {
      if (this.interval !== null) {
        clearInterval(this.interval);
        this.interval = null;
      }
    },
    async startTokenRefresh() {
      this.clearInterval();
      await this.checkTokenRefresh();
      this.interval = setInterval(this.checkTokenRefresh, TOKEN_CHECK_INTERVAL_TIME);
    },
    async checkTokenRefresh() {
      if (!this.loggedIn) {
        // not logged in, still we are here...
        // we takled about this
        // no interval if we are not logged in
        // yet, every time you call this function
        // I am putting an end to it now, once and for all
        this.clearInterval();
        return;
      }
      if (this.refreshTokenExpiration === 0 || this.authTokenExpiration === 0 || new Date().getTime() > this.refreshTokenExpiration) {
        // refresh token expired or auth token exipry is not set correctly
        this.clearInterval();
        this.setLoggedOut();
        return;
      }

      if (
        this.refreshTokenExpiration - new Date().getTime() < TOKEN_REFRESH_THRESHOLD ||
        this.authTokenExpiration - new Date().getTime() < TOKEN_REFRESH_THRESHOLD ||
        this.refreshTokenExpiration === 0 ||
        this.authTokenExpiration === 0 ||
        isNaN(this.refreshTokenExpiration) ||
        isNaN(this.authTokenExpiration)
      ) {
        const response = await refreshToken();
        const authErrors = [
          AuthorizationResponseCode.INVALID_TOKEN,
          AuthorizationResponseCode.MISSING_TOKEN,
          AuthorizationResponseCode.INVALID_REFRESH_TOKEN,
          AuthorizationResponseCode.EXPIRED_TOKEN,
        ];
        if (authErrors.includes(response.code)) {
          this.setLoggedOut();
          return;
        }
      }
    },
    async setLoggedOut() {
      this.loggedIn = false;
      useUserStore().setUser(null);
      if (this.interval !== null) {
        clearInterval(this.interval);
      }
      this.authTokenExpiration = 0;
      this.refreshTokenExpiration = 0;
      emitter.emit("pushToLogin");
      emitter.emit("logout");
    },
    async setLoggedIn() {
      this.loggedIn = true;
      this.startTokenRefresh();
      await useUserStore().refreshUser();
      emitter.emit("login");
    },
  },
});
