import { User } from "api/user";

export interface Camera {
  id: string;
  name: string;
  description: string;
  edges: {
    owner: User;
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface UpdateCameraInput {
  name?: string;
  description?: string;
}

export interface CreateCameraInput {
  name: string;
  description: string;
}
