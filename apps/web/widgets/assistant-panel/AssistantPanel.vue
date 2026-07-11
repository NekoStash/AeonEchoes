<script setup lang="ts">
import { Bot, Check, ChevronDown, History, Play, Plus, Replace, RotateCcw, X } from '@lucide/vue'
import { computed, ref } from 'vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiButton from '~/components/ui/Button.vue'
import UiEmptyState from '~/components/ui/EmptyState.vue'
import UiInlineNotice from '~/components/ui/InlineNotice.vue'
import UiInput from '~/components/ui/Input.vue'
import UiSelect from '~/components/ui/Select.vue'
import UiSwitch from '~/components/ui/Switch.vue'
import UiTextarea from '~/components/ui/Textarea.vue'
import type { AgentConfig } from '~/entities/agent'
import type { Chapter, ChapterVersion } from '~/entities/chapter'
import type { AgentProposal } from '~/features/agent-run'
import type { ContextSelectState } from '~/features/context-select'
import type { EditorDraftSnapshot } from '~/features/editor-draft-recovery'
import type { ModelResolution, StoryBible, ToolTrace } from '~/lib/types'

const props = defineProps<{
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
  loadingVersions: boolean
  diagnostics: { modelResolution: ModelResolution | null; toolTrace: ToolTrace[] }
}>()

const emit = defineEmits<{
  'update:prompt': [value: string]
  'update:selectedAgentId': [value: string]
  'update:contextState': [value: ContextSelectState]
  retryAgents: []
  run: []
  insert: [proposalId: string]
  replace: [proposalId: string]
  append: [proposalId: string]
  reject: [proposalId: string]
  restoreDraft: []
  keepBackend: []
  viewDraftDiff: []
  loadVersion: [versionId: string]
}>()

const { t } = useI18n()
const diagnosticsOpen = ref(false)
const agentOptions = computed(() => props.agents.map((agent) => ({
  label: agent.name,
  value: agent.id,
  description: `${agent.project_id === props.projectId ? t('editor.assistant.scopeProject') : t('editor.assistant.scopeGlobal')} · ${agent.role ? t(`agents.roles.${agent.role.replaceAll('-', '_')}`) : t('agents.roles.default')}`
})))
const currentChapterIndex = computed(() => props.chapters.findIndex((chapter) => chapter.id === props.chapter.id))
const previousLimit = computed(() => Math.max(0, currentChapterIndex.value))
const syncedCharacters = computed(() => (props.bible?.characters || []).filter((character) => character.entity_id?.trim()))

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
  <div class="space-y-5">
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
        <UiButton class="w-full" :loading="runningAgent" :disabled="!selectedAgentId || !prompt.trim()" @click="emit('run')">
          <Play class="h-4 w-4" />
          {{ t('editor.actions.runAgent') }}
        </UiButton>
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
      <UiEmptyState v-if="proposals.length === 0" class="min-h-32" :title="t('editor.proposals.emptyTitle')" :description="t('editor.proposals.emptyDescription')" />
      <article v-for="proposal in proposals" v-else :key="proposal.id" class="border border-current/15 bg-background/45 p-4">
        <div class="flex items-center justify-between gap-3">
          <UiBadge :tone="proposal.status === 'pending' ? 'info' : proposal.status === 'applied' ? 'success' : 'muted'">
            {{ t(`editor.proposals.status.${proposal.status}`) }}
          </UiBadge>
          <span class="text-[11px] text-muted-foreground">{{ new Date(proposal.createdAt).toLocaleString() }}</span>
        </div>
        <pre class="mt-3 max-h-72 overflow-auto whitespace-pre-wrap font-serif text-sm leading-7 subtle-scrollbar">{{ proposal.content }}</pre>
        <div v-if="proposal.status === 'pending'" class="mt-4 grid grid-cols-2 gap-2">
          <UiButton size="sm" variant="outline" @click="emit('insert', proposal.id)"><Plus class="h-4 w-4" />{{ t('editor.proposals.insert') }}</UiButton>
          <UiButton size="sm" variant="outline" :disabled="!selectedText" @click="emit('replace', proposal.id)"><Replace class="h-4 w-4" />{{ t('editor.proposals.replace') }}</UiButton>
          <UiButton size="sm" variant="outline" @click="emit('append', proposal.id)"><Check class="h-4 w-4" />{{ t('editor.proposals.append') }}</UiButton>
          <UiButton size="sm" variant="ghost" @click="emit('reject', proposal.id)"><X class="h-4 w-4" />{{ t('editor.proposals.reject') }}</UiButton>
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
