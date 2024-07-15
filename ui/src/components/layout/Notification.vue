<template>
  <div aria-live="assertive" class="pointer-events-none fixed inset-0 flex items-end px-4 py-20 sm:items-start">
    <div class="flex w-full flex-col items-center space-y-4 sm:items-end">
      <div
        v-for="notification in notifications"
        :key="notification.id"
        :class="[`pointer-events-auto w-full max-w-sm overflow-hidden rounded-lg bg-gray-50 dark:bg-primary-600 shadow-lg ring-1 ring-black ring-opacity-5`, notification.classes]"
      >
        <div class="p-4">
          <div class="flex items-start">
            <div class="flex-shrink-0">
              <CheckCircleIcon v-if="notification.type == 'success'" class="h-6 w-6 text-green-400"></CheckCircleIcon>
              <InformationCircleIcon v-else-if="notification.type == 'info'" class="h-6 w-6 text-blue-600 dark:text-blue-300"></InformationCircleIcon>
              <ExclamationTriangleIcon v-else-if="notification.type == 'warning'" class="h-6 w-6 text-orange-400"></ExclamationTriangleIcon>
              <ExclamationCircleIcon v-else-if="notification.type == 'error'" class="h-6 w-6 text-red-600"></ExclamationCircleIcon>
            </div>
            <div class="ml-3 w-0 flex-1 pt-0.5">
              <p class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ notification.headline }}</p>
              <p v-if="notification.message && notification.message.length != 0" class="mt-1 text-sm text-gray-500 dark:text-gray-300">{{ notification.message }}</p>
            </div>
            <div class="ml-4 flex flex-shrink-0">
              <button
                type="button"
                @click="notification.closeCallback"
                class="inline-flex rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
              >
                <span class="sr-only">Close</span>
                <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { emitter, NotificationEvent } from "src/boot/mitt";
import { nanoid } from "nanoid";
import { Ref, onUnmounted, ref } from "vue";

import { ExclamationCircleIcon, ExclamationTriangleIcon, InformationCircleIcon, CheckCircleIcon } from "@heroicons/vue/24/outline";

type Notification = NotificationEvent & {
  id: string;
  classes: string[];
  closeCallback: () => void;
};

const notifications: Ref<Notification[]> = ref([]);

const animationEnterStartClasses = "transform ease-out duration-300 transition translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2";
const animationEnterEndClasses = "transform ease-out duration-300 transition translate-y-0 opacity-100 sm:translate-x-0";
const animationExitStartClasses = "transition ease-in duration-100 opacity-100";
const animationExitEndClasses = "transition ease-in duration-100 opacity-0";

emitter.on("notification", (args: any) => {
  const notificationArgs: NotificationEvent = args;
  const id = nanoid();

  const TOTAL_ANIMATION_DURATION = notificationArgs.timeout || 3000;

  function setNotificationClasses(classes: string[]) {
    const notification = notifications.value.find((n) => n.id === id);
    if (notification) {
      notification.classes = classes;
    }
  }

  let timeouts: any[] = [];
  let closeTriggered = false;

  function closeNotification() {
    if (closeTriggered) {
      return;
    }
    closeTriggered = true;
    timeouts.forEach((t) => clearTimeout(t));
    timeouts = [];
    setNotificationClasses([animationExitStartClasses]);
    setTimeout(() => {
      setNotificationClasses([animationExitEndClasses]);
    }, 50);
    setTimeout(() => {
      notifications.value = notifications.value.filter((n) => n.id !== id);
    }, 150);
  }

  const notification = { ...notificationArgs, id, classes: [animationEnterStartClasses], closeCallback: closeNotification };

  timeouts.push(
    setTimeout(() => {
      setNotificationClasses([animationEnterEndClasses]);
    }, 100)
  );
  timeouts.push(setTimeout(closeNotification, TOTAL_ANIMATION_DURATION - 150));

  notifications.value.push(notification);
});

onUnmounted(() => {
  emitter.off("notification");
});
</script>
