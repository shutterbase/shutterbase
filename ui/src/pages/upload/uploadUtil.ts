import { useUserStore } from "src/stores/user-store";
import { UploadsResponse } from "src/types/pocketbase";

const userStore = useUserStore();

export function showUploadEdit(item: UploadsResponse): boolean {
  return item.user === userStore.user?.id || userStore.isProjectAdminOrHigher();
}

export function isUploadReadOnly(item: UploadsResponse): boolean {
  return !showUploadEdit(item);
}
