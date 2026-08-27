export const galleryRoutes = [
  {
    name: "galleries",
    path: "/galleries",
    component: () => import("pages/gallery/Galleries.vue"),
  },
  {
    name: "gallery-create",
    path: "/galleries/create",
    component: () => import("pages/gallery/GalleryCreate.vue"),
  },
  {
    name: "gallery-edit",
    path: "/galleries/:id",
    component: () => import("pages/gallery/GalleryEdit.vue"),
  },
];
