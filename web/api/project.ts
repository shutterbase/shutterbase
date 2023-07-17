import { Role, User } from "api/user";

export interface Project {
  id: string;
  name: string;
  description: string;
  edges: {
    assignments: ProjectAssignment[];
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface UpdateProjectInput {
  name?: string;
  description?: string;
}

export interface CreateProjectInput {
  name: string;
  description: string;
}

interface ProjectAssignment {
  id: string;
  edges: {
    user: User;
    project: Project;
    role: Role;
  };
}
