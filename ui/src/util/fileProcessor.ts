import { DateTime } from "luxon";
import * as fileUtil from "./fileUtil";
import init, { get_image_metadata, process_file } from "image-wasm";
import { Ref, ref } from "vue";
import pb, { URL as BACKEND_BASE_URL } from "src/boot/pocketbase";

export enum ImageStatus {
  PENDING = "pending", // initial state after file selection
  LOADING = "loading", // loading file data into memory
  LOADED = "loaded", // file data is loaded into memory
  RESIZING = "resizing", // file is being resized by WASM
  UPLOADING = "uploading", // files are being uploaded by WASM to S3
  UPLOADED = "uploaded", // files are uploaded to S3
  CREATING = "creating", // image is being created in the database
  DONE = "done",
  ERROR = "error", // an error occurred somewhere along the way
}

const FILE_DIMENSIONS = [256, 512, 1024, 2048];

// TODO: Also add a limit for the total number of images currently being processed
// FIXME: Remove ArrayBuffers once processing is done to free up memory
const poolSizeLimits = {
  loading: 4,
  processing: 2, // resizing and uploading
  creating: 4,
};

export type Image = {
  file: File | null;
  status: ImageStatus;
  originalFileName: string;
  computedFileName?: string;
  originalTime?: DateTime;
  correctedTime?: DateTime;
  data: ArrayBuffer | null;
  thumbnail?: string;
  size: number;
};

export class FileProcessor {
  private images: Ref<Image[]> = ref([]);
  private interval: NodeJS.Timeout | null = null;

  constructor(images: Ref<Image[]>) {
    this.start();
    this.images = images;
  }

  public stop(): void {
    if (this.interval != null) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }

  public start = async (): Promise<void> => {
    await init();
    if (this.interval == null) {
      this.interval = setInterval(this.processImages, 500);
    }
  };

  public isRunning(): boolean {
    return this.interval != null;
  }

  private processImages = async () => {
    const pendingImagesResult = this.processPendingImages();
    const loadedImagesResult = this.processLoadedImages();

    if (!pendingImagesResult && !loadedImagesResult) {
      if (this.interval != null) {
        clearInterval(this.interval);
        this.interval = null;
      }
    }
  };

  private processPendingImages = (): boolean => {
    while (this.getStateCount(ImageStatus.LOADING) < poolSizeLimits.loading) {
      const image = this.getNextImage(ImageStatus.PENDING);
      if (image == null) {
        return false;
      }

      this.setState(image, ImageStatus.LOADING);
      this.loadImage(image)
        .then(() => {
          this.setState(image, ImageStatus.LOADED);
          this.processImages();
        })
        .catch(() => {
          this.setState(image, ImageStatus.ERROR);
        });
    }
    return true;
  };

  private processLoadedImages = (): boolean => {
    while (this.getStateCount([ImageStatus.RESIZING, ImageStatus.UPLOADING]) < poolSizeLimits.processing) {
      const image = this.getNextImage(ImageStatus.LOADED);
      if (image == null) {
        return false;
      }

      this.setState(image, ImageStatus.RESIZING);
      this.processImage(image)
        .then(() => {
          this.setState(image, ImageStatus.UPLOADED);
          this.processImages();
        })
        .catch(() => {
          this.setState(image, ImageStatus.ERROR);
        });
    }
    return true;
  };

  private getNextImage = (status: ImageStatus): Image | null => {
    for (const image of this.images.value) {
      if (image.status === status) {
        return image;
      }
    }
    return null;
  };

  private getStateCount = (status: ImageStatus | ImageStatus[]): number => {
    if (Array.isArray(status)) {
      return this.images.value.filter((image) => status.includes(image.status)).length;
    } else {
      return this.images.value.filter((image) => image.status === status).length;
    }
  };

  private setState = (image: Image, status: ImageStatus) => {
    const oldStatus = image.status;
    image.status = status;
    console.log(`Image ${image.originalFileName} - ${oldStatus} => ${status}`);
  };

  private loadImage = (image: Image): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      if (image.file == null) {
        console.log(`File object of ${image.originalFileName} is null`);
        reject();
        return;
      }
      const data = await fileUtil.loadFile(image.file);
      if (data == null) {
        console.log("Failed to load file");
        reject();
      }
      image.data = data;
      resolve();
    });
  };

  private processImage = (image: Image): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      if (image.data == null) {
        console.log(`Data of ${image.originalFileName} is null`);
        reject();
        return;
      }

      type FileProcessorOptions = {
        file_name: string;
        dimensions: number[];
        thumbnail_size: number;
        auth_token: string;
        api_url: string;
      };

      const options: FileProcessorOptions = {
        file_name: image.originalFileName,
        dimensions: FILE_DIMENSIONS,
        thumbnail_size: 256,
        auth_token: pb.authStore.token,
        api_url: `${BACKEND_BASE_URL}/api`,
      };

      try {
        const processingResult = await process_file(image.data, options);
        console.log(processingResult);
        image.originalTime = DateTime.fromSeconds(processingResult.original_time);
        image.thumbnail = processingResult.thumbnail;
        resolve();
      } catch (error: any) {
        console.log("Failed to process image");
        console.log(error);
        reject();
        return;
      }
    });
  };
}

export function newImage(options: { file: File }): Image {
  return {
    status: ImageStatus.PENDING,
    file: options.file,
    size: options.file.size,
    originalFileName: options.file.name,
    computedFileName: undefined,
    originalTime: undefined,
    correctedTime: undefined,
    data: null,
    thumbnail: undefined,
  };
}
