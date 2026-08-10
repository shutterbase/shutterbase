import { ImageTagAssignmentType, ImageWithTagsType } from "src/types/custom";
import { api } from "src/api";
import { emitter } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { canEditImageTag } from "src/pages/upload/uploadUtil";
import { tagLabel } from "src/util/tagOrder";

// Tag-removal permission and mutation, shared by the sidebar badge list and
// the zen-mode tag bar so both mirror the server rules and never offer a 403.

export function canRemoveTagAssignment(item: ImageWithTagsType | null, tagAssignment: ImageTagAssignmentType): boolean {
  const userStore = useUserStore();
  if (!userStore.isProjectEditorOrHigher()) return false;
  // The upload review flow freezes official tags once an upload is submitted and
  // reserves the error tag for reviewers.
  if (!canEditImageTag(item, tagAssignment.tag)) return false;
  const isOwnImage = item?.user.id === userStore.user?.id;
  const isProjectAdminOrHigher = userStore.isProjectAdminOrHigher();
  if (tagAssignment.tag.type === "default") {
    return isProjectAdminOrHigher;
  } else {
    return isOwnImage || isProjectAdminOrHigher;
  }
}

// Guards itself: ImageTagBadge emits remove on every click, permitted or not.
export async function removeTagAssignment(item: ImageWithTagsType | null, tagAssignment: ImageTagAssignmentType): Promise<void> {
  if (!canRemoveTagAssignment(item, tagAssignment)) {
    return;
  }
  try {
    await api.imageTagAssignments.remove(tagAssignment.id);
    emitter.emit(`notification`, {
      headline: `Tag ${tagLabel(tagAssignment.tag)} removed`,
      type: "success",
    });
    if (item) {
      item.tags.splice(
        item.tags.findIndex((ta) => ta.id === tagAssignment.id),
        1,
      );
      item.updatedAt = new Date().toISOString();
    }
  } catch (error: any) {
    emitter.emit(`notification`, {
      headline: `Error removing tag ${tagLabel(tagAssignment.tag)}`,
      type: "error",
    });
  }
}
