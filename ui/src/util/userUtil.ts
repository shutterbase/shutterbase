import { useUserStore } from "src/stores/user-store";

type NamedUser = { firstName: string; lastName: string };

export function fullName(user?: NamedUser | null): string {
  const userModel = user || useUserStore().user;
  if (userModel) {
    return userModel.firstName + " " + userModel.lastName;
  } else {
    return "Unknown User";
  }
}

export function fullNamePossessive(user?: NamedUser | null): string {
  const userFullName = fullName(user);
  return `${userFullName}'s`;
}
