<template>
  <div class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <div class="flex items-baseline justify-between">
      <div>
        <h2 class="text-base font-semibold text-primary-900 dark:text-white">Face cluster review</h2>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">
          The AI server suggests pairs of face clusters that might be the same person. Confirming a pair merges them into one — reversibly, see below.
        </p>
      </div>
      <div class="flex shrink-0 items-baseline gap-3">
        <span v-if="remaining > 0" class="label-mono-sm text-primary-500 dark:text-primary-400">{{ remaining }} pair{{ remaining === 1 ? "" : "s" }} left</span>
        <button
          v-if="skippedCount > 0"
          @click="resetSkipped"
          class="label-mono-sm cursor-pointer text-primary-500 underline transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-200"
          title="Skipped pairs are hidden across reloads — show them again"
        >
          {{ skippedCount }} skipped — show again
        </button>
      </div>
    </div>

    <div v-if="noAiServer" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">No AI server is configured for this instance.</div>
    <div v-else-if="loading" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">Loading…</div>
    <div v-else-if="!candidate" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">
      {{
        skippedCount > 0
          ? "Only skipped pairs remain — use “show again” above to review them."
          : reviewed > 0
            ? "All caught up — no more similar clusters to review."
            : "No similar clusters to review."
      }}
    </div>

    <template v-else>
      <div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2">
        <div v-for="side in sides" :key="side.ref" class="rounded-lg border border-primary-200 p-4 dark:border-primary-800">
          <div class="mb-3 flex items-baseline justify-between">
            <span class="text-sm font-medium text-primary-900 dark:text-white">{{ side.label }}</span>
            <span class="label-mono-sm text-primary-500 dark:text-primary-400">{{ side.total }} appearance{{ side.total === 1 ? "" : "s" }}</span>
          </div>
          <FaceClusterGrid :items="side.items" />
        </div>
      </div>

      <div class="mt-6 flex flex-wrap items-center justify-center gap-3">
        <span class="label-mono-sm mr-2 text-primary-500 dark:text-primary-400">similarity {{ (candidate.sim * 100).toFixed(0) }}%</span>
        <button
          @click="decide('same')"
          :disabled="busy"
          class="inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 disabled:opacity-50"
        >
          Same person — merge
        </button>
        <button
          @click="decide('different')"
          :disabled="busy"
          class="inline-flex items-center justify-center gap-1.5 rounded-md bg-primary-200 px-4 py-2 text-sm font-semibold text-primary-900 shadow-sm transition-colors hover:bg-primary-300 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 disabled:opacity-50 dark:bg-primary-800 dark:text-white dark:hover:bg-primary-700"
        >
          Different people
        </button>
        <button
          @click="skipPair"
          :disabled="busy"
          class="cursor-pointer px-3 py-2 text-sm font-medium text-primary-500 transition-colors hover:text-primary-900 disabled:opacity-50 dark:text-primary-400 dark:hover:text-white"
        >
          Skip
        </button>
      </div>
    </template>

    <div v-if="!noAiServer" class="mt-12 border-t border-primary-200 pt-8 dark:border-primary-800">
      <h2 class="text-base font-semibold text-primary-900 dark:text-white">Merged clusters</h2>
      <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">
        Merges are reversible: unmerging splits the two clusters again and the pair returns to the review queue. Click a cluster to inspect its own faces.
      </p>

      <div v-if="merges.length === 0" class="mt-6 text-sm text-primary-500 dark:text-primary-400">No merged clusters yet.</div>
      <ul v-else class="mt-4 divide-y divide-primary-200 dark:divide-primary-800">
        <li v-for="m in merges" :key="mergeKey(m)" class="py-3">
          <div class="flex flex-wrap items-center gap-3">
            <button
              v-for="side in ['A', 'B'] as const"
              :key="side"
              @click="toggleExpand(m, side)"
              :class="[
                'cursor-pointer rounded-md border px-3 py-1.5 text-sm font-medium transition-colors',
                isExpanded(m, side)
                  ? 'border-accent-500 bg-accent-600/10 text-accent-700 dark:text-accent-300'
                  : 'border-primary-200 text-primary-700 hover:border-primary-400 dark:border-primary-700 dark:text-primary-200 dark:hover:border-primary-500',
              ]"
            >
              Cluster {{ side }}
              <span class="label-mono-sm ml-1 text-primary-500 dark:text-primary-400">{{ side === "A" ? m.personA : m.personB }}</span>
            </button>
            <span class="label-mono-sm text-primary-500 dark:text-primary-400">merged {{ dateTimeUtil.dateTimeFromBackend(m.createdAt) }}</span>
            <button
              @click="unmerge(m)"
              :disabled="busy"
              class="ml-auto cursor-pointer px-3 py-1.5 text-sm font-medium text-red-600 transition-colors hover:text-red-500 disabled:opacity-50 dark:text-red-400 dark:hover:text-red-300"
            >
              Unmerge
            </button>
          </div>
          <div v-if="expanded && expanded.key === mergeKey(m)" class="mt-3">
            <p class="label-mono-sm mb-2 text-primary-500 dark:text-primary-400">
              cluster {{ expanded.side }} — {{ expanded.total }} appearance{{ expanded.total === 1 ? "" : "s" }} in this project
            </p>
            <FaceClusterGrid :items="expanded.items" :cols="4" />
          </div>
        </li>
      </ul>
    </div>

    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { useStorage } from "@vueuse/core";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import FaceClusterGrid from "src/components/project/FaceClusterGrid.vue";
import { api } from "src/api";
import { AiMerge, AiMergeCandidate, AiPersonImage } from "src/api/ai";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { showNotificationToast } from "src/boot/mitt";

const route = useRoute();
const projectId = computed(() => `${route.params.id}`);

const loading = ref(true);
const busy = ref(false);
const noAiServer = ref(false);
const remaining = ref(0);
const reviewed = ref(0);
// Skipped pairs survive reloads (localStorage, keyed by project). Keys from
// an older clustering generation never match and are dropped via the reset
// link. ponytail: capped at 300 keys per project.
const skippedStore = useStorage<Record<string, string[]>>("shutterbase-face-merge-skipped", {});
const skippedCount = computed(() => (skippedStore.value[projectId.value] ?? []).length);
const candidate = ref<AiMergeCandidate | null>(null);
const samplesA = ref<{ items: AiPersonImage[]; total: number }>({ items: [], total: 0 });
const samplesB = ref<{ items: AiPersonImage[]; total: number }>({ items: [], total: 0 });

const merges = ref<AiMerge[]>([]);
const expanded = ref<{ key: string; side: "A" | "B"; items: AiPersonImage[]; total: number } | null>(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const sides = computed(() => [
  { ref: candidate.value?.personA ?? "a", label: "Cluster A", ...samplesA.value },
  { ref: candidate.value?.personB ?? "b", label: "Cluster B", ...samplesB.value },
]);

function mergeKey(m: AiMerge) {
  return `${m.personA}/${m.personB}`;
}

function isExpanded(m: AiMerge, side: "A" | "B") {
  return expanded.value?.key === mergeKey(m) && expanded.value?.side === side;
}

function fail(error: any) {
  if (error?.response?.status === 501) {
    noAiServer.value = true;
  } else {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function pairKey(c: AiMergeCandidate) {
  return `${c.personA}/${c.personB}`;
}

async function load() {
  loading.value = true;
  candidate.value = null;
  try {
    // step past locally skipped pairs — the server has no skip state.
    // ponytail: sequential refetch, capped; batch endpoint if anyone ever
    // skips hundreds in one generation.
    const skipped = new Set(skippedStore.value[projectId.value] ?? []);
    let offset = 0;
    let next = await api.ai.mergeNext(projectId.value, offset);
    while (next.candidate && skipped.has(pairKey(next.candidate)) && offset < 200) {
      offset++;
      next = await api.ai.mergeNext(projectId.value, offset);
    }
    remaining.value = next.remaining;
    if (next.candidate) {
      const [a, b] = await Promise.all([api.ai.personImages(projectId.value, next.candidate.personA, 0, 4), api.ai.personImages(projectId.value, next.candidate.personB, 0, 4)]);
      samplesA.value = { items: a.items, total: a.total };
      samplesB.value = { items: b.items, total: b.total };
      candidate.value = next.candidate;
    }
  } catch (error: any) {
    fail(error);
  } finally {
    loading.value = false;
  }
}

async function loadMerges() {
  try {
    merges.value = await api.ai.merges(projectId.value);
  } catch (error: any) {
    fail(error);
  }
}

async function toggleExpand(m: AiMerge, side: "A" | "B") {
  if (isExpanded(m, side)) {
    expanded.value = null;
    return;
  }
  try {
    const personRef = side === "A" ? m.personA : m.personB;
    const page = await api.ai.personImages(projectId.value, personRef, 0, 8, true);
    expanded.value = { key: mergeKey(m), side, items: page.items, total: page.total };
  } catch (error: any) {
    fail(error);
  }
}

async function decide(verdict: "same" | "different") {
  if (!candidate.value) return;
  busy.value = true;
  try {
    await api.ai.mergeDecide(projectId.value, candidate.value.personA, candidate.value.personB, verdict);
    reviewed.value++;
    if (verdict === "same") {
      showNotificationToast({ headline: "Clusters merged", type: "success" });
      await loadMerges();
    }
    await load();
  } catch (error: any) {
    fail(error);
  } finally {
    busy.value = false;
  }
}

async function unmerge(m: AiMerge) {
  busy.value = true;
  try {
    await api.ai.deleteMerge(projectId.value, m.personA, m.personB);
    showNotificationToast({ headline: "Clusters unmerged — the pair is back in the review queue", type: "success" });
    expanded.value = null;
    await Promise.all([loadMerges(), load()]);
  } catch (error: any) {
    fail(error);
  } finally {
    busy.value = false;
  }
}

async function skipPair() {
  if (!candidate.value) return;
  const list = skippedStore.value[projectId.value] ?? [];
  skippedStore.value = { ...skippedStore.value, [projectId.value]: [...list, pairKey(candidate.value)].slice(-300) };
  await load();
}

async function resetSkipped() {
  skippedStore.value = { ...skippedStore.value, [projectId.value]: [] };
  await load();
}

onMounted(() => {
  load();
  loadMerges();
});
</script>
