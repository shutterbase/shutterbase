import { Project } from "~/api/project";
import { Batch } from "~/api/batch";
import { User } from "~/api/user";
import { Camera } from "api/camera";
import { Tag } from "~/api/tag";

export interface Image {
  id: string;
  thumbnailId: string;
  fileName: string;
  computedFileName: string;
  description: string;
  exifData: object;
  edges: {
    tagAssignments: TagAssignment[];
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

export interface TagAssignment {
  id: string;
  type: string;
  edges: {
    tag: Tag;
    image: Image;
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}
