import { test, expect } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { loginAs, collectJsErrors } from "./helpers";

// The browser ingest pipeline — the one path no Go e2e test can reach:
// file -> WASM resize/EXIF -> presigned PUT to S3 -> image record.
//
// This spec exists because smoke.spec.ts visiting /uploads/:id/edit proved only
// that the page paints. The offset mapping it feeds to WASM is a lazy computed
// with no template consumer, so a render-only test can never evaluate it — a
// fractional-timestamp crash there killed every upload while the suite stayed green.
//
// Prerequisites beyond the usual dev stack (see README): the WASM module must be
// built (`./image-wasm/hack/build.sh`) and S3/RustFS must be reachable, since this
// test genuinely resizes and uploads bytes.

// The name matters: the pipeline extracts a 4-digit frame number from it
// (image-wasm/src/filename.rs), so a fixture without one is rejected outright.
const FIXTURE_NAME = "20240817-0B8A0042.jpg";
const FIXTURE = path.join(path.dirname(fileURLToPath(import.meta.url)), "fixtures", FIXTURE_NAME);

// Canon R5 is the seeded camera that owns a *fresh* time offset and belongs to
// projectEditor; without an offset the pipeline correctly refuses to process.
const SEED_CAMERA = "Canon R5";

test.describe("browser upload pipeline", () => {
  // Real WASM resize into 4 dimensions + 5 presigned S3 PUTs per image — well past
  // the suite's 30s default.
  test.describe.configure({ timeout: 150_000 });

  test("processes a real JPEG through WASM to a persisted image", async ({ page }) => {
    const errors = collectJsErrors(page);
    // The pipeline reports its failures through the logger, not exceptions — without
    // this, a broken upload just times out with no reason attached.
    const pipelineLog: string[] = [];
    page.on("console", (m) => pipelineLog.push(`${m.type()}: ${m.text()}`));
    const project = await loginAs(page, "projectEditor");
    expect(project, "projectEditor must see the seed project").not.toBeNull();

    const upload = await page.evaluate(
      async ({ projectId, cameraName }) => {
        const j = async (u: string) => {
          const r = await fetch(u, { credentials: "include" });
          const b = await r.json().catch(() => ({}));
          return b?.items || b?.data || b?.results || (Array.isArray(b) ? b : []);
        };
        const me = await (await fetch("/api/v1/users/me", { credentials: "include" })).json();
        const cameras = await j("/api/v1/cameras?limit=50");
        const camera = cameras.find((c: any) => c.name === cameraName);
        if (!camera) return { error: `seed camera ${cameraName} missing` };
        const r = await fetch("/api/v1/uploads", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ name: "e2e upload pipeline", projectId, cameraId: camera.id, userId: me.id }),
        });
        if (!r.ok) return { error: `create upload: ${r.status} ${await r.text()}` };
        // copyrightTag gates the pipeline (FileProcessor bails without one) — assert
        // the seeded persona actually has it rather than debugging a silent stall.
        return { id: (await r.json()).id, copyrightTag: me.copyrightTag };
      },
      { projectId: project!.id, cameraName: SEED_CAMERA },
    );

    expect(upload.error, "upload fixture setup").toBeUndefined();
    expect(upload.copyrightTag, "seeded user needs a copyrightTag to upload").toBeTruthy();

    // computedFileName is globally unique and derives from the fixture's fixed EXIF
    // timestamp, so it is identical on every run. Without this the second run of the
    // spec fails with a 409 from the create step (the suite normally reseeds first).
    await page.evaluate(
      async ({ projectId, fileName }) => {
        const r = await fetch(`/api/v1/images?projectId=${projectId}&search=${encodeURIComponent(fileName)}&limit=50`, { credentials: "include" });
        const b = await r.json().catch(() => ({}));
        for (const img of b?.items || b?.data || []) {
          if (img.fileName === fileName) await fetch(`/api/v1/images/${img.id}`, { method: "DELETE", credentials: "include" });
        }
      },
      { projectId: project!.id, fileName: FIXTURE_NAME },
    );

    await page.goto(`/uploads/${upload.id}/edit`);
    await expect(page.locator("#dropzoneFile")).toBeAttached();
    await page.setInputFiles("#dropzoneFile", FIXTURE);

    // The tile appears immediately and carries the pipeline state; DONE removes
    // the overlay, so poll the server instead of racing the UI's final frame.
    try {
      await expect
        .poll(
          () =>
            page.evaluate(
              async ({ uploadId, projectId }) => {
                // projectId is mandatory on this endpoint — omitting it 400s, which would
                // read here as "no images yet" and mask a working pipeline.
                const r = await fetch(`/api/v1/images?projectId=${projectId}&uploadId=${uploadId}&limit=10`, { credentials: "include" });
                if (!r.ok) throw new Error(`images list ${r.status}`);
                const b = await r.json().catch(() => ({}));
                return (b?.items || b?.data || []).length;
              },
              { uploadId: upload.id, projectId: project!.id },
            ),
          { message: "image record created by the upload pipeline", timeout: 90_000, intervals: [500, 1000, 2000] },
        )
        .toBe(1);
    } catch (e) {
      // The pipeline swallows failures into the tile's "error" state, so surface the
      // logger output — otherwise this is an unreadable timeout.
      throw new Error(`${(e as Error).message}\n\npipeline log:\n${pipelineLog.filter((l) => /error|ERROR|fail/i.test(l)).join("\n") || "(no errors logged)"}`);
    }

    const image = await page.evaluate(
      async ({ uploadId, projectId }) => {
        const r = await fetch(`/api/v1/images?projectId=${projectId}&uploadId=${uploadId}&limit=10`, { credentials: "include" });
        const b = await r.json().catch(() => ({}));
        return (b?.items || b?.data || [])[0];
      },
      { uploadId: upload.id, projectId: project!.id },
    );

    expect(image.fileName).toBe(FIXTURE_NAME);
    // Proof the WASM EXIF + time-offset path ran: both are derived from the
    // fixture's DateTimeOriginal corrected by the seeded camera offset.
    expect(image.computedFileName, "computedFileName derives from corrected time").toBeTruthy();
    expect(image.capturedAt, "capturedAt comes from EXIF DateTimeOriginal").toBeTruthy();
    expect(image.capturedAtCorrected, "capturedAtCorrected applies the camera time offset").toBeTruthy();
    expect(image.capturedAtCorrected).not.toBe(image.capturedAt);

    expect(errors, "no uncaught JS errors during the upload pipeline").toEqual([]);
  });

  test("refuses to process when the user has no copyright tag", async ({ page }) => {
    const project = await loginAs(page, "projectEditor");
    const result = await page.evaluate(
      async ({ projectId, cameraName }) => {
        const j = async (u: string) => {
          const r = await fetch(u, { credentials: "include" });
          const b = await r.json().catch(() => ({}));
          return b?.items || b?.data || (Array.isArray(b) ? b : []);
        };
        const me = await (await fetch("/api/v1/users/me", { credentials: "include" })).json();
        const camera = (await j("/api/v1/cameras?limit=50")).find((c: any) => c.name === cameraName);
        const r = await fetch("/api/v1/uploads", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ name: "e2e no-copyright", projectId, cameraId: camera.id, userId: me.id }),
        });
        const uploadId = (await r.json()).id;
        // Drop the copyright tag for this one run, then restore it below.
        await fetch(`/api/v1/users/${me.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ copyrightTag: "" }),
        });
        return { uploadId, userId: me.id, restore: me.copyrightTag };
      },
      { projectId: project!.id, cameraName: SEED_CAMERA },
    );

    try {
      await page.goto(`/uploads/${result.uploadId}/edit`);
      await page.setInputFiles("#dropzoneFile", FIXTURE);
      // Scope to this image's own tile and match the status text exactly — a loose
      // page-wide "error" match would pass on any unrelated copy.
      const tile = page.locator("figure").filter({ hasText: FIXTURE_NAME });
      await expect(tile.getByText("error", { exact: true })).toBeVisible({ timeout: 30_000 });
    } finally {
      await page.evaluate(
        async ({ userId, restore }) => {
          await fetch(`/api/v1/users/${userId}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({ copyrightTag: restore }),
          });
        },
        { userId: result.userId, restore: result.restore },
      );
    }
  });
});
