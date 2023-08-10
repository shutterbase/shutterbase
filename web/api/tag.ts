import { User } from "api/user";

export interface Tag {
  id: string;
  name: string;
  description: string;
  type: string;
  isAlbum: boolean;
  edges: {
    // images: Image[];
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface UpdateTagInput {
  name?: string;
  description?: string;
  type?: string;
  isAlbum?: boolean;
}

export interface CreateTagInput {
  name: string;
  description: string;
  type: string;
  isAlbum: boolean;
}

export interface CreateTagsInput {
  tags: CreateTagInput[];
}
