import { ImageTagAssignmentsResponse, ImageTagsResponse, ImagesResponse, ProjectsResponse } from "src/types/pocketbase";

type DownloadUrls = {
  256: string;
  512: string;
  1024: string;
  2048: string;
  original: string;
};

export type ImageTagAssignmentType = ImageTagAssignmentsResponse & {
  expand: {
    imageTag: ImageTagsResponse;
  };
};

export type ImageWithTagsType = ImagesResponse & {
  downloadUrls: DownloadUrls;
  expand: {
    image_tag_assignments_via_image: ImageTagAssignmentType[];
  };
};

export type ProjectWithTagsType = ProjectsResponse & {
  expand: {
    image_tags_via_project: ImageTagsResponse[];
  };
};
