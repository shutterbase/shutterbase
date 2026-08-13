import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ref } from "vue";

vi.mock("image-wasm", () => ({
  default: vi.fn(async () => {}),
  set_log_level: vi.fn(),
  process_file: vi.fn(),
}));
vi.mock("src/boot/axios", () => ({ API_BASE: "" }));
vi.mock("src/boot/mitt", () => ({ showNotificationToast: vi.fn() }));
vi.mock("src/api", () => ({ api: { images: { create: vi.fn() } } }));
vi.mock("src/stores/user-store", () => ({ useUserStore: () => ({ user: { copyrightTag: "max" } }) }));
vi.mock("src/util/fileUtil", () => ({ loadFile: vi.fn(async () => new ArrayBuffer(4)) }));

import { process_file } from "image-wasm";
import { api } from "src/api";
import { FileProcessor, Image, ImageStatus, newImage } from "src/util/fileProcessor";

function deferred<T>() {
  let resolve!: (v: T) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

const wasmResult = {
  storage_id: "storage-1",
  camera_time_unix_seconds: 1_700_000_000,
  corrected_camera_time_unix_seconds: 1_700_000_010,
  thumbnail: "thumb",
  metadata: new Map<string, string>(),
  original_width: 100,
  original_height: 50,
};

describe("FileProcessor pause/resume", () => {
  const upload = ref({ id: "upl1", project: { id: "prj1" }, camera: { id: "cam1" } } as any);
  const timeOffsets = ref([{}] as any);
  let images: ReturnType<typeof ref<Image[]>>;
  let processor: FileProcessor;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(process_file).mockReset();
    vi.mocked(api.images.create).mockReset();
    images = ref<Image[]>([]);
  });

  afterEach(() => {
    processor?.stop();
    vi.useRealTimers();
  });

  function addImage(name = "a.jpg"): Image {
    const image = newImage({ file: new File(["x"], name) });
    images.value!.push(image);
    return image;
  }

  async function tick(ms = 300) {
    await vi.advanceTimersByTimeAsync(ms);
  }

  it("pause during processing resets the image to PENDING and drops the stale completion", async () => {
    const wasm = deferred<typeof wasmResult>();
    vi.mocked(process_file).mockReturnValue(wasm.promise as any);
    const image = addImage();
    processor = new FileProcessor(upload, images as any, timeOffsets);
    await tick();
    expect(image.status).toBe(ImageStatus.RESIZING);

    processor.pause();
    expect(processor.paused.value).toBe(true);
    expect(image.status).toBe(ImageStatus.PENDING);
    expect(image.data).toBeNull();

    // the unabortable WASM call finishes after the pause — it must not advance the image
    wasm.resolve(wasmResult);
    await tick();
    expect(image.status).toBe(ImageStatus.PENDING);
    expect(image.storageId).toBeUndefined();
    expect(api.images.create).not.toHaveBeenCalled();
  });

  it("pause during CREATING lets the backend create finish (no duplicate row on resume)", async () => {
    vi.mocked(process_file).mockResolvedValue(wasmResult as any);
    const create = deferred<{ id: string; computedFileName: string }>();
    vi.mocked(api.images.create).mockReturnValue(create.promise as any);
    const image = addImage();
    processor = new FileProcessor(upload, images as any, timeOffsets);
    await tick();
    expect(image.status).toBe(ImageStatus.CREATING);

    processor.pause();
    // NOT reset: the DB row may already exist — resetting would re-create it on
    // resume and collide on the unique computedFileName
    expect(image.status).toBe(ImageStatus.CREATING);

    create.resolve({ id: "img1", computedFileName: "20260813_10-00-00_x_max" });
    await tick();
    expect(image.status).toBe(ImageStatus.DONE);
    expect(image.id).toBe("img1");
    expect(api.images.create).toHaveBeenCalledTimes(1);
  });

  it("resume re-runs a reset image through the full pipeline exactly once", async () => {
    const firstRun = deferred<typeof wasmResult>();
    vi.mocked(process_file).mockReturnValueOnce(firstRun.promise as any).mockResolvedValue({ ...wasmResult, storage_id: "storage-2" } as any);
    vi.mocked(api.images.create).mockResolvedValue({ id: "img1", computedFileName: "x" } as any);
    const image = addImage();
    processor = new FileProcessor(upload, images as any, timeOffsets);
    await tick();

    processor.pause();
    firstRun.resolve(wasmResult); // stale
    await tick();
    expect(image.status).toBe(ImageStatus.PENDING);

    processor.resume();
    await tick();
    expect(image.status).toBe(ImageStatus.DONE);
    expect(image.storageId).toBe("storage-2");
    expect(api.images.create).toHaveBeenCalledTimes(1);
    expect(vi.mocked(api.images.create).mock.calls[0][0]).toMatchObject({ storageId: "storage-2", uploadId: "upl1" });
  });

  it("while paused, new PENDING images are not picked up", async () => {
    vi.mocked(process_file).mockResolvedValue(wasmResult as any);
    processor = new FileProcessor(upload, images as any, timeOffsets);
    await tick();
    processor.pause();
    const image = addImage();
    await tick();
    expect(image.status).toBe(ImageStatus.PENDING);
    expect(process_file).not.toHaveBeenCalled();
  });
});
