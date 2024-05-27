import * as fileUtil from "./fileUtil";
import init, { get_image_metadata, get_time_offset } from "image-wasm";

export enum ImageStatus {
  PENDING = "pending",
  LOADING = "loading",
  LOADED = "loaded",
  RESIZING = "resizing",
  RESIZED = "resized",
  UPLOADING = "uploading",
  UPLOADED = "uploaded",
  CREATING = "creating",
  CREATED = "created",
}

export type Image = {
  file: File | null;
  status: ImageStatus;
  originalFileName: string;
  data: ArrayBuffer | null;
  size: number;
};

export async function loadImage(image: Image) {
  if (image.file == null) {
    console.log(`File object of ${image.originalFileName} is null`);
    return;
  }
  const data = await fileUtil.loadFile(image.file);
  if (data == null) {
    console.log("Failed to load file");
    return;
  }
  image.data = data;
}

export async function loadFileMetadata(image: Image) {
  if (image.data == null) {
    console.log(`Data of ${image.originalFileName} is null`);
    return;
  }

  try {
    await init();
    const imageMetadata = await get_image_metadata(image.data);
    console.log(imageMetadata);
  } catch (error: any) {
    console.log("Failed to get image metadata");
    console.log(error);
    return;
  }
}
