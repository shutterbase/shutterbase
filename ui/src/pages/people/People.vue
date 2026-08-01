<template>
  <div class="mx-auto w-full max-w-7xl px-4 pt-8 pb-12 sm:px-6 lg:px-8">
    <div class="flex items-baseline justify-between">
      <div>
        <h1 class="text-base font-semibold text-primary-900 dark:text-white">People</h1>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">Detected persons across all your projects, ranked by how often they appear.</p>
      </div>
      <span v-if="personsTotal > 0" class="label-mono-sm shrink-0 text-primary-500 dark:text-primary-400">{{ personsTotal.toLocaleString() }} people</span>
    </div>

    <div v-if="noAiServer" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">No AI server is configured for this instance.</div>

    <template v-else>
      <div class="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
        <div
          v-for="(person, i) in persons"
          :key="person.personRef"
          :class="[
            'rounded-lg border border-primary-200 p-3 text-left dark:border-primary-800',
            canBrowse ? 'cursor-pointer transition-colors hover:border-accent-400 dark:hover:border-accent-500' : '',
          ]"
          :role="canBrowse ? 'button' : undefined"
          :tabindex="canBrowse ? 0 : undefined"
          :title="canBrowse ? 'Show this person\'s photos in the gallery' : ''"
          @click="canBrowse && showInGallery(person)"
          @keydown.enter="canBrowse && showInGallery(person)"
        >
          <FaceCarousel v-if="person.sample" :key="`${person.personRef}:${person.count}`" :person-ref="person.personRef" :sample="person.sample" :total="person.count" />
          <div v-else class="flex aspect-square w-full items-center justify-center rounded-sm bg-primary-100 dark:bg-primary-900">
            <span class="label-mono-sm text-primary-400 dark:text-primary-600">no sample</span>
          </div>
          <div v-if="person.name || canReview" class="mt-2" @click.stop>
            <input
              v-if="editingRef === person.personRef"
              v-focus
              v-model="editingName"
              type="text"
              placeholder="Name"
              maxlength="100"
              @keydown.enter="saveName(person)"
              @keydown.esc="editingRef = null"
              @blur="saveName(person)"
              class="h-8 w-full rounded-md border border-primary-200 bg-surface px-2 text-sm text-primary-900 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100"
            />
            <div v-else class="flex items-center gap-1.5">
              <span v-if="person.name" class="truncate text-sm font-medium text-primary-900 dark:text-white">{{ person.name }}</span>
              <span v-else class="truncate text-sm italic text-primary-400 dark:text-primary-600">unnamed</span>
              <button
                v-if="canReview"
                @click="startNameEdit(person)"
                title="Name this person"
                class="shrink-0 cursor-pointer rounded p-0.5 text-primary-400 transition-colors hover:text-primary-700 dark:hover:text-primary-200"
              >
                <PencilIcon class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
          <div class="mt-2 flex items-baseline justify-between">
            <span class="label-mono-sm text-primary-500 dark:text-primary-400">#{{ i + 1 }}</span>
            <span class="text-sm font-medium text-primary-900 dark:text-white">{{ person.count.toLocaleString() }} photo{{ person.count === 1 ? "" : "s" }}</span>
          </div>
        </div>
      </div>
      <div v-if="personsLoading" class="mt-6 text-center text-sm text-primary-500 dark:text-primary-400">Loading…</div>
      <div v-else-if="persons.length === 0" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">No detected persons yet.</div>
      <div v-else-if="persons.length < personsTotal" class="mt-6 text-center">
        <button
          @click="loadMorePersons"
          class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
        >
          Load more
        </button>
      </div>
    </template>

    <!-- merge review: server-gated (projectAdmin on ≥1 project); hidden on 403 -->
    <div v-if="canReview && !noAiServer" class="mt-12 border-t border-primary-200 pt-8 dark:border-primary-800">
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

      <div v-if="reviewLoading" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">Loading…</div>
      <div v-else-if="!candidateEntry" class="mt-10 text-center text-sm text-primary-500 dark:text-primary-400">
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
          <div v-for="side in sides" :key="side.key" class="rounded-lg border border-primary-200 p-4 dark:border-primary-800">
            <div class="mb-3 flex items-baseline justify-between">
              <span class="text-sm font-medium text-primary-900 dark:text-white">
                {{ side.label }}
                <span v-if="side.name" class="text-accent-600 dark:text-accent-400">— {{ side.name }}</span>
              </span>
              <span class="label-mono-sm text-primary-500 dark:text-primary-400">{{ side.page0.total }} appearance{{ side.page0.total === 1 ? "" : "s" }}</span>
            </div>
            <FaceCarousel :key="side.key" :person-ref="side.ref" :page0="side.page0" class="mx-auto max-w-sm" />
          </div>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-center gap-3">
          <span class="label-mono-sm mr-2 text-primary-500 dark:text-primary-400">similarity {{ (candidateEntry.candidate.sim * 100).toFixed(0) }}%</span>
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

      <div class="mt-12 border-t border-primary-200 pt-8 dark:border-primary-800">
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
                <span class="label-mono-sm ml-1 text-primary-500 dark:text-primary-400">{{ mergeNames[mergePersonRef(m, side)] ?? mergePersonRef(m, side) }}</span>
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
              <p class="label-mono-sm mb-2 text-primary-500 dark:text-primary-400">cluster {{ expanded.side }} — its own faces across your projects</p>
              <FaceCarousel :key="`${expanded.key}:${expanded.side}`" :person-ref="expanded.personRef" raw class="max-w-52" />
            </div>
          </li>
        </ul>
      </div>
    </div>

    <!-- both merged clusters were named: the reviewer picks the final name -->
    <TransitionRoot as="template" :show="nameConflict !== null">
      <Dialog as="div" class="relative z-10" @close="nameConflict = null">
        <TransitionChild
          as="template"
          enter="ease-out duration-300"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="ease-in duration-200"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity" />
        </TransitionChild>
        <div class="fixed inset-0 z-10 w-screen overflow-y-auto">
          <div class="flex min-h-full items-center justify-center p-4">
            <TransitionChild
              as="template"
              enter="ease-out duration-300"
              enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
              enter-to="opacity-100 translate-y-0 sm:scale-100"
              leave="ease-in duration-200"
              leave-from="opacity-100 translate-y-0 sm:scale-100"
              leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            >
              <DialogPanel
                class="relative w-full max-w-md transform overflow-hidden rounded-lg border border-primary-200 bg-surface p-6 text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark"
              >
                <template v-if="nameConflict">
                  <DialogTitle as="h3" class="display text-lg text-primary-900 dark:text-white">Both clusters are named</DialogTitle>
                  <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">The merged person keeps one name — pick one or type a new one.</p>
                  <div class="mt-4 flex flex-wrap gap-2">
                    <button
                      v-for="option in nameConflict.options"
                      :key="option"
                      @click="nameConflict.chosen = option"
                      :class="[
                        'cursor-pointer rounded-md border px-3 py-1.5 text-sm font-medium transition-colors',
                        nameConflict.chosen === option
                          ? 'border-accent-500 bg-accent-600/10 text-accent-700 dark:text-accent-300'
                          : 'border-primary-200 text-primary-700 hover:border-primary-400 dark:border-primary-700 dark:text-primary-200 dark:hover:border-primary-500',
                      ]"
                    >
                      {{ option }}
                    </button>
                  </div>
                  <input
                    v-model="nameConflict.chosen"
                    type="text"
                    maxlength="100"
                    aria-label="Final name"
                    @keydown.enter="resolveNameConflict"
                    class="mt-3 h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100"
                  />
                  <div class="mt-5 flex flex-row-reverse gap-3">
                    <button
                      @click="resolveNameConflict"
                      :disabled="!nameConflict.chosen.trim()"
                      class="inline-flex cursor-pointer items-center justify-center rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 disabled:opacity-50"
                    >
                      Set name
                    </button>
                    <button
                      @click="nameConflict = null"
                      class="cursor-pointer px-3 py-2 text-sm font-medium text-primary-500 transition-colors hover:text-primary-900 dark:text-primary-400 dark:hover:text-white"
                    >
                      Keep as is
                    </button>
                  </div>
                </template>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>

    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useStorage } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { PencilIcon } from "@heroicons/vue/24/solid";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import FaceCarousel from "src/components/project/FaceCarousel.vue";
import { api } from "src/api";
import { AiMerge, AiMergeCandidate, AiPersonImage, AiPersonImagesPage, AiRankedPerson } from "src/api/ai";
import { faceRendition } from "src/util/aiDetection";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";

const router = useRouter();
const { activeProjectId } = storeToRefs(useUserStore());

const PERSONS_PAGE_SIZE = 24;

const noAiServer = ref(false);

// --- ranked persons -----------------------------------------------------------
const persons = ref<AiRankedPerson[]>([]);
const personsTotal = ref(0);
const personsPage = ref(0);
const personsLoading = ref(true);
// the gallery person filter needs a project anchor; without an active project
// the cards are plain tiles
const canBrowse = computed(() => !!activeProjectId.value);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function fail(error: any) {
  if (error?.response?.status === 501) {
    noAiServer.value = true;
  } else {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function loadPersons(reset = false) {
  personsLoading.value = true;
  try {
    if (reset) {
      personsPage.value = 0;
      persons.value = [];
    }
    const page = await api.ai.rankedPersons(personsPage.value, PERSONS_PAGE_SIZE);
    personsTotal.value = page.total;
    persons.value.push(...page.items);
  } catch (error: any) {
    fail(error);
  } finally {
    personsLoading.value = false;
  }
}

async function loadMorePersons() {
  personsPage.value++;
  await loadPersons();
}

function showInGallery(person: AiRankedPerson) {
  router.push({ name: "images", query: { person: person.personRef, personScope: "all" } });
}

// --- person naming -------------------------------------------------------------
const vFocus = { mounted: (el: HTMLElement) => el.focus() };
const editingRef = ref<string | null>(null);
const editingName = ref("");

function startNameEdit(person: AiRankedPerson) {
  editingRef.value = person.personRef;
  editingName.value = person.name ?? "";
}

// Enter saves and the following blur no-ops (editingRef is already cleared);
// Esc clears editingRef so the blur no-ops too.
async function saveName(person: AiRankedPerson) {
  if (editingRef.value !== person.personRef) return;
  editingRef.value = null;
  const name = editingName.value.trim();
  if (name === (person.name ?? "")) return;
  try {
    await api.ai.setPersonName(person.personRef, name);
    person.name = name || undefined;
  } catch (error: any) {
    fail(error);
  }
}

// --- merge review (moved here from project settings — it is cross-project) ----
const canReview = ref(true); // server-gated: a 403 on the queue hides the section
const busy = ref(false);
const reviewed = ref(0);
// Skipped pairs survive reloads (localStorage). Keys from an older clustering
// generation never match and are dropped via the reset link.
// ponytail: capped at 300 keys.
const skippedStore = useStorage<string[]>("shutterbase-face-merge-skipped-global", []);
const skippedCount = computed(() => skippedStore.value.length);

// Candidates are prefetched PREFETCH deep (pair + one page of face samples per
// side, first thumbnails warmed), so a "different"/"skip" verdict shows the
// next pair instantly. The queue mirrors a prefix of the AI server's candidate
// list: `cursor` counts consumed server offsets (queued + locally skipped);
// deciding the head shifts every later offset down by one (cursor--). A merge
// may reshuffle candidates around the merged pair, so "same" flushes the queue.
const PREFETCH = 10;
const SIDE_PAGE_SIZE = 8;
interface ReviewEntry {
  candidate: AiMergeCandidate;
  a: AiPersonImagesPage;
  b: AiPersonImagesPage;
  names: Record<string, string>; // cluster names by personRef (absent = unnamed)
  remaining: number; // as reported by the server when this entry was fetched
}
const queue = ref<ReviewEntry[]>([]);
const exhausted = ref(false);
let cursor = 0;
let refillGen = 0;
let refilling = false;

const candidateEntry = computed(() => queue.value[0] ?? null);
const remaining = computed(() => candidateEntry.value?.remaining ?? 0);
const reviewLoading = computed(() => !candidateEntry.value && !exhausted.value);

const sides = computed(() => {
  const entry = candidateEntry.value;
  if (!entry) return [];
  const key = pairKey(entry.candidate);
  return [
    { key: `${key}:A`, ref: entry.candidate.personA, label: "Cluster A", page0: entry.a, name: entry.names[entry.candidate.personA] },
    { key: `${key}:B`, ref: entry.candidate.personB, label: "Cluster B", page0: entry.b, name: entry.names[entry.candidate.personB] },
  ];
});

const merges = ref<AiMerge[]>([]);
const expanded = ref<{ key: string; side: "A" | "B"; personRef: string } | null>(null);

function pairKey(c: AiMergeCandidate) {
  return `${c.personA}/${c.personB}`;
}

function mergeKey(m: AiMerge) {
  return `${m.personA}/${m.personB}`;
}

function isExpanded(m: AiMerge, side: "A" | "B") {
  return expanded.value?.key === mergeKey(m) && expanded.value?.side === side;
}

function reviewFail(error: any) {
  if (error?.response?.status === 403) {
    canReview.value = false;
    return;
  }
  fail(error);
}

function preload(item?: AiPersonImage) {
  if (!item) return;
  const size = faceRendition(item, item.image.width ?? 0, item.image.height ?? 0);
  const url = item.image.downloadUrls?.[size] ?? item.image.downloadUrls?.["512"];
  if (url) new Image().src = url;
}

function flushQueue() {
  refillGen++;
  refilling = false;
  queue.value = [];
  cursor = 0;
  exhausted.value = false;
}

async function refill() {
  if (refilling || exhausted.value) return;
  refilling = true;
  const gen = refillGen;
  try {
    const skipped = new Set(skippedStore.value);
    // ponytail: cursor capped at 500 — with that many locally skipped pairs in
    // one generation the server needs a skip-aware endpoint anyway.
    while (queue.value.length < PREFETCH && cursor < 500) {
      const next = await api.ai.mergeNext(cursor);
      if (gen !== refillGen) return;
      if (!next.candidate) {
        exhausted.value = true;
        break;
      }
      cursor++;
      const candidate = next.candidate;
      if (skipped.has(pairKey(candidate))) continue;
      // a decide racing an in-flight fetch can hand back a pair already queued
      if (queue.value.some((e) => pairKey(e.candidate) === pairKey(candidate))) continue;
      const [a, b, names] = await Promise.all([
        api.ai.personImagesGlobal(candidate.personA, 0, SIDE_PAGE_SIZE),
        api.ai.personImagesGlobal(candidate.personB, 0, SIDE_PAGE_SIZE),
        api.ai.personNames([candidate.personA, candidate.personB]).catch(() => ({}) as Record<string, string>),
      ]);
      if (gen !== refillGen) return;
      queue.value.push({ candidate, a, b, names, remaining: next.remaining });
      preload(a.items[0]);
      preload(b.items[0]);
    }
    if (cursor >= 500) exhausted.value = true;
  } catch (error: any) {
    if (gen !== refillGen) return;
    if (!error?.response) {
      // response-less = transient network blip (pod roll, flaky wifi) — a
      // background prefetch must self-heal, not raise the error modal. With
      // an empty queue nothing else re-triggers refill, so retry on a timer.
      if (!queue.value.length) setTimeout(() => refill(), 5000);
    } else {
      reviewFail(error);
    }
  } finally {
    if (gen === refillGen) refilling = false;
  }
}

const mergeNames = ref<Record<string, string>>({});

function mergePersonRef(m: AiMerge, side: "A" | "B") {
  return side === "A" ? m.personA : m.personB;
}

async function loadMerges() {
  try {
    merges.value = await api.ai.merges();
    const refs = [...new Set(merges.value.flatMap((m) => [m.personA, m.personB]))];
    mergeNames.value = refs.length ? await api.ai.personNames(refs.slice(0, 200)) : {};
  } catch (error: any) {
    reviewFail(error);
  }
}

function toggleExpand(m: AiMerge, side: "A" | "B") {
  if (isExpanded(m, side)) {
    expanded.value = null;
    return;
  }
  expanded.value = { key: mergeKey(m), side, personRef: side === "A" ? m.personA : m.personB };
}

async function decide(verdict: "same" | "different") {
  const entry = candidateEntry.value;
  if (!entry) return;
  busy.value = true;
  try {
    await api.ai.mergeDecide(entry.candidate.personA, entry.candidate.personB, verdict);
    reviewed.value++;
    if (verdict === "same") {
      showNotificationToast({ headline: "Clusters merged", type: "success" });
      const nameA = entry.names[entry.candidate.personA];
      const nameB = entry.names[entry.candidate.personB];
      if (nameA && nameB && nameA !== nameB) {
        nameConflict.value = { personRef: entry.candidate.personA, options: [nameA, nameB], chosen: nameA };
      }
      flushQueue();
      refill();
      await Promise.all([loadMerges(), loadPersons(true)]);
    } else {
      queue.value.shift();
      cursor--;
      refill();
    }
  } catch (error: any) {
    fail(error);
  } finally {
    busy.value = false;
  }
}

async function unmerge(m: AiMerge) {
  busy.value = true;
  try {
    await api.ai.deleteMerge(m.personA, m.personB);
    showNotificationToast({ headline: "Clusters unmerged — the pair is back in the review queue", type: "success" });
    expanded.value = null;
    flushQueue();
    refill();
    await Promise.all([loadMerges(), loadPersons(true)]);
  } catch (error: any) {
    fail(error);
  } finally {
    busy.value = false;
  }
}

function skipPair() {
  const entry = candidateEntry.value;
  if (!entry) return;
  skippedStore.value = [...skippedStore.value, pairKey(entry.candidate)].slice(-300);
  queue.value.shift(); // cursor unchanged — the pair still occupies its server offset
  refill();
}

function resetSkipped() {
  skippedStore.value = [];
  flushQueue();
  refill();
}

// Both merged clusters were named: the server keeps the conflicting names
// untouched until the reviewer picks the final one here.
const nameConflict = ref<{ personRef: string; options: string[]; chosen: string } | null>(null);

async function resolveNameConflict() {
  if (!nameConflict.value) return;
  const { personRef, chosen } = nameConflict.value;
  const name = chosen.trim();
  nameConflict.value = null;
  if (!name) return;
  try {
    await api.ai.setPersonName(personRef, name);
    showNotificationToast({ headline: `Merged person named “${name}”`, type: "success" });
    await Promise.all([loadMerges(), loadPersons(true)]);
  } catch (error: any) {
    fail(error);
  }
}

onMounted(() => {
  loadPersons(true);
  refill();
  loadMerges();
});
</script>
