<template>
  <div class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <main v-if="g" class="max-w-3xl">
      <p class="label-mono text-accent-600 dark:text-accent-400">Gallery · {{ g.key }}</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">{{ g.name }}</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
        Everything the public site renders comes from this form. A deployment with <code>GALLERY_KEY={{ g.key }}</code> picks the changes up within its cache window (≤ 10 min).
      </p>

      <form class="mt-10 space-y-12" @submit.prevent="save">
        <section class="space-y-4">
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Site</h2>
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="block"><span class="label">Name</span><input v-model="g.name" class="input" required /></label>
            <label class="block"
              ><span class="label">Domain</span><input v-model="g.domain" class="input" placeholder="media.example.org" /><span class="hint"
                >Canonical host for links, OpenGraph and the sitemap</span
              ></label
            >
            <label class="block sm:col-span-2"><span class="label">Tagline</span><input v-model="g.tagline" class="input" /></label>
            <label class="block sm:col-span-2"><span class="label">About</span><textarea v-model="g.aboutText" rows="3" class="input h-auto py-2"></textarea></label>
            <label class="block sm:col-span-2"><span class="label">Footer text</span><input v-model="g.footerText" class="input" /></label>
            <label class="block"><span class="label">Imprint URL</span><input v-model="g.imprintUrl" class="input" placeholder="https://…" /></label>
            <label class="block"><span class="label">Privacy URL</span><input v-model="g.privacyUrl" class="input" placeholder="https://…" /></label>
            <label class="block"
              ><span class="label">Locale</span>
              <select v-model="g.locale" class="input">
                <option value="de">Deutsch</option>
                <option value="en">English</option>
              </select></label
            >
            <label class="flex items-center gap-3 self-end pb-2"
              ><input type="checkbox" v-model="g.active" class="h-4 w-4" /><span class="text-sm">Active — an inactive gallery serves nothing</span></label
            >
          </div>
        </section>

        <section class="space-y-4">
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Branding</h2>
          <div class="grid gap-4 sm:grid-cols-2">
            <AssetField label="Logo (light)" :value="g.logoStorageId" accept="image/png,image/webp,image/jpeg" @change="(k) => (g!.logoStorageId = k)" :upload="upload" />
            <AssetField label="Logo (dark)" :value="g.logoDarkStorageId" accept="image/png,image/webp,image/jpeg" @change="(k) => (g!.logoDarkStorageId = k)" :upload="upload" />
            <AssetField label="Favicon" :value="g.faviconStorageId" accept="image/png,image/x-icon" @change="(k) => (g!.faviconStorageId = k)" :upload="upload" />
            <AssetField label="Hero image" :value="g.heroStorageId" accept="image/jpeg,image/webp,image/png" @change="(k) => (g!.heroStorageId = k)" :upload="upload" />
          </div>
          <div class="grid gap-4 sm:grid-cols-3">
            <label class="block"><span class="label">Primary</span><input v-model="theme.primary" type="color" class="input p-1" /></label>
            <label class="block"><span class="label">Accent</span><input v-model="theme.accent" type="color" class="input p-1" /></label>
            <label class="block"><span class="label">Surface (light)</span><input v-model="theme.surface" type="color" class="input p-1" /></label>
            <label class="block"><span class="label">Surface (dark)</span><input v-model="theme.surfaceDark" type="color" class="input p-1" /></label>
            <label class="block"><span class="label">Heading font</span><input v-model="theme.fontHeading" class="input" placeholder="Google Fonts family, e.g. Inter" /></label>
            <label class="block"><span class="label">Body font</span><input v-model="theme.fontBody" class="input" placeholder="Google Fonts family" /></label>
            <label class="block"><span class="label">Corner radius</span><input v-model="theme.radius" class="input" placeholder="0.75rem" /></label>
            <label class="block"
              ><span class="label">Logo position</span>
              <select v-model="theme.logoPosition" class="input">
                <option value="">Left</option>
                <option value="center">Centered</option>
              </select></label
            >
            <label class="flex items-center gap-3 self-end pb-2"
              ><input type="checkbox" v-model="theme.defaultDark" class="h-4 w-4" /><span class="text-sm">Dark by default</span></label
            >
          </div>
        </section>

        <section class="space-y-4">
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Vocabulary</h2>
          <p class="text-sm text-primary-500 dark:text-primary-400">What the public site calls things. Empty = default for the locale.</p>
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="block"><span class="label">Project (singular)</span><input v-model="labels.projectSingular" class="input" placeholder="Event" /></label>
            <label class="block"><span class="label">Project (plural)</span><input v-model="labels.projectPlural" class="input" placeholder="Events" /></label>
            <label class="block"><span class="label">Day</span><input v-model="labels.dayLabel" class="input" /></label>
            <label class="block"><span class="label">Photographer</span><input v-model="labels.photographerLabel" class="input" /></label>
            <label class="block"><span class="label">All photos</span><input v-model="labels.allPhotosLabel" class="input" /></label>
          </div>
        </section>

        <section class="space-y-4">
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Footer links</h2>
          <div v-for="(l, i) in links" :key="i" class="flex gap-2">
            <input v-model="l.label" class="input" placeholder="Label" />
            <input v-model="l.url" class="input" placeholder="https://…" />
            <button type="button" class="btn-ghost" @click="links.splice(i, 1)">Remove</button>
          </div>
          <button type="button" class="btn-ghost" @click="links.push({ label: '', url: '' })">Add link</button>
        </section>

        <section class="space-y-4">
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Downloads</h2>
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="flex items-center gap-3"
              ><input type="checkbox" v-model="g.bulkDownloadEnabled" class="h-4 w-4" /><span class="text-sm">Offer ZIP downloads of a selection</span></label
            >
            <label class="block"><span class="label">Max images per ZIP</span><input v-model.number="g.bulkDownloadMaxImages" type="number" min="0" class="input" /></label>
          </div>
        </section>

        <div class="flex items-center gap-3">
          <button type="submit" :disabled="saving" class="btn">Save</button>
          <span v-if="savedAt" class="text-sm text-primary-500">Saved</span>
        </div>
      </form>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import AssetField from "src/pages/gallery/AssetField.vue";
import { api } from "src/api";
import { Gallery, GalleryLabels, GalleryTheme, SocialLink } from "src/types/api";
import { showNotificationToast } from "src/boot/mitt";

const route = useRoute();
const g = ref<Gallery | null>(null);
const theme = reactive<GalleryTheme>({});
const labels = reactive<GalleryLabels>({});
const links = ref<SocialLink[]>([]);
const saving = ref(false);
const savedAt = ref<number | null>(null);
const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function load() {
  try {
    const loaded = await api.galleries.get(`${route.params.id}`);
    g.value = loaded;
    Object.assign(theme, { primary: "#111827", accent: "#2563eb", surface: "#fafaf9", surfaceDark: "#0c0c0e", ...loaded.theme });
    Object.assign(labels, loaded.labels ?? {});
    links.value = [...(loaded.socialLinks ?? [])];
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function upload(file: File): Promise<string> {
  return api.galleries.uploadAsset(g.value!.id, file);
}

async function save() {
  if (!g.value) return;
  saving.value = true;
  try {
    const { id, key, createdAt, updatedAt, ...rest } = g.value;
    g.value = await api.galleries.update(id, {
      ...rest,
      theme: { ...theme },
      labels: { ...labels },
      socialLinks: links.value.filter((l) => l.label && l.url),
    });
    savedAt.value = Date.now();
    showNotificationToast({ headline: "Gallery saved", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>

<style scoped>
.label {
  @apply mb-1 block text-xs font-medium uppercase tracking-wide text-primary-500 dark:text-primary-400;
}
.hint {
  @apply mt-1 block text-xs text-primary-400;
}
.input {
  @apply h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600;
}
.btn {
  @apply inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 disabled:opacity-50;
}
.btn-ghost {
  @apply inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200;
}
</style>
