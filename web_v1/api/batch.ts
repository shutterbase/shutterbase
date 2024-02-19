import { User } from "api/user";

export interface Batch {
  id: string;
  name: string;
  edges: {
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface UpdateBatchInput {
  name: string;
}

export interface CreateBatchInput {
  name: string;
}
