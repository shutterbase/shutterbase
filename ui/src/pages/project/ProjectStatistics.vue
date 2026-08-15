<template>
  <main class="mx-auto w-full max-w-7xl">
    <div class="px-4 sm:px-6 lg:px-8">
      <p class="label-mono text-accent-600 dark:text-accent-400">Project</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">Statistics</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Shooting volume, tagging effort and tag usage across this project.</p>
    </div>

    <!-- stats.totals gates the dashboard: during a rolling deploy an OLD pod can
         still answer with the tags-only payload — render just the tag table then
         instead of throwing on missing fields -->
    <div v-if="stats && stats.totals" class="mt-6 space-y-4 px-4 sm:px-6 lg:px-8">
      <StatTiles :tiles="tiles" />

      <div class="grid gap-4 lg:grid-cols-2">
        <section :class="[cardClass, 'lg:col-span-2']">
          <div class="flex items-center justify-between gap-2">
            <h2 class="label-mono text-primary-500 dark:text-primary-400">Images per day</h2>
            <div class="flex gap-1">
              <button
                v-for="mode in [false, true]"
                :key="String(mode)"
                type="button"
                @click="splitByPhotographer = mode"
                :class="[
                  'label-mono-sm rounded px-2 py-1 transition-colors',
                  splitByPhotographer === mode
                    ? 'bg-accent-500/10 text-accent-700 dark:bg-accent-400/15 dark:text-accent-300'
                    : 'text-primary-500 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-200',
                ]"
              >
                {{ mode ? "By photographer" : "Total" }}
              </button>
            </div>
          </div>
          <DayColumnChart class="mt-4" :days="imageDays" :legend="splitByPhotographer ? photographerSeries : undefined" />
        </section>

        <section :class="[cardClass, 'lg:col-span-2']">
          <h2 class="label-mono text-primary-500 dark:text-primary-400">Tags applied per day</h2>
          <p class="mt-1 text-xs text-primary-500 dark:text-primary-400">Manual vs AI. AI counts show the current tags by their last detection run.</p>
          <DayColumnChart class="mt-4" :days="assignmentDays" :legend="assignmentLegend" />
        </section>

        <section :class="[cardClass, 'lg:col-span-2']">
          <h2 class="label-mono text-primary-500 dark:text-primary-400">Shooting coverage</h2>
          <p class="mt-1 text-xs text-primary-500 dark:text-primary-400">Images per hour of day — gaps show uncovered event time.</p>
          <HourHeatmap class="mt-4" :days="stats.days" />
        </section>

        <section :class="cardClass">
          <h2 class="label-mono text-primary-500 dark:text-primary-400">Photographers</h2>
          <BarList class="mt-4" :items="photographerBars" />
        </section>

        <section :class="cardClass">
          <h2 class="label-mono text-primary-500 dark:text-primary-400">Top tags</h2>
          <BarList class="mt-4" :items="topTagBars" />
        </section>

        <section v-if="aiUsed" :class="cardClass">
          <h2 class="label-mono text-primary-500 dark:text-primary-400">AI detection</h2>
          <SegmentMeter class="mt-4" :segments="aiSegments" />
        </section>

        <section v-if="project?.uploadReviewEnabled" :class="cardClass">
          <h2 class="label-mono text-primary-500 dark:text-primary-400">Review funnel</h2>
          <SegmentMeter class="mt-4" :segments="reviewSegments" />
        </section>
      </div>
    </div>

    <div class="mt-8 px-4 sm:px-6 lg:px-8">
      <h2 class="label-mono text-primary-500 dark:text-primary-400">All tags</h2>
    </div>
    <Table class="mt-2" dense :items="imageTagStatistics" :columns="imageTagColumns" name="" :allow-add="false"></Table>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import Table, { TableColumn } from "src/components/Table.vue";
import StatTiles, { StatTile } from "src/components/statistics/StatTiles.vue";
import DayColumnChart from "src/components/statistics/DayColumnChart.vue";
import HourHeatmap from "src/components/statistics/HourHeatmap.vue";
import BarList, { BarListItem } from "src/components/statistics/BarList.vue";
import SegmentMeter, { MeterSegment } from "src/components/statistics/SegmentMeter.vue";
import { ProjectStatistics, TagStatistic } from "src/api/statistics";
import { Project } from "src/types/api";
import { tagLabel } from "src/util/tagOrder";
import { formatBytes } from "src/util/downloadRunner";
import { shortDayLabel } from "src/util/dateTimeUtil";
import { AI_COLOR, MANUAL_COLOR, CATEGORICAL, OTHER_COLOR } from "src/util/chartColors";
import { buildAssignmentDaySeries, buildImageDaySeries, foldPhotographers, peakDay, photographerLabel } from "src/util/statisticsView";
import { api } from "src/api";

const route = useRoute();

const cardClass = "rounded-lg border border-primary-200 bg-surface p-6 shadow-panel dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark";

const stats: Ref<ProjectStatistics | null> = ref(null);
const project: Ref<Project | null> = ref(null);
const splitByPhotographer = ref(false);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function loadData() {
  const projectId = `${route.params.id}`;
  if (!projectId) return;
  try {
    [stats.value, project.value] = await Promise.all([api.statistics.project(projectId), api.projects.get(projectId)]);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const tiles = computed<StatTile[]>(() => {
  if (!stats.value) return [];
  const { totals, days } = stats.value;
  const peak = peakDay(days);
  return [
    {
      label: "Images",
      value: `${totals.images}`,
      sub: totals.unbucketedImages > 0 ? `${totals.unbucketedImages} without capture time` : undefined,
    },
    { label: "Photographers", value: `${totals.photographers}` },
    { label: "Manual tags", value: `${totals.manualTags}` },
    {
      label: "AI tags",
      value: `${totals.aiTags}`,
      sub: totals.images > 0 ? `${Math.round((stats.value.aiStatus.done / totals.images) * 100)}% of images detected` : undefined,
    },
    { label: "Peak day", value: peak ? shortDayLabel(peak.date) : "—", sub: peak ? `${peak.total} images` : undefined },
    { label: "Storage", value: formatBytes(totals.storageBytes) },
  ];
});

const photographerSeries = computed(() => (stats.value ? foldPhotographers(stats.value.photographers) : []));
const imageDays = computed(() => (stats.value ? buildImageDaySeries(stats.value.days, photographerSeries.value, splitByPhotographer.value) : []));
const assignmentDays = computed(() => (stats.value ? buildAssignmentDaySeries(stats.value.assignmentsPerDay) : []));
const assignmentLegend = [
  { key: "manual", label: "Manual", color: MANUAL_COLOR },
  { key: "ai", label: "AI", color: AI_COLOR },
];

const photographerBars = computed<BarListItem[]>(() => {
  if (!stats.value) return [];
  const total = stats.value.totals.images;
  return stats.value.photographers.map((p, i) => ({
    key: p.id,
    label: photographerLabel(p),
    sub: total > 0 ? `${p.copyrightTag || "—"} · ${Math.round((p.imageCount / total) * 100)}%` : p.copyrightTag,
    value: p.imageCount,
    color: i < CATEGORICAL.length ? CATEGORICAL[i] : OTHER_COLOR,
  }));
});

const topTagBars = computed<BarListItem[]>(() => {
  if (!stats.value) return [];
  return [...stats.value.tags]
    .sort((a, b) => b.count - a.count)
    .slice(0, 10)
    .filter((t) => t.count > 0)
    .map((t) => ({ key: t.id, label: tagLabel(t), sub: t.type, value: t.count }));
});

// Status colors (reserved palette), always paired with label + count in the legend.
const aiUsed = computed(() => {
  if (!stats.value) return false;
  const s = stats.value.aiStatus;
  return s.done + s.inFlight + s.error > 0;
});
const aiSegments = computed<MeterSegment[]>(() => {
  if (!stats.value) return [];
  const s = stats.value.aiStatus;
  return [
    { key: "done", label: "Done", value: s.done, color: "#2c9152" },
    { key: "inFlight", label: "In flight", value: s.inFlight, color: "#6f8dff" },
    { key: "error", label: "Error", value: s.error, color: "#d44e42" },
    { key: "notQueued", label: "Not queued", value: s.notQueued, color: "#7986a1" },
  ];
});
const reviewSegments = computed<MeterSegment[]>(() => {
  if (!stats.value) return [];
  const s = stats.value.uploadStates;
  return [
    { key: "open", label: "Open", value: s.open, color: "#7986a1" },
    { key: "ready", label: "Ready", value: s.ready, color: "#bd7f2c" },
    { key: "reviewed", label: "Reviewed", value: s.reviewed, color: "#2c9152" },
  ];
});

// The detail table below the dashboard — unchanged behavior.
type TagRow = TagStatistic & { label: string };
const imageTagStatistics = computed<TagRow[]>(() => {
  if (!stats.value) return [];
  return [...stats.value.tags]
    .sort((a, b) => b.count - a.count)
    .map((tag) => ({
      ...tag,
      label: tagLabel(tag),
      description: tag.description && tag.description.length > 50 ? `${tag.description.substring(0, 47)}...` : tag.description,
    }));
});

const imageTagColumns: TableColumn<TagRow>[] = [
  { key: "label", label: "Name" },
  { key: "description", label: "Description" },
  { key: "type", label: "Type" },
  { key: "count", label: "Count" },
];

watch(route, loadData);
onMounted(loadData);
</script>
