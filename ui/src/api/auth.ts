import { http } from "src/boot/axios";
import { CurrentUser } from "src/types/api";

export interface LoginBody {
  identifier: string;
  password: string;
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
