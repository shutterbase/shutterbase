import { DateTime } from "luxon";
import * as fileUtil from "./fileUtil";
import init, { set_log_level, process_file, TimeOffsetResult as WasmTimeOffsetResult, FileProcessorOptions, FileProcessorResult } from "image-wasm";
import { Ref, ref } from "vue";
import { API_BASE } from "src/boot/axios";
import { api } from "src/api";
import { useUserStore } from "src/stores/user-store";
import { getLogLevelString, debug, info, error } from "./logger";
import { errorHeadline } from "./errorDisplay";
import { showNotificationToast } from "src/boot/mitt";
import { Upload, Image as BackendImage } from "src/types/api";
import { parseBackendTime } from "src/util/dateTimeUtil";

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

const PROCESSING_STATES = [ImageStatus.LOADING, ImageStatus.LOADED, ImageStatus.RESIZING, ImageStatus.UPLOADING, ImageStatus.UPLOADED, ImageStatus.CREATING];

// States pause() may reset to PENDING. CREATING is deliberately absent: once the
// backend create is in flight the DB row may already exist — resetting and
// re-running would collide on the unique computedFileName/storageId. A paused
// CREATING image finishes to DONE instead.
const RESETTABLE_STATES = [ImageStatus.LOADING, ImageStatus.LOADED, ImageStatus.RESIZING, ImageStatus.UPLOADING, ImageStatus.UPLOADED];
const FILE_DIMENSIONS = [256, 512, 1024, 2048];
const PARALLEL_PROCESSING = 2;

// TODO: Also add a limit for the total number of images currently being processed
// FIXME: Remove ArrayBuffers once processing is done to free up memory
const poolSizeLimits = {
  loading: 4,
  processing: 2, // resizing and uploading
  creating: 4,
};

// Mirrors the server's computedFileName rule (image_service.go `last4`): a file
// name needs four consecutive digits — the camera frame number — or the API
// rejects the create with 400.
export function hasFrameNumber(fileName: string): boolean {
  return /\d{4}/.test(fileName);
}

export type Image = {
  id?: string;
  storageId?: string;
  file: File | null;
  status: ImageStatus;
  progress: number;
  originalFileName: string;
  computedFileName?: string;
  cameraTime?: DateTime;
  correctedTime?: DateTime;
  data: ArrayBuffer | null;
  errorMessage?: string; // human-readable reason when status === ERROR
  thumbnail?: string;
  downloadUrls?: { [key: string]: string };
  size: number;
  exifData?: any;
  width?: number;
  height?: number;
};

export class FileProcessor {
  private upload: Ref<Upload> = ref({} as Upload);
  private images: Ref<Image[]> = ref([]);
  private timeOffsets: Ref<TimeOffsetResult[]> = ref([]);
  private interval: NodeJS.Timeout | null = null;

  public readonly paused = ref(false);
  // pause() bumps the epoch; every in-flight stage completion carries the epoch
  // it started under and is dropped when it no longer matches — the WASM call
  // itself cannot be aborted, so cancellation happens at the stage boundaries.
  private epoch = 0;

  constructor(upload: Ref<Upload>, images: Ref<Image[]>, timeOffsets: Ref<TimeOffsetResult[]>) {
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

  public pause = (): void => {
    this.paused.value = true;
    this.epoch++;
    for (const image of this.images.value) {
      if (RESETTABLE_STATES.includes(image.status)) {
        this.resetImage(image);
      }
    }
  };

  public resume = (): void => {
    this.paused.value = false;
  };

  // Back to "not uploaded": keeps the file handle (needed to re-enter the
  // pipeline) and the thumbnail (cosmetic), drops everything derived from
  // processing so the resumed run starts clean with a fresh storageId.
  private resetImage = (image: Image): void => {
    this.setState(image, ImageStatus.PENDING);
    image.progress = 0;
    image.data = null;
    image.storageId = undefined;
    image.cameraTime = undefined;
    image.correctedTime = undefined;
    image.exifData = undefined;
    image.width = undefined;
    image.height = undefined;
  };

  private processImages = async () => {
    this.processPendingImages();
  };

  private processPendingImages = (): void => {
    if (this.paused.value) {
      return;
    }
    // if (this.getStateCount(ImageStatus.LOADING) != 0) {
    if (this.getStateCount(PROCESSING_STATES) >= PARALLEL_PROCESSING) {
      return;
    }

    const image = this.getNextImage(ImageStatus.PENDING);
    if (image == null) {
      return;
    }

    // Mirrors the server rule (image_service.go last4): the canonical name keeps
    // the camera frame number, so the API rejects a name without one — fail here
    // before wasting a resize + S3 upload on a doomed file.
    if (!hasFrameNumber(image.originalFileName)) {
      this.fail(image, new Error("file name must contain four consecutive digits (camera frame number)"));
      return;
    }

    const epoch = this.epoch;
    this.setState(image, ImageStatus.LOADING);
    this.loadImage(image)
      .then(() => {
        if (epoch !== this.epoch) return;
        this.setState(image, ImageStatus.LOADED);
        this.processLoadedImage(image, epoch);
      })
      .catch((err) => {
        if (epoch === this.epoch) this.fail(image, err);
      });
  };

  private processLoadedImage = (image: Image, epoch: number): void => {
    this.setState(image, ImageStatus.RESIZING);
    this.processImage(image, epoch)
      .then(() => {
        if (epoch !== this.epoch) return;
        this.setState(image, ImageStatus.UPLOADED);
        this.processUploadedImage(image);
      })
      .catch((err) => {
        if (epoch === this.epoch) this.fail(image, err);
      });
  };

  private processUploadedImage = (image: Image): void => {
    this.setState(image, ImageStatus.CREATING);
    this.createBackendImage(image)
      .then(() => {
        this.setState(image, ImageStatus.DONE);
      })
      .catch((err) => {
        this.fail(image, err);
      });
  };

  // Every failure funnels through here: the tile shows the reason, the log
  // keeps it, and the first image failing for a given reason raises a toast —
  // batch-wide problems (no copyright tag, no time offset) toast once, not
  // once per file.
  private toastedReasons = new Set<string>();

  private fail = (image: Image, err: any): void => {
    const reason = String(err?.message ?? err ?? "unknown error").replace(/^Error:\s*/, "");
    image.errorMessage = reason;
    error(`${image.originalFileName}: ${reason}`);
    if (!this.toastedReasons.has(reason)) {
      this.toastedReasons.add(reason);
      showNotificationToast({ headline: `${image.originalFileName}: ${reason}`, type: "error" });
    }
    this.setState(image, ImageStatus.ERROR);
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
    info(`Image ${image.originalFileName} - ${oldStatus} => ${status}`);
  };

  // Plain async (no Promise executor): a rejection from loadFile must reach the
  // caller's .catch — inside an async executor it would strand the tile in LOADING.
  private loadImage = async (image: Image): Promise<void> => {
    if (image.file == null) {
      throw new Error("file handle is gone — remove the tile and add the file again");
    }
    image.data = await fileUtil.loadFile(image.file).catch(() => {
      throw new Error("could not read the file — it may have moved or be unreadable");
    });
  };

  private processImage = (image: Image, epoch: number): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      if (image.data == null) {
        reject(new Error("file data is missing — remove the tile and add the file again"));
        return;
      }

      if (this.timeOffsets.value.length == 0) {
        reject(new Error("this camera has no time offset yet — photograph the time-sync QR code first"));
        return;
      }

      // The copyright tag is no longer needed for processing (the server names
      // the file), but a photographer without one would get an unnamed image —
      // still worth refusing here rather than after the S3 upload.
      const copyrightTag = useUserStore().user?.copyrightTag;
      if (copyrightTag == null || copyrightTag == "") {
        reject(new Error("your profile has no copyright tag — add one in your profile settings, then retry the upload"));
        return;
      }

      const options: FileProcessorOptions = {
        time_offsets: this.timeOffsets.value,
        dimensions: FILE_DIMENSIONS,
        thumbnail_size: 256,
        // cookie-session: WASM uploads use credentials:include, no bearer token.
        api_url: API_BASE,
        // binds the presign request to this upload (server checks CanModifyUpload).
        upload_id: this.upload.value.id,
      };

      try {
        const processingResult: FileProcessorResult = await process_file(image.data, options, (status: ImageStatus, progress: number) => {
          // a paused image was reset to PENDING — the unabortable WASM call
          // must not flip it back to a progress state
          if (epoch !== this.epoch) return;
          image.status = status;
          image.progress = progress;
        });
        debug(processingResult);
        if (epoch !== this.epoch) {
          resolve();
          return;
        }
        image.storageId = processingResult.storage_id;
        image.cameraTime = DateTime.fromSeconds(processingResult.camera_time_unix_seconds);
        image.correctedTime = DateTime.fromSeconds(processingResult.corrected_camera_time_unix_seconds);
        image.thumbnail = processingResult.thumbnail;
        image.exifData = Object.fromEntries(processingResult.metadata);
        image.width = processingResult.original_width;
        image.height = processingResult.original_height;

        resolve();
      } catch (err: any) {
        // WASM errors name their failure mode (e.g. the hard timestamp rule:
        // "image has no EXIF capture time") — fail() shows them verbatim.
        reject(err);
        return;
      }
    });
  };

  private createBackendImage = (image: Image): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      api.images
        .create({
          storageId: image.storageId!,
          fileName: image.originalFileName,
          size: image.size,
          width: image.width,
          height: image.height,
          capturedAt: image.cameraTime?.toISO() ?? undefined,
          uploadId: this.upload.value.id,
          projectId: this.upload.value.project.id,
          cameraId: this.upload.value.camera.id,
          exifData: image.exifData,
        })
        .then((response) => {
          image.id = response.id;
          // The server owns the canonical name — it renders the timestamp in the
          // event zone (TIMEZONE), which a browser cannot know. Taking it from
          // the response keeps the tile identical to the persisted name instead
          // of computing a second, browser-zone version of the same rule.
          image.computedFileName = response.computedFileName;
          resolve();
        })
        .catch((err) => {
          // API failures (duplicate image, upload no longer accepts images, …)
          // surface with the server's message, not a bare error tile.
          reject(new Error(errorHeadline(err)));
        });
    });
  };
}

export function newImage(options: { file: File }): Image {
  return {
    id: undefined,
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
    downloadUrls: undefined,
  };
}

export function newImageFromBackendImage(backendImage: BackendImage): Image {
  return {
    id: backendImage.id,
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
    downloadUrls: backendImage.downloadUrls,
  };
}
