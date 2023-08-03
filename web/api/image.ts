import { Project } from "~/api/project";
import { Batch } from "~/api/batch";
import { User } from "~/api/user";
import { Camera } from "api/camera";

export interface Image {
  id: string;
  thumbnailId: string;
  fileName: string;
  description: string;
  exifData: object;
  edges: {
    user: User;
    batch: Batch;
    project: Project;
    camera: Camera;
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface UpdateImageInput {
  name?: string;
  description?: string;
}
