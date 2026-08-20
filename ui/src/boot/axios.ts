import { boot } from "quasar/wrappers";
import axios, { AxiosInstance } from "axios";

export const API_BASE = "/api/v1";

// Single axios instance: cookie-session auth (withCredentials), base /api/v1.
export const http: AxiosInstance = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
});

// 401 -> bounce to /login (except while already there, to avoid a redirect
// loop), carrying the current page as ?redirect= so a deep link (e.g. a shared
// image permalink) or an expired mid-session view survives the round trip.
http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401) {
      const path = window.location.pathname;
      if (!path.startsWith("/login")) {
        const target = path + window.location.search;
        window.location.assign(target === "/" ? "/login" : `/login?redirect=${encodeURIComponent(target)}`);
      }
    }
    return Promise.reject(error);
  },
);

// WebSocket lives at /ws (not under /api/v1), cookie-authenticated.
export function websocketUrl(): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws`;
}

export default boot(({ app }) => {
  app.config.globalProperties.$http = http;
});
