import { http } from "src/boot/axios";
import { CurrentUser } from "src/types/api";

export interface LoginBody {
  identifier: string;
  password: string;
}

export interface SignupBody {
  username: string;
  email: string;
  password: string;
  firstName: string;
  lastName: string;
  copyrightTag?: string;
}

// Public server info — carries whether self-signup is open.
export interface ServerInfo {
  version: string;
  signupEnabled: boolean;
}

export interface ChangePasswordBody {
  currentPassword: string;
  newPassword: string;
  newPasswordConfirm: string;
}

export async function me(): Promise<CurrentUser> {
  const { data } = await http.get<CurrentUser>("/users/me");
  return data;
}

export async function login(body: LoginBody): Promise<CurrentUser> {
  const { data } = await http.post<CurrentUser>("/auth/login", body);
  return data;
}

export async function logout(): Promise<void> {
  await http.post("/auth/logout");
}

// Always 202 "pending activation": accounts are created inactive and a platform
// admin has to activate them. Never reveals whether the identity already exists.
export async function signup(body: SignupBody): Promise<{ status: string; message: string }> {
  const { data } = await http.post("/auth/signup", body);
  return data;
}

export async function serverInfo(): Promise<ServerInfo> {
  const { data } = await http.get<ServerInfo>("/version");
  return data;
}

export async function changePassword(body: ChangePasswordBody): Promise<CurrentUser> {
  const { data } = await http.put<CurrentUser>("/auth/change-password", body);
  return data;
}

export async function impersonate(userId: string): Promise<CurrentUser> {
  const { data } = await http.post<CurrentUser>(`/auth/impersonate/${userId}`);
  return data;
}

export async function stopImpersonate(): Promise<CurrentUser> {
  const { data } = await http.delete<CurrentUser>("/auth/impersonate");
  return data;
}
