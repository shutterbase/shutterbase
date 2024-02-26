import { RouteRecordRaw } from "vue-router";
import pb from "src/boot/pocketbase";

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
      {
        name: "projects",
        path: "/projects",
        component: () => import("pages/project/Projects.vue"),
      },
      {
        name: "project-create",
        path: "/projects/create",
        component: () => import("pages/project/ProjectCreate.vue"),
      },
      {
        name: "project",
        path: "/projects/:id",
        component: () => import("pages/project/Project.vue"),
        children: [
          {
            name: "project-general",
            path: "general",
            component: () => import("pages/project/ProjectGeneral.vue"),
          },
          {
            name: "project-tags",
            path: "tags",
            component: () => import("pages/project/ProjectTags.vue"),
          },
          {
            name: "project-statistics",
            path: "statistics",
            component: () => import("pages/project/ProjectStatistics.vue"),
          },
          {
            name: "project-members",
            path: "members",
            component: () => import("pages/project/ProjectMembers.vue"),
          },
          {
            name: "project-danger-zone",
            path: "danger-zone",
            component: () => import("pages/project/ProjectDangerZone.vue"),
          },
        ],
      },
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
