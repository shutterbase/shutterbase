import { DateTime } from "luxon";
import * as fileUtil from "./fileUtil";
import init, { set_log_level, process_file, TimeOffsetResult as WasmTimeOffsetResult, FileProcessorOptions, FileProcessorResult } from "image-wasm";
import { Ref, ref } from "vue";
import pb, { URL as BACKEND_BASE_URL } from "src/boot/pocketbase";
import { getLogLevelString, debug, info, error } from "./logger";
import { time } from "console";
import { UploadsResponse, ImagesRecord, ImagesResponse } from "src/types/pocketbase";
import { dateTimeFromBackend, parseBackendTime } from "src/util/dateTimeUtil";

export type TimeOffsetResult = WasmTimeOffsetResult;

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
  storageId?: string;
  file: File | null;
  status: ImageStatus;
  progress: number;
  originalFileName: string;
  computedFileName?: string;
  cameraTime?: DateTime;
  correctedTime?: DateTime;
  data: ArrayBuffer | null;
  thumbnail?: string;
  size: number;
};

export class FileProcessor {
  private upload: Ref<UploadsResponse> = ref({} as UploadsResponse);
  private images: Ref<Image[]> = ref([]);
  private timeOffsets: Ref<TimeOffsetResult[]> = ref([]);
  private interval: NodeJS.Timeout | null = null;

  constructor(upload: Ref<UploadsResponse>, images: Ref<Image[]>, timeOffsets: Ref<TimeOffsetResult[]>) {
    this.upload = upload;
    this.images = images;
    this.timeOffsets = timeOffsets;
    this.start();
  }

  public stop(): void {
    if (this.interval != null) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }

  public start = async (): Promise<void> => {
    if (this.interval == null) {
      await init();
      set_log_level(getLogLevelString());
      this.interval = setInterval(this.processImages, 100);
    }
  };

  public isRunning(): boolean {
    return this.interval != null;
  }

  private processImages = async () => {
    const pendingImagesResult = this.processPendingImages();
    const loadedImagesResult = this.processLoadedImages();
    const createdImagesResult = this.processUploadedImages();

    // if (!pendingImagesResult && !loadedImagesResult && !createdImagesResult) {
    //   if (this.interval != null) {
    //     clearInterval(this.interval);
    //     this.interval = null;
    //   }
    // }
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
          // this.processImages();
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
          // this.processImages();
        })
        .catch(() => {
          this.setState(image, ImageStatus.ERROR);
        });
    }
    return true;
  };

  private processUploadedImages = (): boolean => {
    while (this.getStateCount(ImageStatus.CREATING) < poolSizeLimits.creating) {
      const image = this.getNextImage(ImageStatus.UPLOADED);
      if (image == null) {
        return false;
      }

      this.setState(image, ImageStatus.CREATING);
      this.createBackendImage(image)
        .then(() => {
          this.setState(image, ImageStatus.DONE);
          // this.processImages();
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
    debug(`Image ${image.originalFileName} - ${oldStatus} => ${status}`);
  };

  private loadImage = (image: Image): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      if (image.file == null) {
        error(`File object of ${image.originalFileName} is null`);
        reject();
        return;
      }
      const data = await fileUtil.loadFile(image.file);
      if (data == null) {
        error("Failed to load file");
        reject();
      }
      image.data = data;
      resolve();
    });
  };

  private processImage = (image: Image): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      if (image.data == null) {
        error(`Data of ${image.originalFileName} is null`);
        reject();
        return;
      }

      if (this.timeOffsets.value.length == 0) {
        error("No time offsets available");
        reject();
        return;
      }

      const copyrightTag = pb.authStore.model?.copyrightTag;
      if (copyrightTag == null || copyrightTag == "") {
        error("No copyright tag available");
        reject();
        return;
      }

      const authToken = pb.authStore.token;
      if (authToken == null || authToken == "") {
        error("No auth token available");
        reject();
        return;
      }

      const options: FileProcessorOptions = {
        file_name: image.originalFileName,
        time_offsets: this.timeOffsets.value,
        copyright_tag: copyrightTag,
        dimensions: FILE_DIMENSIONS,
        thumbnail_size: 256,
        auth_token: authToken,
        api_url: `${BACKEND_BASE_URL}/api`,
      };

      try {
        const processingResult: FileProcessorResult = await process_file(image.data, options, (status: ImageStatus, progress: number) => {
          image.status = status;
          image.progress = progress;
        });
        debug(processingResult);
        image.storageId = processingResult.storage_id;
        image.cameraTime = DateTime.fromSeconds(processingResult.camera_time_unix_seconds);
        image.correctedTime = DateTime.fromSeconds(processingResult.corrected_camera_time_unix_seconds);
        image.computedFileName = processingResult.computed_file_name;
        image.thumbnail = processingResult.thumbnail;
        resolve();
      } catch (err: any) {
        error(`Failed to process image: ${err}`);
        reject();
        return;
      }
    });
  };

  private createBackendImage = (image: Image): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      pb.collection<ImagesRecord>("images")
        .create({
          storageId: image.storageId,
          fileName: image.originalFileName,
          computedFileName: image.computedFileName,
          size: image.size,
          capturedAt: image.cameraTime?.toISO(),
          capturedAtCorrected: image.correctedTime?.toISO(),
          user: pb.authStore.model?.id,
          upload: this.upload.value.id,
          project: this.upload.value.project,
          camera: this.upload.value.camera,
        })
        .then((response) => {
          resolve();
        })
        .catch((err) => {
          error(`Failed to create image in backend: ${err}`);
          reject();
        });
    });
  };
}

export function newImage(options: { file: File }): Image {
  return {
    storageId: undefined,
    status: ImageStatus.PENDING,
    progress: 0,
    file: options.file,
    size: options.file.size,
    originalFileName: options.file.name,
    computedFileName: undefined,
    cameraTime: undefined,
    correctedTime: undefined,
    data: null,
    thumbnail: undefined,
  };
}

export function newImageFromBackendImage(backendImage: ImagesResponse): Image {
  return {
    storageId: backendImage.storageId,
    status: ImageStatus.DONE,
    progress: 100,
    file: null,
    size: backendImage.size,
    originalFileName: backendImage.fileName,
    computedFileName: backendImage.computedFileName,
    cameraTime: DateTime.fromJSDate(parseBackendTime(backendImage.capturedAt)),
    correctedTime: DateTime.fromJSDate(parseBackendTime(backendImage.capturedAtCorrected)),
    data: null,
    thumbnail: undefined,
  };
}
