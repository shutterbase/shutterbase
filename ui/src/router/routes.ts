import { RouteRecordRaw } from "vue-router";
import pb from "src/boot/pocketbase";
import { projectRoutes } from "src/router/routes/project";
import { userRoutes } from "src/router/routes/user";

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    component: () => import("layouts/MainLayout.vue"),
    children: [
      {
        name: "index",
        path: "",
        component: () => import("pages/IndexPage.vue"),
      },
      ...projectRoutes,
      ...userRoutes,
      {
        name: "sandbox",
        path: "/sandbox",
        component: () => import("pages/Sandbox.vue"),
      },
    ],
  },
  {
    name: "login",
    path: "/login",
    component: () => import("pages/Login.vue"),
    beforeEnter: (to, from, next) => {
      if (pb.authStore.isValid) {
        next({ name: "index" });
      } else {
        next();
      }
    },
  },
  {
    name: "logout",
    path: "/logout",
    component: () => import("pages/Logout.vue"),
  },
  {
    name: "signup",
    path: "/signup",
    component: () => import("pages/Signup.vue"),
  },

  // Always leave this as last one,
  // but you can also remove it
  {
    path: "/:catchAll(.*)*",
    component: () => import("pages/ErrorNotFound.vue"),
  },
];

export default routes;
