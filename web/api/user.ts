import { ListRequestOptions, ListResult, SingleResult, requestList, requestSingle, requestUpdate } from "./common";

export interface User {
  id: string;
  firstName: string;
  lastName: string;
  copyrightTag: string;
  email: string;
  emailValidated: boolean;
  active: boolean;
  edges: {
    role: Role;
    createdBy: User;
    updatedBy: User;
  };
  createdAt: string;
  updatedAt: string;
}

export interface Role {
  id: string;
  key: string;
  description: string;
}

export interface UpdateUserInput {
  firstName?: string;
  lastName?: string;
  copyrightTag?: string;
  active?: boolean;
  emailValidated?: boolean;
  password?: string;
  role?: string;
}

export async function loadOwnUser(): Promise<SingleResult<User>> {
  return await requestSingle("/users/me");
}

export async function getUserById(id: string): Promise<SingleResult<User>> {
  return await requestSingle(`/users/${id}`);
}

export async function getUsers(options: ListRequestOptions): Promise<ListResult<User>> {
  return await requestList("/users", options);
}

export async function updateUserRole(id: string, role: string): Promise<SingleResult<User>> {
  return await requestUpdate(`/users/${id}/role`, { role });
}

export async function getRoles(): Promise<ListResult<Role>> {
  return await requestList("/roles", {});
}

export async function updateUser(id: string, data: UpdateUserInput): Promise<SingleResult<User>> {
  return await requestUpdate(`/users/${id}`, { data });
}
