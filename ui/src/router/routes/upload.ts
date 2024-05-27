export const uploadRoutes = [
  {
    name: "uploads",
    path: "/uploads",
    component: () => import("pages/upload/Uploads.vue"),
  },
  {
    name: "upload-create",
    path: "/uploads/create",
    component: () => import("pages/upload/UploadCreate.vue"),
  },
  {
    name: "upload",
    path: "/uploads/:id",
    children: [
      {
        name: "upload-edit",
        path: "edit",
        component: () => import("pages/upload/UploadEdit.vue"),
      },
    ],
  },
];
