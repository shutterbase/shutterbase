export const projectRoutes = [
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
];
