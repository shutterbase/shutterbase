<template>
  <main class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl space-y-12">
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Project Information"
        subtitle="General information concerning this project"
        :fields="informationFields"
        :item="item"
      />
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Event Period"
        subtitle="Frames the schedule calendar — from first to last event day"
        :fields="periodFields"
        :item="item"
      />
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Copyright Data"
        subtitle="Copyright information to be embedded into the EXIF data"
        :fields="copyrightFields"
        :item="item"
      />
      <div>
        <DetailEditGroup
          :allow-edit="userStore.isProjectAdminOrHigher()"
          @edit-save="saveItem"
          headline="AI Options"
          subtitle="Options for AI image tagging"
          :fields="aiFields"
          :item="item"
        />
        <div v-if="userStore.isProjectAdminOrHigher()" class="mt-4 flex flex-wrap gap-3">
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="rerunFailed"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Re-queue failed images
          </button>
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="showRecomputeConfirm = true"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Recompute all images
          </button>
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="showRerunNumbersConfirm = true"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Re-read car numbers
          </button>
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="showReclusterConfirm = true"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Recluster faces
          </button>
        </div>
      </div>
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Upload Review"
        subtitle="Let photographers submit uploads for review before their tags are final"
        :fields="reviewFields"
        :item="item"
      />
      <!-- MQTT / WLED integration -->
      <div v-if="userStore.isProjectAdminOrHigher()">
        <div class="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
          <div class="min-w-0 flex-1">
            <h2 class="display text-xl text-primary-900 dark:text-white">MQTT / WLED Integration</h2>
            <p class="mt-1 max-w-prose text-sm text-primary-500 dark:text-primary-400">Publish upload events to an MQTT broker for WLED and other smart-home devices</p>
            <div
              :class="[
                'mt-2 inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
                mqttConfigured
                  ? mqttReachable
                    ? 'border-success-400/60 bg-success-500/10 text-success-700 dark:text-success-300'
                    : 'border-warning-400/60 bg-warning-500/10 text-warning-700 dark:text-warning-300'
                  : 'border-primary-300 bg-transparent text-primary-400 dark:border-primary-700 dark:text-primary-500',
              ]"
            >
              <span
                :class="[
                  'h-1.5 w-1.5 rounded-full',
                  mqttConfigured ? (mqttReachable ? 'bg-success-500' : 'bg-warning-500') : 'bg-primary-400',
                ]"
              ></span>
              {{ mqttConfigured ? (mqttReachable ? 'Broker reachable' : 'Broker unreachable') : 'Not configured' }}
            </div>
            <p v-if="mqttConfigured && mqttError" class="mt-2 text-xs text-error-600 dark:text-error-400">{{ mqttError }}</p>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <button
              v-if="mqttEditing"
              type="button"
              @click="cancelMqttEdit"
              class="inline-flex items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
            >
              Cancel
            </button>
            <button
              v-if="mqttEditing"
              type="button"
              @click="saveMqttSettings"
              :disabled="savingMqtt"
              class="inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 disabled:opacity-50"
            >
              <ArrowPathIcon v-if="savingMqtt" class="h-4 w-4 animate-spin" />
              Save
            </button>
            <button
              v-else
              type="button"
              @click="startMqttEdit"
              class="inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950"
            >
              Edit
            </button>
          </div>
        </div>

        <!-- Display mode -->
        <div v-if="!mqttEditing" class="mt-4 space-y-4 rounded-md border border-primary-200 dark:border-primary-700 p-4">
          <dl class="grid grid-cols-1 gap-x-4 gap-y-2 sm:grid-cols-2">
            <div>
              <dt class="text-xs font-medium text-primary-500 dark:text-primary-400">Broker</dt>
              <dd class="mt-0.5 text-sm text-primary-900 dark:text-primary-100">{{ mqttForm.broker || '—' }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium text-primary-500 dark:text-primary-400">Client ID</dt>
              <dd class="mt-0.5 text-sm text-primary-900 dark:text-primary-100">{{ mqttForm.clientId || '—' }}</dd>
            </div>
          </dl>
          <div class="border-t border-primary-200 dark:border-primary-700 pt-3">
            <div class="flex items-center gap-2">
              <span
                :class="[
                  'h-2 w-2 rounded-full',
                  mqttForm.publishEvents ? 'bg-success-500' : 'bg-primary-300 dark:bg-primary-600',
                ]"
              ></span>
              <span class="text-sm font-medium text-primary-700 dark:text-primary-300">General Events</span>
              <span class="text-xs text-primary-500 dark:text-primary-400">{{ mqttForm.publishEvents ? 'Enabled' : 'Disabled' }}</span>
            </div>
            <p v-if="mqttForm.publishEvents" class="mt-1 ml-4 text-xs text-primary-400 dark:text-primary-500">
              Topics: <code class="font-mono">{{ mqttForm.topicPrefix || 'shutterbase' }}/{{ route.params.id }}/upload/{uploadId}/{event}</code>
              <br />Events: <span v-for="(ev, i) in mqttEventList" :key="ev.key" class="font-mono">{{ i > 0 ? ', ' : '' }}{{ ev.slug }}</span>
            </p>
          </div>
          <div class="border-t border-primary-200 dark:border-primary-700 pt-3">
            <div class="flex items-center gap-2">
              <span
                :class="[
                  'h-2 w-2 rounded-full',
                  mqttForm.wledControl ? 'bg-success-500' : 'bg-primary-300 dark:bg-primary-600',
                ]"
              ></span>
              <span class="text-sm font-medium text-primary-700 dark:text-primary-300">WLED Control</span>
              <span class="text-xs text-primary-500 dark:text-primary-400">{{ mqttForm.wledControl ? 'Enabled' : 'Disabled' }}</span>
            </div>
            <p v-if="mqttForm.wledControl" class="mt-1 ml-4 text-xs text-primary-400 dark:text-primary-500">
              Device: <code class="font-mono">{{ mqttForm.wledDeviceTopic || '—' }}/api</code>
            </p>
            <div v-if="mqttForm.wledControl" class="mt-2 ml-4 flex flex-wrap gap-1.5">
              <span
                v-for="event in mqttEventList"
                :key="event.key"
                :class="[
                  'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs',
                  mqttForm.events[event.key]
                    ? 'border-success-400/60 bg-success-500/10 text-success-700 dark:text-success-300'
                    : 'border-primary-200 bg-transparent text-primary-400 dark:border-primary-700',
                ]"
              >
                <span
                  :class="[
                    'h-1 w-1 rounded-full',
                    mqttForm.events[event.key] ? 'bg-success-500' : 'bg-primary-400',
                  ]"
                ></span>
                {{ event.label }}
              </span>
            </div>
          </div>
        </div>

        <!-- Edit mode -->
        <form v-if="mqttEditing" @submit.prevent="saveMqttSettings" class="mt-4 space-y-6">
          <!-- Broker Connection -->
          <div class="space-y-4">
            <h4 class="text-sm font-medium text-primary-700 dark:text-primary-300">Broker Connection</h4>
            <div>
              <label for="mqtt-broker" class="block text-sm text-primary-600 dark:text-primary-400">Broker URL</label>
              <input
                id="mqtt-broker"
                v-model="mqttForm.broker"
                type="text"
                placeholder="tcp://localhost:1883"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
            </div>
            <div>
              <label for="mqtt-clientid" class="block text-sm text-primary-600 dark:text-primary-400">Client ID</label>
              <input
                id="mqtt-clientid"
                v-model="mqttForm.clientId"
                type="text"
                placeholder="shutterbase"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="mqtt-username" class="block text-sm text-primary-600 dark:text-primary-400">Username</label>
                <input
                  id="mqtt-username"
                  v-model="mqttForm.username"
                  type="text"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
              </div>
              <div>
                <label for="mqtt-password" class="block text-sm text-primary-600 dark:text-primary-400">Password</label>
                <input
                  id="mqtt-password"
                  v-model="mqttForm.password"
                  type="password"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
              </div>
            </div>
            <!-- Test Connection -->
            <div class="border-t border-primary-200 pt-3 dark:border-primary-700">
              <button
                type="button"
                @click="testMqttConnection"
                :disabled="mqttTesting || !mqttForm.broker"
                class="inline-flex items-center gap-1.5 rounded-md border border-primary-300 bg-surface px-3 py-1.5 text-sm font-medium text-primary-700 transition-colors hover:border-primary-400 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-500 dark:hover:text-white cursor-pointer"
              >
                <ArrowPathIcon v-if="mqttTesting" class="h-4 w-4 animate-spin" />
                <SignalIcon v-else class="h-4 w-4" />
                {{ mqttTesting ? 'Testing…' : 'Test Connection' }}
              </button>
              <div v-if="Object.keys(mqttTestResults).length > 0" class="mt-2 space-y-1">
                <div v-for="(result, step) in mqttTestResults" :key="step" class="flex items-center gap-2 text-xs">
                  <CheckCircleIcon v-if="result.ok" class="h-4 w-4 text-success-500" />
                  <XCircleIcon v-else class="h-4 w-4 text-error-500" />
                  <span class="font-medium text-primary-700 dark:text-primary-300">{{ step }}:</span>
                  <span v-if="result.ok" class="text-success-600 dark:text-success-400">OK</span>
                  <span v-else class="text-error-600 dark:text-error-400">{{ result.error }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Section 1: General Events (structured topics) -->
          <div class="space-y-4">
            <div class="flex items-center gap-3">
              <label class="flex items-center gap-2">
                <input
                  type="checkbox"
                  v-model="mqttForm.publishEvents"
                  class="h-4 w-4 rounded border-primary-300 text-accent-500 focus:ring-accent-500"
                />
                <h4 class="text-sm font-medium text-primary-700 dark:text-primary-300">Publish General Events</h4>
              </label>
            </div>
            <p v-if="mqttForm.publishEvents" class="text-xs text-primary-500 dark:text-primary-400">
              Publish structured events to MQTT for any consumer (Home Assistant, Node-RED, custom apps).
            </p>
            <div v-if="mqttForm.publishEvents" class="space-y-4 pl-7">
              <div>
                <label for="mqtt-topicprefix" class="block text-sm text-primary-600 dark:text-primary-400">Topic Prefix</label>
                <input
                  id="mqtt-topicprefix"
                  v-model="mqttForm.topicPrefix"
                  type="text"
                  placeholder="shutterbase"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
                <p class="mt-1 text-xs text-primary-400 dark:text-primary-500">
                  Topics: <code class="font-mono">{{ mqttForm.topicPrefix || 'shutterbase' }}/{{ route.params.id }}/upload/{uploadId}/{event}</code>
                  <br />Events: <span v-for="(ev, i) in mqttEventList" :key="ev.key" class="font-mono">{{ i > 0 ? ', ' : '' }}{{ ev.slug }}</span>
                </p>
              </div>
              <div class="space-y-2">
                <h5 class="text-xs font-medium text-primary-600 dark:text-primary-400">Events</h5>
                <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
                  <label v-for="event in mqttEventList" :key="event.key" class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      v-model="mqttForm.events[event.key]"
                      class="h-3.5 w-3.5 rounded border-primary-300 text-accent-500 focus:ring-accent-500"
                    />
                    <span class="text-xs text-primary-600 dark:text-primary-400">{{ event.label }}</span>
                  </label>
                </div>
              </div>
            </div>
          </div>

          <!-- Section 2: WLED Control (direct device) -->
          <div class="space-y-4">
            <div class="flex items-center gap-3">
              <label class="flex items-center gap-2">
                <input
                  type="checkbox"
                  v-model="mqttForm.wledControl"
                  class="h-4 w-4 rounded border-primary-300 text-accent-500 focus:ring-accent-500"
                />
                <h4 class="text-sm font-medium text-primary-700 dark:text-primary-300">Control WLED Devices</h4>
              </label>
            </div>
            <p v-if="mqttForm.wledControl" class="text-xs text-primary-500 dark:text-primary-400">
              Send commands directly to a WLED device's API topic. Supports presets, effects, and raw JSON.
            </p>
            <div v-if="mqttForm.wledControl" class="space-y-4 pl-7">
              <div>
                <label for="mqtt-wledtopic" class="block text-sm text-primary-600 dark:text-primary-400">WLED Device Topic</label>
                <input
                  id="mqtt-wledtopic"
                  v-model="mqttForm.wledDeviceTopic"
                  type="text"
                  placeholder="wled/device1"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
                <p class="mt-1 text-xs text-primary-400 dark:text-primary-500">
                  Commands are published to <code class="font-mono">{{ mqttForm.wledDeviceTopic || 'wled/device1' }}/api</code>
                </p>
              </div>
              <div class="space-y-4">
                <h5 class="text-xs font-medium text-primary-600 dark:text-primary-400">Events</h5>
                <div v-for="event in mqttEventList" :key="event.key" class="rounded-md border border-primary-200 dark:border-primary-700 p-3 space-y-3">
                  <div class="flex items-center gap-3">
                    <label class="flex items-center gap-2 min-w-[200px]">
                      <input
                        type="checkbox"
                        v-model="mqttForm.events[event.key]"
                        class="h-4 w-4 rounded border-primary-300 text-accent-500 focus:ring-accent-500"
                      />
                      <span class="text-sm font-medium text-primary-700 dark:text-primary-300">{{ event.label }}</span>
                    </label>
                    <div v-if="mqttForm.events[event.key]" class="flex items-center gap-2 ml-auto">
                      <label class="text-xs text-primary-500 dark:text-primary-400">Auto-off:</label>
                      <input
                        v-model.number="mqttForm.durations[event.key]"
                        type="number"
                        min="0"
                        placeholder="0"
                        class="w-16 rounded-md border border-primary-300 bg-white px-2 py-1 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                      />
                      <span class="text-xs text-primary-400 dark:text-primary-500">sec</span>
                    </div>
                  </div>
                  <div v-if="mqttForm.events[event.key]" class="flex flex-wrap items-center gap-3 pl-7">
                    <label class="flex items-center gap-1.5">
                      <input type="radio" :name="'mode-'+event.key" value="preset" v-model="mqttForm.wledCommands[event.key].mode" class="h-3.5 w-3.5 text-accent-500 focus:ring-accent-500" />
                      <span class="text-xs text-primary-600 dark:text-primary-400">Preset</span>
                    </label>
                    <input
                      v-if="mqttForm.wledCommands[event.key].mode === 'preset'"
                      v-model.number="mqttForm.wledCommands[event.key].preset"
                      type="number"
                      min="0"
                      placeholder="#"
                      class="w-20 rounded-md border border-primary-300 bg-white px-2 py-1 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                    />
                    <label class="flex items-center gap-1.5">
                      <input type="radio" :name="'mode-'+event.key" value="effect" v-model="mqttForm.wledCommands[event.key].mode" class="h-3.5 w-3.5 text-accent-500 focus:ring-accent-500" />
                      <span class="text-xs text-primary-600 dark:text-primary-400">Effect</span>
                    </label>
                    <select
                      v-if="mqttForm.wledCommands[event.key].mode === 'effect'"
                      v-model.number="mqttForm.wledCommands[event.key].effect"
                      class="rounded-md border border-primary-300 bg-white px-2 py-1 text-sm text-primary-900 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100"
                    >
                      <option :value="0">0 — Solid</option>
                      <option v-for="fx in WLED_EFFECTS" :key="fx.id" :value="fx.id">{{ fx.id }} — {{ fx.name }}</option>
                    </select>
                    <label class="flex items-center gap-1.5">
                      <input type="radio" :name="'mode-'+event.key" value="raw" v-model="mqttForm.wledCommands[event.key].mode" class="h-3.5 w-3.5 text-accent-500 focus:ring-accent-500" />
                      <span class="text-xs text-primary-600 dark:text-primary-400">Raw JSON</span>
                    </label>
                    <input
                      v-if="mqttForm.wledCommands[event.key].mode === 'raw'"
                      v-model="mqttForm.wledCommands[event.key].raw"
                      type="text"
                      placeholder='{"seg":[{"fx":66}]}'
                      class="flex-1 min-w-[200px] rounded-md border border-primary-300 bg-white px-2 py-1 font-mono text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                    />
                  </div>
                </div>
              </div>
              <!-- Tag Triggers -->
              <div class="space-y-2">
                <h5 class="text-xs font-medium text-primary-600 dark:text-primary-400">Tag Triggers</h5>
                <p class="text-xs text-primary-500 dark:text-primary-400">When "Tag assigned" is enabled above, specify which tag names trigger a WLED command.</p>
                <input
                  v-model="mqttTriggerTagsInput"
                  type="text"
                  placeholder="error, vip, highlight (comma-separated)"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
              </div>
            </div>
          </div>
        </form>
      </div>
    </div>
  </main>
  <ModalMessage
    :show="showRecomputeConfirm"
    :type="MessageType.CONFIRM_WARNING"
    headline="Recompute all images?"
    message="Every image of this project is re-queued for AI detection. Existing AI tags, descriptions and face data are replaced, and the full run costs AI credits."
    confirmText="Recompute all"
    cancelText="Cancel"
    @confirmed="rerunAll"
    @closed="showRecomputeConfirm = false"
  />
  <ModalMessage
    :show="showRerunNumbersConfirm"
    :type="MessageType.CONFIRM_WARNING"
    headline="Re-read car numbers?"
    message="Every image of this project is re-queued for a car-number re-read with the currently configured AI model. Number and scene tags are recomputed; faces, similarity data and descriptions are kept. Cheaper than a full recompute, but the run still costs AI credits."
    confirmText="Re-read numbers"
    cancelText="Cancel"
    @confirmed="rerunNumbers"
    @closed="showRerunNumbersConfirm = false"
  />
  <ModalMessage
    :show="showReclusterConfirm"
    :type="MessageType.CONFIRM_WARNING"
    headline="Recluster faces?"
    message="Person clusters are rebuilt from the existing face data — no AI credits are used. All cluster merges and merge decisions are discarded, and the review queue starts fresh. This affects every project, since face clusters are shared."
    confirmText="Recluster"
    cancelText="Cancel"
    @confirmed="recluster"
    @closed="showReclusterConfirm = false"
  />
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ArrowPathIcon, SignalIcon, CheckCircleIcon, XCircleIcon } from "@heroicons/vue/24/outline";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import DetailEditGroup, { Field, FieldType, EditData } from "src/components/DetailEditGroup.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
import { useUserStore } from "src/stores/user-store";
const route = useRoute();

const userStore = useUserStore();

type ITEM_TYPE = ProjectsResponse;
const ITEM_COLLECTION = "projects";
const ITEM_NAME = "project";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const mqttConfigured = ref(false);
const mqttReachable = ref(false);
const mqttError = ref("");
const savingMqtt = ref(false);
const mqttEditing = ref(false);
const mqttFormBackup = ref<string>("");
const mqttTriggerTagsBackup = ref("");
const mqttTriggerTagsInput = ref("");
const mqttTesting = ref(false);
const mqttTestResults = ref<Record<string, { ok: boolean; error?: string }>>({});

const mqttEventList = [
  { key: "uploadCreated", label: "Upload created", slug: "created" },
  { key: "imageUploaded", label: "Image uploaded", slug: "image-uploaded" },
  { key: "ready", label: "Ready for review", slug: "ready" },
  { key: "approved", label: "Approved", slug: "approved" },
  { key: "rejected", label: "Rejected / sent back", slug: "rejected" },
  { key: "imageRejected", label: "Image rejected (tag)", slug: "image-rejected" },
  { key: "tagAssigned", label: "Tag assigned", slug: "tag-assigned" },
];

const WLED_EFFECTS = [
  { id: 0, name: "Solid" },
  { id: 1, name: "Blink" },
  { id: 2, name: "Breathe" },
  { id: 3, name: "Wipe" },
  { id: 4, name: "Wipe Random" },
  { id: 5, name: "Random Colors" },
  { id: 6, name: "Sweep" },
  { id: 7, name: "Dynamic" },
  { id: 8, name: "Rainbow" },
  { id: 9, name: "Rainbow Cycle" },
  { id: 10, name: "Theater Chase" },
  { id: 11, name: "Theater Chase Rainbow" },
  { id: 12, name: "Running Lights" },
  { id: 13, name: "Twinkle" },
  { id: 14, name: "Twinkle Rainbow" },
  { id: 15, name: "Twinklefox" },
  { id: 16, name: "Christmas" },
  { id: 17, name: "Halloween" },
  { id: 18, name: "Fire Flicker" },
  { id: 19, name: "Fire Flicker (soft)" },
  { id: 20, name: "Fire Flicker (intense)" },
  { id: 21, name: "Candle" },
  { id: 22, name: "Fireworks" },
  { id: 23, name: "Fireworks Random" },
  { id: 24, name: "Fireworks Starburst" },
  { id: 25, name: "Halloween Eyes" },
  { id: 26, name: "Party" },
  { id: 27, name: "Android" },
  { id: 28, name: "Strobe" },
  { id: 29, name: "Strobe Rainbow" },
  { id: 30, name: "Mega Strobe" },
  { id: 31, name: "Rainbow Swirl" },
  { id: 32, name: "Rainbow Swirl (fast)" },
  { id: 33, name: "Chase" },
  { id: 34, name: "Chase Rainbow" },
  { id: 35, name: "Chase Flash" },
  { id: 36, name: "Chase Flash Random" },
  { id: 37, name: "Chase Rainbow White" },
  { id: 38, name: "Chase Blob" },
  { id: 39, name: "Lightning" },
  { id: 40, name: "Matrix" },
  { id: 41, name: "Meteor" },
  { id: 42, name: "Meteor Smooth" },
  { id: 43, name: "Railway" },
  { id: 44, name: "Ripple" },
  { id: 45, name: "Ripple Rainbow" },
  { id: 46, name: "Fire 2012" },
  { id: 47, name: "Fire 2012 (64)" },
  { id: 48, name: "Fire 2012 (extended)" },
  { id: 49, name: "Colorwaves" },
  { id: 50, name: "BPM" },
  { id: 51, name: "Fill Noise" },
  { id: 52, name: "Noise 1" },
  { id: 53, name: "Noise 2" },
  { id: 54, name: "Noise 3" },
  { id: 55, name: "Noise 4" },
  { id: 56, name: "Colortwinkles" },
  { id: 57, name: "Lake" },
  { id: 58, name: "Meteor Rainbow" },
  { id: 59, name: "Meteor Storm" },
  { id: 60, name: "Sinelon" },
  { id: 61, name: "Sinelon Dual" },
  { id: 62, name: "Sinelon Rainbow" },
  { id: 63, name: "Popcorn" },
  { id: 64, name: "Drip" },
  { id: 65, name: "Plasma" },
  { id: 66, name: "Percent" },
  { id: 67, name: "Ripple Rainbow" },
  { id: 68, name: "Heartbeat" },
  { id: 69, name: "Pacifica" },
  { id: 70, name: "Candle Multi" },
  { id: 71, name: "Solid Glitter" },
  { id: 72, name: "Sunrise" },
  { id: 73, name: "Phased" },
  { id: 74, name: "TwinkleUp" },
  { id: 75, name: "Noise Pal" },
  { id: 76, name: "Sine" },
  { id: 77, name: "Phased Noise" },
  { id: 78, name: "Flow" },
  { id: 79, name: "Chunchun" },
  { id: 80, name: "Dancing Shadows" },
  { id: 81, name: "Washing Machine" },
  { id: 82, name: "Candy Cane" },
  { id: 83, name: "Blends" },
  { id: 84, name: "TV Simulator" },
  { id: 85, name: "Dynamic Smooth" },
  { id: 86, name: "Spaceships" },
  { id: 87, name: "Aeroplane" },
  { id: 88, name: "Snowball" },
  { id: 89, name: "Fireworks 1D" },
  { id: 90, name: "Bouncing Balls" },
  { id: 91, name: "Solid Pattern" },
  { id: 92, name: "Solid Pattern Tri" },
  { id: 93, name: "Spots" },
  { id: 94, name: "Spots Fade" },
  { id: 95, name: "Glitter" },
  { id: 96, name: "Candle" },
  { id: 97, name: "Fireworks Starburst" },
  { id: 98, name: "Fireworks 1D" },
  { id: 99, name: "Bouncing Balls" },
  { id: 100, name: "Sinelon" },
  { id: 101, name: "Sinelon Dual" },
  { id: 102, name: "Sinelon Rainbow" },
  { id: 103, name: "Popcorn" },
  { id: 104, name: "Drip" },
  { id: 105, name: "Plasma" },
  { id: 106, name: "Percent" },
  { id: 107, name: "Ripple Rainbow" },
  { id: 108, name: "Heartbeat" },
  { id: 109, name: "Pacifica" },
  { id: 110, name: "Candle Multi" },
  { id: 111, name: "Solid Glitter" },
  { id: 112, name: "Sunrise" },
  { id: 113, name: "Phased" },
  { id: 114, name: "TwinkleUp" },
  { id: 115, name: "Noise Pal" },
  { id: 116, name: "Sine" },
  { id: 117, name: "Phased Noise" },
];

type WledMode = "preset" | "effect" | "raw";

interface WledCommandForm {
  mode: WledMode;
  preset: number;
  effect: number;
  raw: string;
}

const mqttForm = ref({
  broker: "",
  clientId: "",
  username: "",
  password: "",
  topicPrefix: "",
  wledDeviceTopic: "",
  publishEvents: false,
  wledControl: false,
  events: {
    uploadCreated: false,
    imageUploaded: false,
    ready: false,
    approved: false,
    rejected: false,
    imageRejected: false,
    tagAssigned: false,
  },
  wledCommands: {
    uploadCreated: { mode: "effect", preset: 0, effect: 3, raw: "" },   // Wipe
    imageUploaded: { mode: "effect", preset: 0, effect: 0, raw: "" },   // Solid (off)
    ready:         { mode: "effect", preset: 0, effect: 8, raw: "" },   // Rainbow
    approved:      { mode: "effect", preset: 0, effect: 2, raw: "" },   // Breathe
    rejected:      { mode: "effect", preset: 0, effect: 1, raw: "" },   // Blink
    imageRejected: { mode: "effect", preset: 0, effect: 1, raw: "" },   // Blink
    tagAssigned:   { mode: "effect", preset: 0, effect: 0, raw: "" },   // Solid (off)
  } as Record<string, WledCommandForm>,
  durations: {
    uploadCreated: 3,
    imageUploaded: 0,
    ready: 5,
    approved: 3,
    rejected: 5,
    imageRejected: 2,
    tagAssigned: 0,
  },
});

function makeDefaultWledCommand(): WledCommandForm {
  return { mode: "effect", preset: 0, effect: 0, raw: "" };
}

function wledCommandToForm(cmd: { preset?: number; effect?: number; raw?: string }): WledCommandForm {
  if (cmd.raw) return { mode: "raw", preset: 0, effect: 0, raw: cmd.raw };
  if (cmd.effect !== undefined && cmd.effect !== null) return { mode: "effect", preset: 0, effect: cmd.effect, raw: "" };
  return { mode: "preset", preset: cmd.preset ?? 0, effect: 0, raw: "" };
}

async function loadItem() {
  const itemId: string = `${route.params.id}`;
  if (!itemId || itemId === "") {
    console.log(`No ${ITEM_NAME} ID provided`);
    return;
  }

  try {
    console.log(`Loading ${ITEM_NAME} ${itemId}`);
    const response = await api.projects.get(itemId);
    item.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function saveItem(editData: EditData<ITEM_TYPE>) {
  if (!item.value) {
    console.log(`No ${ITEM_NAME} to save`);
    return;
  }

  const rollbackData = { ...item.value };
  item.value = { ...item.value, ...editData };

  try {
    console.log(`Saving ${ITEM_NAME} ${item.value.id}`);
    const response = await api.projects.update(item.value.id, editData as Partial<ITEM_TYPE>);
    item.value = response;
    showNotificationToast({ headline: `${capitalize(ITEM_NAME)} saved`, type: "success" });
  } catch (error: any) {
    item.value = rollbackData;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const rerunningFailed = ref(false);
async function rerunFailed() {
  if (!item.value) return;
  rerunningFailed.value = true;
  try {
    const queued = await api.ai.rerunFailed(item.value.id);
    showNotificationToast({
      headline: queued === 0 ? "No failed images to re-queue" : `AI detection re-queued for ${queued} image${queued === 1 ? "" : "s"}`,
      type: "success",
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    rerunningFailed.value = false;
  }
}

const rerunningAll = ref(false);
const showRecomputeConfirm = ref(false);
async function rerunAll() {
  if (!item.value) return;
  showRecomputeConfirm.value = false;
  rerunningAll.value = true;
  try {
    const queued = await api.ai.rerunAll(item.value.id);
    showNotificationToast({
      headline: queued === 0 ? "No images to recompute" : `AI detection re-queued for ${queued} image${queued === 1 ? "" : "s"}`,
      type: "success",
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    rerunningAll.value = false;
  }
}

const rerunningNumbers = ref(false);
const showRerunNumbersConfirm = ref(false);
async function rerunNumbers() {
  if (!item.value) return;
  showRerunNumbersConfirm.value = false;
  rerunningNumbers.value = true;
  try {
    const queued = await api.ai.rerunNumbers(item.value.id);
    showNotificationToast({
      headline: queued === 0 ? "No images to re-read" : `Car-number re-read queued for ${queued} image${queued === 1 ? "" : "s"}`,
      type: "success",
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    rerunningNumbers.value = false;
  }
}

const showReclusterConfirm = ref(false);
async function recluster() {
  if (!item.value) return;
  showReclusterConfirm.value = false;
  try {
    await api.ai.recluster(item.value.id);
    showNotificationToast({ headline: "Recluster started — clusters repopulate on the Faces page", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const informationFields: Field<ITEM_TYPE>[] = [
  { key: "name", label: "Name", type: FieldType.TEXT },
  { key: "description", label: "Description", type: FieldType.TEXT },
];

const aiFields: Field<ITEM_TYPE>[] = [{ key: "aiSystemMessage", label: "System Message", type: FieldType.TEXT }];

const periodFields: Field<ITEM_TYPE>[] = [
  { key: "startAt", label: "Starts", type: FieldType.DATETIME },
  { key: "endAt", label: "Ends", type: FieldType.DATETIME },
];

const reviewFields: Field<ITEM_TYPE>[] = [{ key: "uploadReviewEnabled", label: "Upload reviews", type: FieldType.BOOLEAN, hint: "Enable the open / ready / reviewed flow" }];

const copyrightFields: Field<ITEM_TYPE>[] = [
  { key: "copyright", label: "Copyright", type: FieldType.TEXT },
  { key: "copyrightReference", label: "Copyright reference", type: FieldType.TEXT },
  { key: "copyrightTagPrefix", label: "Copyright tag prefix", type: FieldType.TEXT, hint: "Prepended to the photographer's copyright tag in exported EXIF only, e.g. by_" },
  { key: "locationName", label: "Location name", type: FieldType.TEXT },
  { key: "locationCode", label: "Location code", type: FieldType.TEXT },
  { key: "locationCity", label: "Location city", type: FieldType.TEXT },
];

watch(route, loadItem);
onMounted(() => {
  loadItem();
  loadMqttSettings();
});

async function loadMqttSettings() {
  const projectId = `${route.params.id}`;
  if (!projectId) return;
  try {
    const [settings, status] = await Promise.all([
      api.adminSettings.getProjectMqttSettings(projectId),
      api.mqtt.getProjectMqttStatus(projectId),
    ]);
    mqttForm.value.broker = settings.broker;
    mqttForm.value.clientId = settings.clientId;
    mqttForm.value.username = settings.username;
    mqttForm.value.password = settings.password;
    mqttForm.value.topicPrefix = settings.topicPrefix;
    mqttForm.value.wledDeviceTopic = settings.wledDeviceTopic;
    mqttForm.value.publishEvents = settings.publishEvents;
    mqttForm.value.wledControl = settings.wledControl;
    mqttForm.value.events = settings.events;
    mqttForm.value.durations = settings.durations;
    for (const event of mqttEventList) {
      const cmd = settings.wledCommands?.[event.key as keyof typeof settings.wledCommands];
      mqttForm.value.wledCommands[event.key] = cmd ? wledCommandToForm(cmd) : makeDefaultWledCommand();
    }
    mqttTriggerTagsInput.value = settings.triggerTags?.join(", ") || "";
    mqttConfigured.value = status.configured;
    mqttReachable.value = status.reachable;
    mqttError.value = status.error || "";
  } catch {
    // MQTT not configured for this project
  }
}

function startMqttEdit() {
  mqttFormBackup.value = JSON.stringify(mqttForm.value);
  mqttTriggerTagsBackup.value = mqttTriggerTagsInput.value;
  mqttTestResults.value = {};
  mqttEditing.value = true;
}

function cancelMqttEdit() {
  if (mqttFormBackup.value) {
    Object.assign(mqttForm.value, JSON.parse(mqttFormBackup.value));
  }
  mqttTriggerTagsInput.value = mqttTriggerTagsBackup.value;
  mqttEditing.value = false;
}

async function saveMqttSettings() {
  const projectId = `${route.params.id}`;
  if (!projectId) return;
  savingMqtt.value = true;
  try {
    const triggerTags = mqttTriggerTagsInput.value
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t.length > 0);

    const wledCommands: Record<string, { preset?: number; effect?: number; raw?: string }> = {};
    for (const event of mqttEventList) {
      const cmd = mqttForm.value.wledCommands[event.key];
      if (!cmd) continue;
      if (cmd.mode === "raw" && cmd.raw) {
        wledCommands[event.key] = { raw: cmd.raw };
      } else if (cmd.mode === "effect") {
        wledCommands[event.key] = { effect: cmd.effect };
      } else {
        wledCommands[event.key] = { preset: cmd.preset };
      }
    }

    await api.adminSettings.updateProjectMqttSettings(projectId, {
      broker: mqttForm.value.broker,
      clientId: mqttForm.value.clientId,
      username: mqttForm.value.username,
      password: mqttForm.value.password,
      topicPrefix: mqttForm.value.topicPrefix,
      wledDeviceTopic: mqttForm.value.wledDeviceTopic,
      publishEvents: mqttForm.value.publishEvents,
      wledControl: mqttForm.value.wledControl,
      events: mqttForm.value.events,
      wledCommands: wledCommands as any,
      durations: mqttForm.value.durations,
      triggerTags,
    });
    const status = await api.mqtt.getProjectMqttStatus(projectId);
    mqttConfigured.value = status.configured;
    mqttReachable.value = status.reachable;
    mqttError.value = status.error || "";
    mqttEditing.value = false;
    // Re-sync form from backend to reflect any server-side normalization
    await loadMqttSettings();
    showNotificationToast({ headline: "MQTT settings saved", type: "success" });
  } catch (e: any) {
    unexpectedError.value = e;
    showUnexpectedErrorMessage.value = true;
  } finally {
    savingMqtt.value = false;
  }
}

async function testMqttConnection() {
  const projectId = `${route.params.id}`;
  if (!projectId) return;
  mqttTesting.value = true;
  mqttTestResults.value = {};
  try {
    const results = await api.mqtt.testProjectMqtt(projectId, {
      broker: mqttForm.value.broker,
      clientId: mqttForm.value.clientId,
      username: mqttForm.value.username,
      password: mqttForm.value.password,
      topicPrefix: mqttForm.value.topicPrefix,
    });
    mqttTestResults.value = results;
    // Refresh status badge after test
    const status = await api.mqtt.getProjectMqttStatus(projectId);
    mqttConfigured.value = status.configured;
    mqttReachable.value = status.reachable;
    mqttError.value = status.error || "";
  } catch (e: any) {
    unexpectedError.value = e;
    showUnexpectedErrorMessage.value = true;
  } finally {
    mqttTesting.value = false;
  }
}
</script>
