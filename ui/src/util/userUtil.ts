import pb from "src/boot/pocketbase";
import { UsersResponse } from "src/types/pocketbase";

export function fullName(user?: UsersResponse): string {
  const userModel = user || pb.authStore.model;
  if (userModel) {
    return userModel.firstName + " " + userModel.lastName;
  } else {
    return "Unknown User";
  }
}

export function fullNamePossessive(user?: UsersResponse): string {
  const userFullName = fullName(user);
  return `${userFullName}'s`;
}
