import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { ImageWithTagsType } from "src/types/custom";

const { imagesApi } = vi.hoisted(() => ({
  imagesApi: { list: vi.fn(), get: vi.fn() },
}));

vi.mock("src/api", () => ({
  api: { images: imagesApi },
}));

// imageQueryLogic touches the user store at import time, so pinia must be live first
setActivePinia(createPinia());
const { images, imageIndex, totalImageCount, nextImage, nextRow } = await import("src/pages/image/imageQueryLogic");

function fakeImages(count: number, offset = 0): ImageWithTagsType[] {
  return Array.from({ length: count }, (_, i) => ({ id: `img-${offset + i}`, tags: [] }) as unknown as ImageWithTagsType);
}

describe("detail/zen navigation prefetch", () => {
  beforeEach(() => {
    imagesApi.list.mockReset();
    imagesApi.list.mockResolvedValue({ items: fakeImages(20, 20), total: 40 });
    images.value = fakeImages(20);
    totalImageCount.value = 40;
  });

  it("prefetches the next page when stepping onto the length-4 threshold", () => {
    imageIndex.value = 15;
    nextImage(); // -> 16 === 20 - 4
    expect(imagesApi.list).toHaveBeenCalled();
  });

  it("prefetches when entering past the threshold (regression: strict equality missed the one-shot moment)", () => {
    imageIndex.value = 18; // entered detail/zen on a late image
    nextImage(); // -> 19, already past length - 4
    expect(imagesApi.list).toHaveBeenCalled();
  });

  it("retries the load when stuck on the last loaded image", () => {
    imageIndex.value = 19;
    nextImage(); // cannot advance, but must still trigger the fetch
    expect(imageIndex.value).toBe(19);
    expect(imagesApi.list).toHaveBeenCalled();
  });

  it("does not fetch once everything is loaded", () => {
    totalImageCount.value = 20;
    imageIndex.value = 19;
    nextImage();
    nextRow();
    expect(imagesApi.list).not.toHaveBeenCalled();
  });
});
