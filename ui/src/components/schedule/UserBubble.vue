<template>
  <img
    ref="img"
    :title="`${user.firstName} ${user.lastName}`"
    :alt="`${user.firstName} ${user.lastName}`"
    :class="['rounded-full ring-2 ring-surface dark:ring-surface-dark', sizeClass]"
    src=""
  />
</template>

<script setup lang="ts">
// Gravatar bubble with initials fallback — same avatar-initials mechanism as
// the navbar avatar, so photographers who set a Gravatar show their face on
// every schedule item they cover (transcript 29:34).
import Avatar from "avatar-initials";
import { computed, onMounted, ref, watch } from "vue";
import { EmbeddedUser } from "src/types/api";

const props = withDefaults(defineProps<{ user: EmbeddedUser; size?: "sm" | "md" }>(), { size: "sm" });

const img = ref<HTMLImageElement>();
const sizeClass = computed(() => (props.size === "md" ? "h-8 w-8" : "h-6 w-6"));

function render() {
  if (!img.value) return;
  Avatar.from(img.value, {
    useGravatar: !!props.user.email,
    email: props.user.email,
    initials: `${props.user.firstName?.charAt(0) ?? ""}${props.user.lastName?.charAt(0) ?? ""}`,
    color: "#FFFFFF",
    background: "#37465D",
    fontWeight: 400,
    size: props.size === "md" ? 64 : 48,
  });
}

onMounted(render);
watch(() => props.user, render);
</script>
