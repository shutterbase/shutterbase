import { route } from "quasar/wrappers";
import { RouteRecordName, createMemoryHistory, createRouter, createWebHashHistory, createWebHistory } from "vue-router";

import routes from "./routes";

import pb from "src/boot/pocketbase";
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
    scrollBehavior: () => ({ left: 0, top: 0 }),
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
    if (!pb.authStore.isValid && !PUBLIC_PAGES.includes(toName)) {
      return { name: "login" };
    }
  });

  return Router;
});
