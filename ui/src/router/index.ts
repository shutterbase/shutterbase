import { route } from "quasar/wrappers";
import { RouteRecordName, createMemoryHistory, createRouter, createWebHashHistory, createWebHistory } from "vue-router";

import routes from "./routes";

import { useUserStore } from "src/stores/user-store";
import { emitter } from "src/boot/mitt";

/*
 * If not building with SSR mode, you can
 * directly export the Router instantiation;
 *
 * The function below can be async too; either use
 * async/await or return a Promise which resolves
 * with the Router instance.
 */

export default route(function (/* { store, ssrContext } */) {
  const createHistory = process.env.SERVER ? createMemoryHistory : process.env.VUE_ROUTER_MODE === "history" ? createWebHistory : createWebHashHistory;

  const Router = createRouter({
    // Query-only navigation (e.g. the image grid's view/filter state) must not
    // touch the scroll position; real page changes start at the top.
    scrollBehavior: (to, from) => (to.path === from.path ? false : { left: 0, top: 0 }),
    routes,

    // Leave this as is and make changes in quasar.conf.js instead!
    // quasar.conf.js -> build -> vueRouterMode
    // quasar.conf.js -> build -> publicPath
    history: createHistory(process.env.VUE_ROUTER_BASE),
  });

  const PUBLIC_PAGES = ["login", "signup", "about", "sandbox"] as RouteRecordName[];
  Router.beforeEach(async (to, from) => {
    emitter.emit("router:change", { to, from });
    const toName = to.name || "";
    if (PUBLIC_PAGES.includes(toName)) {
      return;
    }
    const userStore = useUserStore();
    if (!userStore.isAuthenticated) {
      // Cookie session may exist server-side even though the SPA just booted —
      // probe /users/me once before bouncing to login.
      try {
        await userStore.load();
      } catch {
        // not authenticated
      }
    }
    if (!userStore.isAuthenticated) {
      // carry the target (e.g. a shared image permalink) through the login flow
      return { name: "login", query: to.fullPath === "/" ? {} : { redirect: to.fullPath } };
    }
    // Enforce password rotation on every navigation, not just post-login
    // (Login.vue). A refresh/deep-link/restored session with forcePasswordChange
    // still set otherwise sails past this guard and 403s every API call the page
    // fires. Mirrors the backend forcePasswordChangeMiddleware.
    if (userStore.user?.forcePasswordChange && toName !== "change-password") {
      return { name: "change-password", query: to.fullPath === "/" ? {} : { redirect: to.fullPath } };
    }
  });

  return Router;
});
