import { useUserStore } from "src/stores/user-store";
import { UploadsResponse } from "src/types/pocketbase";
import { EmbeddedTag, Image } from "src/types/api";
import { canEditTag } from "src/util/uploadReview";

const userStore = useUserStore();

export function showUploadEdit(item: UploadsResponse): boolean {
  return item.user?.id === userStore.user?.id || userStore.isProjectAdminOrHigher();
}

export function isUploadReadOnly(item: UploadsResponse): boolean {
  return !showUploadEdit(item);
}

// Store-aware wrapper around the pure review rules: may the current user still
// add or remove this tag on this image? Mirrors authorization.CanAssignTag, so
// the UI never offers an action the API would reject.
export function canEditImageTag(image: Pick<Image, "upload"> | null | undefined, tag: Pick<EmbeddedTag, "type" | "name">): boolean {
  return canEditTag({
    reviewEnabled: !!userStore.activeProject?.uploadReviewEnabled,
    uploadState: image?.upload?.state,
    tagType: tag.type,
    tagName: tag.name,
    isReviewer: userStore.isProjectAdminOrHigher(),
    isEditor: userStore.isProjectEditorOrHigher(),
  });
}

// True when the review flow has frozen this image's official tags for the
// current user (custom tags stay editable).
export function officialTagsFrozen(image: Pick<Image, "upload"> | null | undefined): boolean {
  return !canEditImageTag(image, { type: "manual", name: "" }) && userStore.isProjectEditorOrHigher();
}
