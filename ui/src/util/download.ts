import { EXIF_WORKER_URL } from "src/boot/pocketbase";
import { ImageWithTagsType, DownloadUrls } from "src/types/custom";
import pb from "src/boot/pocketbase";
import { emitter } from "src/boot/mitt";

async function fetchImage(image: ImageWithTagsType, resolution: keyof DownloadUrls): Promise<ArrayBuffer> {
  const id = image.id;
  const url = `${EXIF_WORKER_URL}/api/download/${id}/${resolution}`;
  const token = pb.authStore.token;

  emitter.emit("notification", { headline: "Downloading image...", type: "info" });
  try {
    const response = await fetch(url, {
      headers: {
        Authorization: `${token}`,
      },
    });

    if (!response.ok) {
      throw new Error("Failed to download image");
    }
    emitter.emit("notification", { headline: "Download ready", type: "success" });
    return response.arrayBuffer();
  } catch (error) {
    emitter.emit("notification", { headline: "Failed to download image", type: "error" });
    throw error;
  }
}

export async function downloadImage(image: ImageWithTagsType, resolution: keyof DownloadUrls): Promise<void> {
  const buffer = await fetchImage(image, resolution);
  const blob = new Blob([buffer], { type: "image/jpeg" });
  const url = URL.createObjectURL(blob);

  const a = document.createElement("a");
  a.href = url;
  a.download = getDownloadFileName(image, resolution);
  a.click();
}

function getDownloadFileName(image: ImageWithTagsType, resolution: keyof DownloadUrls): string {
  let fileName = image.computedFileName;
  if (resolution !== "original") {
    fileName += `_${resolution}`;
  }

  return `${fileName}.jpg`;
}
