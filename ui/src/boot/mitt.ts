import mitt, { Emitter } from "mitt";

type Events = {
  notification: NotificationEvent;
  "show-hotkey-help": void;
  "show-tagging-dialog": void;
  "reset-tagging-dialog": void;
  "update-image-grid-scroll-position": void;
  "current-image-deleted": string; // ID of the deleted image
};

export const emitter: Emitter<Events> = mitt<Events>();

export type NotificationEvent = {
  type: "success" | "error" | "warning" | "info";
  headline: string;
  message?: string;
  timeout?: number;
};

export function showNotificationToast(event: NotificationEvent) {
  emitter.emit("notification", event);
}
