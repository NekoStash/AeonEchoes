<script setup lang="ts">
import { Bot, Check, ChevronDown, History, Lightbulb, Play, Plus, Replace, RotateCcw, Square, X } from '@lucide/vue'
import { computed, ref } from 'vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiButton from '~/components/ui/Button.vue'
import UiEmptyState from '~/components/ui/EmptyState.vue'
import UiInlineNotice from '~/components/ui/InlineNotice.vue'
import UiInput from '~/components/ui/Input.vue'
import UiSelect from '~/components/ui/Select.vue'
import UiSwitch from '~/components/ui/Switch.vue'
import UiTextarea from '~/components/ui/Textarea.vue'
import type { AgentConfig, AgentRunStreamState, AgentRunStreamTool } from '~/entities/agent'
import type { Chapter, ChapterVersion } from '~/entities/chapter'
import { canCancelAgentRun, type AgentProposal } from '~/features/agent-run'
import type { ContextSelectState } from '~/features/context-select'
import type { EditorDraftSnapshot } from '~/features/editor-draft-recovery'
import type { ModelResolution, StoryBible, ToolTrace } from '~/lib/types'

const props = withDefaults(defineProps<{
  agents: AgentConfig[]
  projectId: string
  agentLoadError: string
  chapters: Chapter[]
  chapter: Chapter
  bible: StoryBible | null
  versions: ChapterVersion[]
  proposals: AgentProposal[]
  prompt: string
  selectedAgentId: string
  contextState: ContextSelectState
  selectedText: string
  localDraft: EditorDraftSnapshot | null
  loadingAgents: boolean
  runningAgent: boolean
  streamState: AgentRunStreamState
  loadingVersions: boolean
  diagnostics: { modelResolution: ModelResolution | null; toolTrace: ToolTrace[] }
  planningChapter?: boolean
  isEmptyChapter?: boolean
}>(), {
  planningChapter: false,
  isEmptyChapter: false
})

const emit = defineEmits<{
  'update:prompt': [value: string]
  'update:selectedAgentId': [value: string]
  'update:contextState': [value: ContextSelectState]
  retryAgents: []
  run: []
  planChapter: []
  cancelRun: []
  insert: [proposalId: string]
  replace: [proposalId: string]
  append: [proposalId: string]
  overwrite: [proposalId: string]
  reject: [proposalId: string]
  restoreDraft: []
  keepBackend: []
  viewDraftDiff: []
  loadVersion: [versionId: string]
}>()

const { t } = useI18n()
const diagnosticsOpen = ref(false)
/** Collapsed by default: stream tool list + each tool I/O + proposal tool list. */
const streamToolsOpen = ref(false)
const openToolKeys = ref<string[]>([])
const openProposalToolLists = ref<string[]>([])
const agentOptions = computed(() => props.agents.map((agent) => ({
  label: agent.name,
  value: agent.id,
  description: `${agent.project_id === props.projectId ? t('editor.assistant.scopeProject') : t('editor.assistant.scopeGlobal')} · ${agent.role ? t(`agents.roles.${agent.role.replaceAll('-', '_')}`) : t('agents.roles.default')}`
})))
const currentChapterIndex = computed(() => props.chapters.findIndex((chapter) => chapter.id === props.chapter.id))
const previousLimit = computed(() => Math.max(0, currentChapterIndex.value))
const syncedCharacters = computed(() => (props.bible?.characters || []).filter((character) => character.entity_id?.trim()))
const showStreamingCard = computed(() => props.streamState.status !== 'idle'
  && !props.proposals.some((proposal) => proposal.runId === props.streamState.runId))
const showStreamWaiting = computed(() => !props.streamState.content && props.runningAgent)
const streamCharCount = computed(() => Array.from(props.streamState.content || '').length)
const showStreamCharCount = computed(() => props.streamState.status !== 'idle'
  && props.streamState.status !== 'completed'
  && (streamCharCount.value > 0 || props.runningAgent))
const canCancelRun = computed(() => canCancelAgentRun(props.streamState.status))
const streamTone = computed(() => {
  if (props.streamState.status === 'failed') return 'danger' as const
  if (props.streamState.status === 'cancelled') return 'muted' as const
  if (props.streamState.status === 'completed') return 'success' as const
  return 'info' as const
})
const canPlanChapter = computed(() => !props.runningAgent && !props.planningChapter)

function toolLabel(tool: AgentRunStreamTool) {
  return tool.name.trim() || t('editor.stream.unknownTool')
}

function toolStatus(tool: AgentRunStreamTool) {
  return t(`editor.stream.toolStatus.${tool.status}`)
}

function toolKey(prefix: string, tool: AgentRunStreamTool, index: number) {
  return `${prefix}:${tool.call_id || tool.name || 'tool'}:${index}`
}

function isToolOpen(key: string) {
  return openToolKeys.value.includes(key)
}

function toggleToolOpen(key: string) {
  if (isToolOpen(key)) openToolKeys.value = openToolKeys.value.filter((item) => item !== key)
  else openToolKeys.value = [...openToolKeys.value, key]
}

function isProposalToolsOpen(proposalId: string) {
  return openProposalToolLists.value.includes(proposalId)
}

function toggleProposalToolsOpen(proposalId: string) {
  if (isProposalToolsOpen(proposalId)) {
    openProposalToolLists.value = openProposalToolLists.value.filter((item) => item !== proposalId)
  } else {
    openProposalToolLists.value = [...openProposalToolLists.value, proposalId]
  }
}

function formatToolPayload(value: unknown) {
  if (value === undefined || value === null) return ''
  try {
    return JSON.stringify(value, null, 2)
  } catch (cause) {
    console.error('[AeonEchoes AssistantPanel] Failed to format tool payload.', cause)
    return String(value)
  }
}

function toolHasDetails(tool: AgentRunStreamTool) {
  return Boolean(tool.arguments || tool.result)
}

function patchContext(patch: Partial<ContextSelectState>) {
  emit('update:contextState', { ...props.contextState, ...patch })
}

function toggleCharacter(characterId: string) {
  const selected = new Set(props.contextState.characterIds)
  if (selected.has(characterId)) selected.delete(characterId)
  else selected.add(characterId)
  patchContext({ characterIds: Array.from(selected) })
}

function versionLabel(version: ChapterVersion) {
  return t('editor.versionLabel', { version: version.version, title: version.title })
}
</script>

<template>
  <div data-testid="assistant-panel" class="space-y-5">
    <UiInlineNotice
      v-if="localDraft"
      tone="warning"
      :title="t('editor.recovery.title')"
      :description="t('editor.recovery.description', { time: new Date(localDraft.updated_at).toLocaleString() })"
    >
      <template #actions>
        <div class="flex flex-wrap gap-2">
          <UiButton size="sm" variant="outline" @click="emit('viewDraftDiff')">{{ t('editor.recovery.viewDiff') }}</UiButton>
          <UiButton size="sm" variant="outline" @click="emit('keepBackend')">{{ t('editor.recovery.keepBackend') }}</UiButton>
          <UiButton size="sm" @click="emit('restoreDraft')"><RotateCcw class="h-4 w-4" />{{ t('editor.recovery.restore') }}</UiButton>
        </div>
      </template>
    </UiInlineNotice>

    <section class="border border-current/15 bg-background/45 p-4">
      <div class="flex items-center gap-3">
        <Bot class="h-5 w-5" />
        <div>
          <h3 class="font-semibold">{{ t('editor.assistant.title') }}</h3>
          <p class="text-xs leading-5 text-muted-foreground">{{ t('editor.assistant.description') }}</p>
        </div>
      </div>
      <div class="mt-4 space-y-3">
        <UiInlineNotice
          v-if="agentLoadError"
          tone="danger"
          :title="t('editor.assistant.loadFailedTitle')"
          :description="agentLoadError"
        >
          <template #actions>
            <UiButton size="sm" variant="outline" @click="emit('retryAgents')">{{ t('common.retry') }}</UiButton>
          </template>
        </UiInlineNotice>
        <UiEmptyState
          v-else-if="!loadingAgents && agents.length === 0"
          class="min-h-32"
          :title="t('editor.assistant.emptyAgentsTitle')"
          :description="t('editor.assistant.emptyAgentsDescription')"
        >
          <template #actions>
            <UiButton size="sm" variant="outline" @click="navigateTo('/settings/agents')">{{ t('editor.assistant.openAgentSettings') }}</UiButton>
          </template>
        </UiEmptyState>
        <UiSelect
          :model-value="selectedAgentId"
          :options="agentOptions"
          :disabled="loadingAgents"
          :placeholder="t('editor.assistant.agentPlaceholder')"
          :empty-text="t('editor.assistant.emptyAgentOptions')"
          :aria-label="t('editor.assistant.agentLabel')"
          @update:model-value="emit('update:selectedAgentId', $event)"
        />
        <UiTextarea
          :model-value="prompt"
          :rows="6"
          :placeholder="t('editor.assistant.promptPlaceholder')"
          @update:model-value="emit('update:prompt', $event)"
        />
        <UiInlineNotice
          v-if="isEmptyChapter"
          tone="info"
          :title="t('editor.assistant.emptyChapterTitle')"
          :description="t('editor.assistant.emptyChapterDescription')"
        />
        <UiButton
          class="w-full"
          :variant="isEmptyChapter ? 'default' : 'outline'"
          :loading="planningChapter"
          :disabled="!canPlanChapter"
          data-testid="plan-chapter-button"
          @click="emit('planChapter')"
        >
          <Lightbulb class="h-4 w-4" />
          {{ planningChapter ? t('editor.actions.planningChapter') : t('editor.actions.planChapter') }}
        </UiButton>
        <div class="grid gap-2" :class="canCancelRun ? 'grid-cols-[1fr_auto]' : 'grid-cols-1'">
          <UiButton class="w-full" :loading="runningAgent" :disabled="runningAgent || planningChapter || !selectedAgentId || !prompt.trim()" @click="emit('run')">
            <Play class="h-4 w-4" />
            {{ t('editor.actions.runAgent') }}
          </UiButton>
          <UiButton v-if="canCancelRun" variant="outline" :aria-label="t('editor.stream.cancel')" @click="emit('cancelRun')">
            <Square class="h-4 w-4" />
            {{ t('editor.stream.cancel') }}
          </UiButton>
        </div>
      </div>
    </section>

    <section class="border border-current/15 bg-background/45 p-4">
      <h3 class="font-semibold">{{ t('editor.context.title') }}</h3>
      <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ t('editor.context.description') }}</p>
      <div class="mt-4 space-y-3">
        <UiSwitch
          :model-value="contextState.includeCurrentChapter"
          :label="t('editor.context.currentChapter')"
          @update:model-value="patchContext({ includeCurrentChapter: $event })"
        />
        <UiSwitch
          :model-value="contextState.includeWorldRules"
          :label="t('editor.context.worldRules')"
          @update:model-value="patchContext({ includeWorldRules: $event })"
        />
        <label class="block space-y-2">
          <span class="text-sm font-semibold">{{ t('editor.context.previousChapters') }}</span>
          <UiInput
            :model-value="contextState.previousChapterCount"
            type="number"
            min="0"
            :max="previousLimit"
            @update:model-value="patchContext({ previousChapterCount: Math.min(previousLimit, Math.max(0, Number($event))) })"
          />
        </label>
        <div>
          <p class="text-sm font-semibold">{{ t('editor.context.characters') }}</p>
          <p v-if="syncedCharacters.length === 0" class="mt-2 text-xs text-muted-foreground">{{ t('editor.context.noSyncedCharacters') }}</p>
          <div v-else class="mt-2 flex flex-wrap gap-2">
            <button
              v-for="character in syncedCharacters"
              :key="character.id"
              type="button"
              :class="[
                'focus-ring border px-3 py-2 text-left text-xs transition-colors',
                contextState.characterIds.includes(character.id) ? 'border-foreground bg-foreground text-background' : 'border-border bg-background text-muted-foreground hover:text-foreground'
              ]"
              @click="toggleCharacter(character.id)"
            >
              {{ character.name }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <section class="space-y-3">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h3 class="font-semibold">{{ t('editor.proposals.title') }}</h3>
          <p class="text-xs leading-5 text-muted-foreground">{{ t('editor.proposals.description') }}</p>
        </div>
        <UiBadge tone="muted">{{ proposals.filter((proposal) => proposal.status === 'pending').length }}</UiBadge>
      </div>
      <UiEmptyState v-if="proposals.length === 0 && !showStreamingCard" class="min-h-32" :title="t('editor.proposals.emptyTitle')" :description="t('editor.proposals.emptyDescription')" />
      <article
        v-if="showStreamingCard"
        data-testid="agent-stream-card"
        class="border border-current/15 bg-background/45 p-4"
      >
        <div class="flex items-center justify-between gap-3">
          <div role="status" aria-live="polite" aria-atomic="true">
            <UiBadge :tone="streamTone">{{ t(`editor.stream.status.${streamState.status}`) }}</UiBadge>
          </div>
          <span v-if="streamState.runId" class="max-w-36 truncate text-[11px] text-muted-foreground">{{ streamState.runId }}</span>
        </div>
        <p
          v-if="showStreamCharCount"
          data-testid="agent-stream-char-count"
          class="mt-2 text-xs text-muted-foreground"
          aria-live="polite"
        >
          {{ t('editor.stream.charsOutput', { count: streamCharCount }) }}
        </p>
        <pre
          v-if="streamState.content"
          data-testid="agent-stream-content"
          class="mt-3 max-h-72 overflow-auto whitespace-pre-wrap font-serif text-sm leading-7 subtle-scrollbar"
        >{{ streamState.content }}</pre>
        <p v-else-if="showStreamWaiting" class="mt-3 text-sm text-muted-foreground">{{ t('editor.stream.waiting') }}</p>
        <div v-if="streamState.tools.length" data-testid="agent-stream-tools" class="mt-3 border-t border-current/10 pt-3">
          <button
            type="button"
            class="focus-ring flex w-full items-center justify-between gap-3 py-1 text-left text-xs font-semibold"
            :aria-expanded="streamToolsOpen"
            data-testid="agent-stream-tools-toggle"
            @click="streamToolsOpen = !streamToolsOpen"
          >
            <span>{{ t('editor.stream.tools') }} · {{ streamState.tools.length }}</span>
            <ChevronDown :class="['h-4 w-4 shrink-0 transition-transform', streamToolsOpen && 'rotate-180']" aria-hidden="true" />
          </button>
          <div v-if="streamToolsOpen" class="mt-2 space-y-2">
            <div
              v-for="(tool, index) in streamState.tools"
              :key="toolKey('stream', tool, index)"
              class="border border-current/10 bg-background/30"
              data-testid="agent-stream-tool-item"
            >
              <button
                type="button"
                class="focus-ring flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-xs"
                :aria-expanded="isToolOpen(toolKey('stream', tool, index))"
                :disabled="!toolHasDetails(tool)"
                @click="toolHasDetails(tool) && toggleToolOpen(toolKey('stream', tool, index))"
              >
                <span class="min-w-0 truncate font-medium text-foreground">{{ toolLabel(tool) }}</span>
                <span class="flex shrink-0 items-center gap-2">
                  <UiBadge tone="muted">{{ toolStatus(tool) || t('editor.stream.toolRunning') }}</UiBadge>
                  <ChevronDown
                    v-if="toolHasDetails(tool)"
                    :class="['h-3.5 w-3.5 transition-transform', isToolOpen(toolKey('stream', tool, index)) && 'rotate-180']"
                    aria-hidden="true"
                  />
                </span>
              </button>
              <div v-if="isToolOpen(toolKey('stream', tool, index)) && toolHasDetails(tool)" class="space-y-2 border-t border-current/10 px-3 py-2">
                <div v-if="tool.arguments">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">{{ t('editor.stream.toolInput') }}</p>
                  <pre class="mt-1 max-h-40 overflow-auto whitespace-pre-wrap font-mono text-[11px] leading-5 subtle-scrollbar">{{ formatToolPayload(tool.arguments) }}</pre>
                </div>
                <div v-if="tool.result">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">{{ t('editor.stream.toolOutput') }}</p>
                  <pre class="mt-1 max-h-40 overflow-auto whitespace-pre-wrap font-mono text-[11px] leading-5 subtle-scrollbar">{{ formatToolPayload(tool.result) }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
        <UiInlineNotice
          v-if="streamState.error"
          class="mt-3"
          :tone="streamState.status === 'failed' ? 'danger' : 'warning'"
          :title="streamState.status === 'cancelled' ? t('editor.stream.cancelledTitle') : t('editor.stream.failedTitle')"
          :description="streamState.error"
        />
      </article>
      <article v-for="proposal in proposals" :key="proposal.id" class="border border-current/15 bg-background/45 p-4" data-testid="agent-proposal-card">
        <div class="flex items-center justify-between gap-3">
          <UiBadge :tone="proposal.status === 'pending' ? 'info' : proposal.status === 'applied' ? 'success' : 'muted'">
            {{ t(`editor.proposals.status.${proposal.status}`) }}
          </UiBadge>
          <span class="text-[11px] text-muted-foreground">{{ new Date(proposal.createdAt).toLocaleString() }}</span>
        </div>
        <pre class="mt-3 max-h-72 overflow-auto whitespace-pre-wrap font-serif text-sm leading-7 subtle-scrollbar">{{ proposal.content }}</pre>
        <div v-if="proposal.tools?.length" class="mt-3 border-t border-current/10 pt-3" data-testid="agent-proposal-tools">
          <button
            type="button"
            class="focus-ring flex w-full items-center justify-between gap-3 py-1 text-left text-xs font-semibold"
            :aria-expanded="isProposalToolsOpen(proposal.id)"
            data-testid="agent-proposal-tools-toggle"
            @click="toggleProposalToolsOpen(proposal.id)"
          >
            <span>{{ t('editor.proposals.tools') }} · {{ proposal.tools.length }}</span>
            <ChevronDown :class="['h-4 w-4 shrink-0 transition-transform', isProposalToolsOpen(proposal.id) && 'rotate-180']" aria-hidden="true" />
          </button>
          <div v-if="isProposalToolsOpen(proposal.id)" class="mt-2 space-y-2">
            <div
              v-for="(tool, index) in proposal.tools"
              :key="toolKey(proposal.id, tool, index)"
              class="border border-current/10 bg-background/30"
              data-testid="agent-proposal-tool-item"
            >
              <button
                type="button"
                class="focus-ring flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-xs"
                :aria-expanded="isToolOpen(toolKey(proposal.id, tool, index))"
                :disabled="!toolHasDetails(tool)"
                @click="toolHasDetails(tool) && toggleToolOpen(toolKey(proposal.id, tool, index))"
              >
                <span class="min-w-0 truncate font-medium text-foreground">{{ toolLabel(tool) }}</span>
                <span class="flex shrink-0 items-center gap-2">
                  <UiBadge tone="muted">{{ toolStatus(tool) || t('editor.stream.toolRunning') }}</UiBadge>
                  <ChevronDown
                    v-if="toolHasDetails(tool)"
                    :class="['h-3.5 w-3.5 transition-transform', isToolOpen(toolKey(proposal.id, tool, index)) && 'rotate-180']"
                    aria-hidden="true"
                  />
                </span>
              </button>
              <div v-if="isToolOpen(toolKey(proposal.id, tool, index)) && toolHasDetails(tool)" class="space-y-2 border-t border-current/10 px-3 py-2">
                <div v-if="tool.arguments">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">{{ t('editor.stream.toolInput') }}</p>
                  <pre class="mt-1 max-h-40 overflow-auto whitespace-pre-wrap font-mono text-[11px] leading-5 subtle-scrollbar">{{ formatToolPayload(tool.arguments) }}</pre>
                </div>
                <div v-if="tool.result">
                  <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">{{ t('editor.stream.toolOutput') }}</p>
                  <pre class="mt-1 max-h-40 overflow-auto whitespace-pre-wrap font-mono text-[11px] leading-5 subtle-scrollbar">{{ formatToolPayload(tool.result) }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="proposal.status === 'pending'" class="mt-4 grid grid-cols-2 gap-2">
          <UiButton size="sm" variant="outline" @click="emit('insert', proposal.id)"><Plus class="h-4 w-4" />{{ t('editor.proposals.insert') }}</UiButton>
          <UiButton size="sm" variant="outline" :disabled="!selectedText" @click="emit('replace', proposal.id)"><Replace class="h-4 w-4" />{{ t('editor.proposals.replace') }}</UiButton>
          <UiButton size="sm" variant="outline" @click="emit('append', proposal.id)"><Check class="h-4 w-4" />{{ t('editor.proposals.append') }}</UiButton>
          <UiButton size="sm" variant="outline" @click="emit('overwrite', proposal.id)"><Replace class="h-4 w-4" />{{ t('editor.proposals.overwrite') }}</UiButton>
          <UiButton class="col-span-2" size="sm" variant="ghost" @click="emit('reject', proposal.id)"><X class="h-4 w-4" />{{ t('editor.proposals.reject') }}</UiButton>
        </div>
      </article>
    </section>

    <section class="space-y-3">
      <div class="flex items-center gap-2">
        <History class="h-4 w-4" />
        <h3 class="font-semibold">{{ t('editor.versions') }}</h3>
      </div>
      <p v-if="loadingVersions" class="text-sm text-muted-foreground">{{ t('editor.states.loadingVersionsDescription') }}</p>
      <UiEmptyState v-else-if="versions.length === 0" class="min-h-28" :title="t('editor.states.emptyVersionsTitle')" :description="t('editor.emptyVersions')" />
      <div v-else class="space-y-2">
        <button
          v-for="version in versions"
          :key="version.id"
          type="button"
          class="focus-ring w-full border border-current/15 bg-background/45 px-3 py-3 text-left transition-colors hover:border-current/40"
          @click="emit('loadVersion', version.id)"
        >
          <span class="block text-sm font-semibold">{{ versionLabel(version) }}</span>
          <span class="mt-1 block text-xs text-muted-foreground">{{ new Date(version.created_at).toLocaleString() }}</span>
        </button>
      </div>
    </section>

    <section class="border-t border-current/15 pt-3">
      <button type="button" class="focus-ring flex w-full items-center justify-between py-2 text-left text-sm font-semibold" @click="diagnosticsOpen = !diagnosticsOpen">
        <span>{{ t('editor.diagnostics.title') }}</span>
        <ChevronDown :class="['h-4 w-4 transition-transform', diagnosticsOpen && 'rotate-180']" />
      </button>
      <div v-if="diagnosticsOpen" class="space-y-3 pt-3 text-xs leading-5 text-muted-foreground">
        <div v-if="diagnostics.modelResolution" class="border border-current/15 p-3">
          <p><strong class="text-foreground">{{ t('editor.diagnostics.provider') }}:</strong> {{ diagnostics.modelResolution.provider_name || diagnostics.modelResolution.provider_id }}</p>
          <p><strong class="text-foreground">{{ t('editor.diagnostics.model') }}:</strong> {{ diagnostics.modelResolution.model_name || diagnostics.modelResolution.model_id }}</p>
          <p><strong class="text-foreground">{{ t('editor.diagnostics.route') }}:</strong> {{ diagnostics.modelResolution.route_key }}</p>
        </div>
        <div v-if="diagnostics.toolTrace.length" class="border border-current/15 p-3">
          <p class="font-semibold text-foreground">{{ t('editor.toolTrace.title') }}</p>
          <pre class="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono text-[11px]">{{ JSON.stringify(diagnostics.toolTrace, null, 2) }}</pre>
        </div>
        <p v-if="!diagnostics.modelResolution && diagnostics.toolTrace.length === 0">{{ t('editor.diagnostics.empty') }}</p>
      </div>
    </section>
  </div>
</template>
