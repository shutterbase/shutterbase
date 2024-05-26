export const userRoutes = [
  {
    name: "users",
    path: "/users",
    component: () => import("pages/user/Users.vue"),
  },
  {
    name: "camera-create",
    path: "/users/:userid/cameras/create",
    component: () => import("pages/user/CameraCreate.vue"),
  },
  {
    name: "camera-time-offset",
    path: "/cameras/:cameraid/time-offset",
    component: () => import("pages/user/TimeOffsetCreate.vue"),
  },
  {
    name: "user",
    path: "/users/:userid",
    component: () => import("pages/user/User.vue"),
    children: [
      {
        name: "user-general",
        path: "general",
        component: () => import("pages/user/UserGeneral.vue"),
      },
      {
        name: "cameras",
        path: "cameras",
        component: () => import("pages/user/Cameras.vue"),
      },
      // {
      //   name: "camera",
      //   path: "cameras/:cameraid",
      //   component: () => import("pages/user/Camera.vue"),
      // },
    ],
  },
];
