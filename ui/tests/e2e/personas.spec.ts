import { test, expect, Page } from "@playwright/test";
import { loginAs, collectJsErrors, Persona } from "./helpers";

// Capability matrix grounded in the UI's role gating (see api authorization +
// component v-ifs). Each flag maps to an observable affordance:
//   imagesNav   — header shows Images/Uploads (requires an active project)
//   seesProject — the seed project is listed on /projects
//   membersTab  — project sub-nav shows "Members"   (isProjectAdminOrHigher)
//   dangerTab   — project sub-nav shows "Danger Zone" (isAdmin only)
//   tagsAdd     — "Add Project Tag" button           (isProjectAdminOrHigher)
//   generalEdit — DetailEditGroup "Edit" buttons      (isProjectAdminOrHigher)
//   uploadAdd   — "Add Upload" button                 (isProjectEditorOrHigher)
type Caps = {
  imagesNav: boolean;
  seesProject: boolean;
  membersTab: boolean;
  dangerTab: boolean;
  tagsAdd: boolean;
  generalEdit: boolean;
  uploadAdd: boolean;
};

const MATRIX: Record<Persona, Caps> = {
  admin: { imagesNav: true, seesProject: true, membersTab: true, dangerTab: true, tagsAdd: true, generalEdit: true, uploadAdd: true },
  user: { imagesNav: false, seesProject: false, membersTab: false, dangerTab: false, tagsAdd: false, generalEdit: false, uploadAdd: false },
  projectAdmin: { imagesNav: true, seesProject: true, membersTab: true, dangerTab: false, tagsAdd: true, generalEdit: true, uploadAdd: true },
  projectEditor: { imagesNav: true, seesProject: true, membersTab: false, dangerTab: false, tagsAdd: false, generalEdit: false, uploadAdd: true },
  projectViewer: { imagesNav: true, seesProject: true, membersTab: false, dangerTab: false, tagsAdd: false, generalEdit: false, uploadAdd: false },
};

const visibleOrAbsent = async (locator: ReturnType<Page["getByText"]>, expected: boolean, label: string) => {
  if (expected) await expect(locator.first(), `${label} should be visible`).toBeVisible();
  else await expect(locator, `${label} should be absent`).toHaveCount(0);
};

for (const role of Object.keys(MATRIX) as Persona[]) {
  test(`persona ${role}: nav + action gating matches role`, async ({ page }) => {
    const caps = MATRIX[role];
    const errors = collectJsErrors(page);
    const project = await loginAs(page, role);

    // --- top nav ---
    await page.goto("/");
    const topnav = page.locator("header nav");
    await expect(topnav.getByText("Projects", { exact: true })).toBeVisible();
    await visibleOrAbsent(topnav.getByText("Images", { exact: true }), caps.imagesNav, `${role} Images nav`);
    await visibleOrAbsent(topnav.getByText("Uploads", { exact: true }), caps.imagesNav, `${role} Uploads nav`);

    // --- projects list scoping ---
    await page.goto("/projects");
    if (caps.seesProject) {
      await expect(page.getByText("Formula Student Test").first()).toBeVisible();
    } else {
      await expect(page.getByText(/No projects found/i)).toBeVisible();
    }

    // --- project-scoped gating (skip personas with no project) ---
    if (project) {
      const pid = project.id;

      await page.goto(`/projects/${pid}/general`);
      const subnav = page.locator("main nav").filter({ hasText: "General" });
      await expect(subnav.getByText("General", { exact: true })).toBeVisible();
      await visibleOrAbsent(subnav.getByText("Members", { exact: true }), caps.membersTab, `${role} Members tab`);
      await visibleOrAbsent(subnav.getByText("Danger Zone", { exact: true }), caps.dangerTab, `${role} Danger Zone tab`);

      const editButtons = page.getByRole("button", { name: /^Edit$/ });
      if (caps.generalEdit) await expect(editButtons.first(), `${role} general Edit`).toBeVisible();
      else await expect(editButtons, `${role} general Edit absent`).toHaveCount(0);

      await page.goto(`/projects/${pid}/tags`);
      await visibleOrAbsent(page.getByRole("button", { name: /Add Project Tag/i }), caps.tagsAdd, `${role} Add Project Tag`);

      await page.goto("/uploads");
      await visibleOrAbsent(page.getByRole("button", { name: /Add Upload/i }), caps.uploadAdd, `${role} Add Upload`);
    }

    expect(errors, `JS errors for ${role}:\n${errors.join("\n")}`).toHaveLength(0);
  });
}
