import { useLoginStore } from "src/stores/login-store";
import { useUserStore } from "src/stores/user-store";
import { RouteRecordRaw } from "vue-router";

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    component: () => import("layouts/MainLayout.vue"),
    children: [
      { path: "", component: () => import("pages/IndexPage.vue") },
      { path: "/register", component: () => import("pages/Register.vue") },
      {
        path: "/request-password-reset",
        component: () => import("pages/RequestPasswordReset.vue"),
      },
      {
        path: "/password-reset",
        component: () => import("pages/PasswordReset.vue"),
      },
      { path: "/login", component: () => import("pages/Login.vue") },
      { path: "/logout", component: () => import("pages/Logout.vue") },
      { path: "/check-email", component: () => import("pages/CheckEmail.vue") },
      {
        path: "/confirm-email/",
        component: () => import("pages/ConfirmEmail.vue"),
      },
      { path: "/help/", component: () => import("pages/chore/Help.vue") },
      { path: "/privacy/", component: () => import("pages/chore/Privacy.vue") },
      { path: "/terms/", component: () => import("pages/chore/Terms.vue") },
      { path: "/legal/", component: () => import("pages/chore/Legal.vue") },
      {
        path: "/codebooks/:id",
        component: () => import("pages/CodebookDetail.vue"),
      },
    ],
  },
  {
    path: "/dashboard",
    beforeEnter: (to, from) => {
      if (!useLoginStore().isLoggedIn) {
        return "/login";
      }
      return true;
    },
    component: () => import("layouts/MainLayout.vue"),
    children: [
      { path: "", component: () => import("pages/Dashboard.vue") },
      {
        path: "users",
        component: () => import("pages/dashboard/Users.vue"),
        beforeEnter: (to, from) => {
          if (!useUserStore().isAdmin) {
            return "/dashboard";
          }
          return true;
        },
      },
      {
        path: "users/:id",
        component: () => import("pages/dashboard/UserDetail.vue"),
        beforeEnter: (to, from) => {
          if (!useUserStore().isAdmin && to.params.id != useUserStore().ownUser()?.id) {
            return "/dashboard";
          }
          return true;
        },
      },
    ],
  },
  // Always leave this as last one,
  // but you can also remove it
  {
    path: "/:catchAll(.*)*",
    component: () => import("pages/ErrorNotFound.vue"),
  },
];

export default routes;
