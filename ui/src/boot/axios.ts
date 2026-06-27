import { boot } from "quasar/wrappers";
import axios, { AxiosInstance } from "axios";

export const API_BASE = "/api/v1";

// Single axios instance: cookie-session auth (withCredentials), base /api/v1.
export const http: AxiosInstance = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
});

// 401 -> bounce to /login (except while already there, to avoid a redirect loop).
http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401) {
      const path = window.location.pathname;
      if (!path.startsWith("/login")) {
        window.location.assign("/login");
      }
    }
    return Promise.reject(error);
  }
);

// WebSocket lives at /ws (not under /api/v1), cookie-authenticated.
export function websocketUrl(): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws`;
}

export default boot(({ app }) => {
  app.config.globalProperties.$http = http;
});
