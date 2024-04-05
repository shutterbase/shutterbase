import mitt from "mitt";

export const emitter = mitt();

export type NotificationEvent = {
  type: "success" | "error" | "warning" | "info";
  headline: string;
  message?: string;
};

export function showNotificationToast(event: NotificationEvent) {
  emitter.emit("notification", event);
}
