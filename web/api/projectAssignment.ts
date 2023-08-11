import { Project } from "api/project";
import { User, Role } from "api/user";

export interface ProjectAssignment {
  id: string;
  edges: {
    project: Project;
    user: User;
    role: Role;
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface UpdateProjectAssignmentInput {
  role: string;
}

export interface CreateProjectAssignmentInput {
  userId: string;
  role: string;
}
