export const imageRoutes = [
  {
    name: "images",
    path: "/images",
    component: () => import("pages/image/Images.vue"),
  },
  {
    name: "image",
    path: "/images/:id",
    component: () => import("pages/image/Image.vue"),
  },
];
