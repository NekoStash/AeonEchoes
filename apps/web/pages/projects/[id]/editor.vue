<script setup lang="ts">
import { ArrowLeft, Bot, FileWarning, Loader2 } from '@lucide/vue'
import UiButton from '~/components/ui/Button.vue'
import UiDialog from '~/components/ui/Dialog.vue'
import UiEmptyState from '~/components/ui/EmptyState.vue'
import UiInlineNotice from '~/components/ui/InlineNotice.vue'
import UiSheet from '~/components/ui/Sheet.vue'
import { applyAgentProposal, createAgentProposal, type AgentProposal, type ProposalApplyMode } from '~/features/agent-run'
import { buildChapterVersionPayload, latestChapterVersion, loadChapterVersion, sortChapterVersions } from '~/features/chapter-version'
import { countWritingMetrics, resolveStrictChapter, type TextSelection } from '~/features/chapter-write'
import { buildContextSelection, createContextSelectState, type ContextSelectState } from '~/features/context-select'
import {
  buildLineDiff,
  draftDiffersFromBackend,
  readEditorDraft,
  removeEditorDraft,
  writeEditorDraft,
  type DiffLine,
  type EditorDraftSnapshot
} from '~/features/editor-draft-recovery'
import AssistantPanel from '~/widgets/assistant-panel/AssistantPanel.vue'
import WritingWorkspace from '~/widgets/writing-workspace/WritingWorkspace.vue'
import type { AgentRunResult } from '~/entities/agent'
import { preferredAgent, useAgentStore } from '~/entities/agent'
import type { Chapter, ChapterVersion } from '~/entities/chapter'
import { useChapterStore } from '~/entities/chapter'
import { useStoryBibleStore } from '~/entities/story-bible'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const chapterStore = useChapterStore()
const storyBibleStore = useStoryBibleStore()
const agentStore = useAgentStore()
const toast = useToast()

const projectId = computed(() => String(route.params.id || '').trim())
const routeChapterId = computed(() => {
  const query = route.query.chapter
  return String(Array.isArray(query) ? query[0] || '' : query || '').trim()
})
const chapters = computed(() => chapterStore.byProjectId[projectId.value] || [])
const storyBible = computed(() => storyBibleStore.byProjectId[projectId.value] || null)
const strictResolution = computed(() => resolveStrictChapter(chapters.value, routeChapterId.value))
const currentChapter = computed(() => strictResolution.value.state === 'ready' ? strictResolution.value.chapter : null)
const chapterId = computed(() => currentChapter.value?.id || '')

const title = ref('')
const content = ref('')
const baseTitle = ref('')
const baseContent = ref('')
const parentVersionId = ref('')
const versions = computed<ChapterVersion[]>(() => chapterStore.versionsByChapterId[chapterId.value] || [])
const agentListOptions = computed(() => ({ projectId: projectId.value, enabled: true as const }))
const agents = computed(() => agentStore.itemsFor(agentListOptions.value).filter((agent) => agent.enabled && (!agent.project_id || agent.project_id === projectId.value)))
const proposals = ref<AgentProposal[]>([])
const selection = ref<TextSelection>({ start: 0, end: 0 })
const prompt = ref('')
const selectedAgentId = ref('')
const contextState = ref<ContextSelectState>(createContextSelectState())
const localDraft = ref<EditorDraftSnapshot | null>(null)
const draftError = ref('')
const pageError = ref('')
const chapterLoadError = ref('')
const agentLoadError = ref('')
const loading = ref(true)
const loadingAgents = ref(false)
const loadingVersions = computed(() => chapterStore.versionListRequest.loading)
const savingVersion = computed(() => chapterStore.versionSaveRequest.loading)
const runningAgent = computed(() => agentStore.runRequest.loading)
const assistantOpen = ref(false)
const fullscreen = ref(false)
const diffOpen = ref(false)
const diffLines = ref<DiffLine[]>([])
const latestAgentRun = ref<AgentRunResult | null>(null)
const writingWorkspace = ref<{ focus: () => void; setSelection: (selection: TextSelection) => void } | null>(null)
let persistTimer: ReturnType<typeof setTimeout> | null = null
let loadSequence = 0

const dirty = computed(() => title.value !== baseTitle.value || content.value !== baseContent.value)
const metrics = computed(() => countWritingMetrics(content.value))
const selectedText = computed(() => content.value.slice(selection.value.start, selection.value.end))
const diagnostics = computed(() => ({
  modelResolution: latestAgentRun.value?.model_resolution || null,
  toolTrace: latestAgentRun.value?.tool_trace || []
}))

onMounted(loadEditor)
onBeforeUnmount(() => {
  if (persistTimer) clearTimeout(persistTimer)
  persistLocalDraftNow()
})

watch(routeChapterId, () => {
  if (!chapterLoadError.value) void loadCurrentChapter()
})
watch([title, content], () => scheduleDraftPersist())

async function loadEditor() {
  loading.value = true
  pageError.value = ''
  chapterLoadError.value = ''
  try {
    await Promise.all([
      storyBibleStore.load(projectId.value),
      chapterStore.load(projectId.value)
    ])
    await loadAgents()
    await loadCurrentChapter()
  } catch (cause) {
    console.error('[AeonEchoes Editor] Failed to load workspace or chapters.', cause)
    chapterLoadError.value = cause instanceof Error ? cause.message : t('editor.errors.loadWorkspaceFailed')
    title.value = ''
    content.value = ''
    baseTitle.value = ''
    baseContent.value = ''
  } finally {
    loading.value = false
  }
}

async function loadAgents() {
  loadingAgents.value = true
  agentLoadError.value = ''
  try {
    await agentStore.load(agentListOptions.value)
    if (!selectedAgentId.value || !agents.value.some((agent) => agent.id === selectedAgentId.value)) {
      selectedAgentId.value = preferredAgent(agents.value, projectId.value)?.id || ''
    }
  } catch (cause) {
    console.error('[AeonEchoes Editor] Failed to load agents.', cause)
    agentLoadError.value = cause instanceof Error ? cause.message : t('editor.errors.loadAgentsFailed')
    selectedAgentId.value = ''
  } finally {
    loadingAgents.value = false
  }
}

async function loadCurrentChapter() {
  const sequence = ++loadSequence
  pageError.value = ''
  proposals.value = []
  latestAgentRun.value = null
  localDraft.value = null
  draftError.value = ''
  parentVersionId.value = ''

  if (strictResolution.value.state === 'empty' || strictResolution.value.state === 'invalid') {
    title.value = ''
    content.value = ''
    baseTitle.value = ''
    baseContent.value = ''
    return
  }

  const chapter = strictResolution.value.chapter
  if (!routeChapterId.value) {
    await router.replace({ path: route.path, query: { ...route.query, chapter: chapter.id } })
    if (sequence !== loadSequence) return
  }

  try {
    const result = await chapterStore.loadVersions(projectId.value, chapter.id)
    if (sequence !== loadSequence) return
    chapterStore.versionsByChapterId[chapter.id] = sortChapterVersions(result.data)
    const latest = latestChapterVersion(versions.value)
    parentVersionId.value = latest?.id || ''
    baseTitle.value = latest?.title || chapter.title
    baseContent.value = latest?.content || ''
    title.value = baseTitle.value
    content.value = baseContent.value
    selection.value = { start: content.value.length, end: content.value.length }
    prompt.value = chapter.summary || t('editor.assistant.defaultPrompt', { title: chapter.title })
    hydrateLocalDraft()
  } catch (cause) {
    console.error('[AeonEchoes Editor] Failed to load chapter versions.', cause)
    pageError.value = chapterStore.versionListRequest.error?.message || (cause instanceof Error ? cause.message : t('editor.errors.loadVersionsFailed'))
  }
}

async function selectChapter(id: string) {
  if (!chapters.value.some((chapter) => chapter.id === id)) {
    pageError.value = t('editor.errors.chapterDoesNotExist', { id })
    return
  }
  persistLocalDraftNow()
  await router.push({ path: route.path, query: { ...route.query, chapter: id } })
}

function hydrateLocalDraft() {
  if (!import.meta.client || !chapterId.value) return
  const result = readEditorDraft(localStorage, projectId.value, chapterId.value)
  if (result.error) {
    draftError.value = t('editor.errors.draftReadFailed')
    return
  }
  if (result.value && draftDiffersFromBackend(result.value, baseTitle.value, baseContent.value, parentVersionId.value)) {
    localDraft.value = result.value
  } else if (result.value) {
    removeEditorDraft(localStorage, projectId.value, chapterId.value)
  }
}

function scheduleDraftPersist() {
  if (!import.meta.client || !chapterId.value || loading.value || loadingVersions.value) return
  if (persistTimer) clearTimeout(persistTimer)
  persistTimer = setTimeout(persistLocalDraftNow, 350)
}

function persistLocalDraftNow() {
  if (!import.meta.client || !chapterId.value || strictResolution.value.state !== 'ready') return
  if (persistTimer) {
    clearTimeout(persistTimer)
    persistTimer = null
  }
  if (!dirty.value) {
    const removal = removeEditorDraft(localStorage, projectId.value, chapterId.value)
    if (removal.error) draftError.value = t('editor.errors.draftRemoveFailed')
    return
  }
  const result = writeEditorDraft(localStorage, {
    project_id: projectId.value,
    chapter_id: chapterId.value,
    title: title.value,
    content: content.value,
    parent_version_id: parentVersionId.value || undefined
  })
  if (result.error) {
    draftError.value = t('editor.errors.draftWriteFailed')
  } else {
    draftError.value = ''
  }
}

function restoreLocalDraft() {
  if (!localDraft.value) return
  title.value = localDraft.value.title
  content.value = localDraft.value.content
  selection.value = { start: content.value.length, end: content.value.length }
  localDraft.value = null
  assistantOpen.value = false
  nextTick(() => writingWorkspace.value?.setSelection(selection.value))
  toast.info(t('editor.recovery.restored'))
}

function keepBackendDraft() {
  if (!import.meta.client || !chapterId.value) return
  const result = removeEditorDraft(localStorage, projectId.value, chapterId.value)
  if (result.error) {
    draftError.value = t('editor.errors.draftRemoveFailed')
    return
  }
  localDraft.value = null
  title.value = baseTitle.value
  content.value = baseContent.value
  assistantOpen.value = false
  toast.info(t('editor.recovery.backendKept'))
}

function showDraftDiff() {
  if (!localDraft.value) return
  diffLines.value = buildLineDiff(baseContent.value, localDraft.value.content)
  diffOpen.value = true
}

async function syncRealChapterTitle(chapter: Chapter, nextTitle: string) {
  const normalizedTitle = nextTitle.trim()
  if (!normalizedTitle || normalizedTitle === chapter.title) return chapter
  const result = await chapterStore.update(projectId.value, {
    chapter_id: chapter.id,
    number: chapter.number,
    title: normalizedTitle,
    status: chapter.status,
    summary: chapter.summary,
    metadata: chapter.metadata
  })
  return result.data
}

async function saveVersion() {
  if (strictResolution.value.state !== 'ready') {
    pageError.value = t('editor.errors.realChapterRequired')
    return
  }
  pageError.value = ''
  try {
    const chapter = strictResolution.value.chapter
    await syncRealChapterTitle(chapter, title.value)
    const payload = buildChapterVersionPayload(chapters.value, projectId.value, chapterId.value, {
      title: title.value,
      content: content.value,
      changeNote: t('editor.changeNotes.manualSave'),
      parentVersionId: parentVersionId.value || undefined
    })
    const result = await chapterStore.saveVersion(projectId.value, payload)
    chapterStore.versionsByChapterId[chapterId.value] = sortChapterVersions(chapterStore.versionsByChapterId[chapterId.value] || [])
    parentVersionId.value = result.data.chapter_version.id
    baseTitle.value = result.data.chapter_version.title
    baseContent.value = result.data.chapter_version.content
    title.value = baseTitle.value
    content.value = baseContent.value
    localDraft.value = null
    if (import.meta.client) removeEditorDraft(localStorage, projectId.value, chapterId.value)
    toast.success(t('editor.feedback.versionSaved'))
  } catch (cause) {
    console.error('[AeonEchoes Editor] Failed to update the chapter title or create a chapter version.', cause)
    pageError.value = chapterStore.updateRequest.error?.message
      || chapterStore.versionSaveRequest.error?.message
      || (cause instanceof Error ? cause.message : t('editor.errors.saveVersionFailed'))
  }
}

async function runAgent() {
  if (strictResolution.value.state !== 'ready') {
    pageError.value = t('editor.errors.realChapterRequired')
    return
  }
  const agent = agents.value.find((item) => item.id === selectedAgentId.value && item.enabled)
  if (!agent) {
    pageError.value = t('editor.errors.noAgentConfigured')
    return
  }
  pageError.value = ''
  try {
    const contextSelection = buildContextSelection(chapters.value, storyBible.value, projectId.value, chapterId.value, contextState.value)
    const result = await agentStore.run(agent.id, {
      project_id: projectId.value,
      task_type: 'generic',
      input: {
        chapter_id: chapterId.value,
        instruction: prompt.value.trim(),
        title: title.value,
        content: content.value,
        selected_text: selectedText.value || undefined
      },
      context_selection: contextSelection
    })
    latestAgentRun.value = result.data
    proposals.value = [createAgentProposal(agent.id, result.data), ...proposals.value]
    toast.success(t('editor.proposals.received'))
  } catch (cause) {
    console.error('[AeonEchoes Editor] Agent Run failed.', cause)
    pageError.value = agentStore.runRequest.error?.message || (cause instanceof Error ? cause.message : t('editor.errors.agentRunFailed'))
  }
}

function handleProposal(proposalId: string, mode: ProposalApplyMode) {
  const index = proposals.value.findIndex((proposal) => proposal.id === proposalId)
  if (index < 0) return
  try {
    const application = applyAgentProposal(content.value, proposals.value[index]!, mode, selection.value)
    proposals.value[index] = application.proposal
    if (mode !== 'reject') {
      content.value = application.content
      selection.value = application.selection
      assistantOpen.value = false
      nextTick(() => writingWorkspace.value?.setSelection(application.selection))
    }
  } catch (cause) {
    pageError.value = cause instanceof Error ? cause.message : t('editor.errors.proposalApplyFailed')
  }
}

function loadVersion(versionId: string) {
  const version = versions.value.find((item) => item.id === versionId)
  if (!version) return
  const loaded = loadChapterVersion(version)
  title.value = loaded.title
  content.value = loaded.content
  baseTitle.value = loaded.title
  baseContent.value = loaded.content
  parentVersionId.value = loaded.parentVersionId
  selection.value = { start: content.value.length, end: content.value.length }
  assistantOpen.value = false
  nextTick(() => writingWorkspace.value?.setSelection(selection.value))
}
</script>

<template>
  <div data-testid="editor-page" class="-mx-4 min-h-[calc(100dvh-var(--layout-height-topbar))] w-[calc(100%+2rem)] max-w-[96rem] px-3 py-3 sm:mx-auto sm:w-full sm:px-4 sm:py-4 lg:px-5">
    <div v-if="loading" class="flex min-h-[60vh] items-center justify-center text-muted-foreground">
      <Loader2 class="mr-2 h-5 w-5 animate-spin" />
      {{ t('editor.states.loadingWorkspace') }}
    </div>

    <UiInlineNotice
      v-else-if="chapterLoadError"
      data-testid="editor-chapter-load-error"
      tone="danger"
      :title="t('editor.errors.title')"
      :description="chapterLoadError"
      class="mx-auto mt-8 max-w-3xl"
    >
      <template #actions><UiButton variant="outline" @click="loadEditor">{{ t('common.retry') }}</UiButton></template>
    </UiInlineNotice>

    <UiEmptyState
      v-else-if="strictResolution.state === 'empty'"
      data-testid="editor-empty-chapters"
      class="mx-auto min-h-[55vh] max-w-3xl"
      :title="t('editor.emptyProject.title')"
      :description="t('editor.emptyProject.description')"
    >
      <template #icon><FileWarning class="h-6 w-6" /></template>
      <template #actions>
        <UiButton :to="`/projects/${projectId}`"><ArrowLeft class="h-4 w-4" />{{ t('editor.emptyProject.back') }}</UiButton>
      </template>
    </UiEmptyState>

    <UiEmptyState
      v-else-if="strictResolution.state === 'invalid'"
      data-testid="editor-invalid-chapter"
      class="mx-auto min-h-[55vh] max-w-3xl"
      :title="t('editor.invalidChapter.title')"
      :description="t('editor.invalidChapter.description', { id: strictResolution.requestedChapterId })"
    >
      <template #icon><FileWarning class="h-6 w-6" /></template>
      <template #actions>
        <UiButton :to="`/projects/${projectId}/editor?chapter=${encodeURIComponent(chapters[0]?.id || '')}`">{{ t('editor.invalidChapter.openFirst') }}</UiButton>
      </template>
    </UiEmptyState>

    <template v-else-if="currentChapter">
      <UiInlineNotice v-if="pageError" tone="danger" :title="t('editor.errors.title')" :description="pageError" class="mb-4" />
      <UiInlineNotice v-if="draftError" tone="danger" :title="t('editor.errors.draftTitle')" :description="draftError" class="mb-4" />

      <div data-testid="editor-layout" class="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_21rem] 2xl:grid-cols-[minmax(0,1fr)_22rem]">
        <WritingWorkspace
          ref="writingWorkspace"
          :chapter="currentChapter"
          :chapters="chapters"
          :title="title"
          :content="content"
          :selected-chapter-id="chapterId"
          :characters="metrics.characters"
          :paragraphs="metrics.paragraphs"
          :dirty="dirty"
          :saving="savingVersion"
          :fullscreen="fullscreen"
          @update:title="title = $event"
          @update:content="content = $event"
          @update:selected-chapter-id="selectChapter"
          @update:fullscreen="fullscreen = $event"
          @selection="selection = $event"
          @save="saveVersion"
          @assistant="assistantOpen = true"
        />

        <aside data-testid="editor-assistant" class="sticky top-[calc(var(--layout-height-topbar)+1rem)] hidden max-h-[calc(100dvh-var(--layout-height-topbar)-2rem)] overflow-y-auto border border-border bg-surface-muted p-3 text-surface-foreground subtle-scrollbar xl:block">
          <AssistantPanel
            :agents="agents"
            :project-id="projectId"
            :agent-load-error="agentLoadError"
            :chapters="chapters"
            :chapter="currentChapter"
            :bible="storyBible"
            :versions="versions"
            :proposals="proposals"
            :prompt="prompt"
            :selected-agent-id="selectedAgentId"
            :context-state="contextState"
            :selected-text="selectedText"
            :local-draft="localDraft"
            :loading-agents="loadingAgents"
            :running-agent="runningAgent"
            :loading-versions="loadingVersions"
            :diagnostics="diagnostics"
            @update:prompt="prompt = $event"
            @update:selected-agent-id="selectedAgentId = $event"
            @update:context-state="contextState = $event"
            @retry-agents="loadAgents"
            @run="runAgent"
            @insert="handleProposal($event, 'insert')"
            @replace="handleProposal($event, 'replace')"
            @append="handleProposal($event, 'append')"
            @reject="handleProposal($event, 'reject')"
            @restore-draft="restoreLocalDraft"
            @keep-backend="keepBackendDraft"
            @view-draft-diff="showDraftDiff"
            @load-version="loadVersion"
          />
        </aside>
      </div>

      <UiButton class="fixed bottom-20 right-4 z-30 border-2 border-foreground xl:hidden" size="lg" @click="assistantOpen = true">
        <Bot class="h-5 w-5" />{{ t('editor.actions.openAssistant') }}
      </UiButton>

      <UiSheet v-model:open="assistantOpen" class="w-[min(96vw,34rem)]" :title="t('editor.assistant.sheetTitle')" :description="t('editor.assistant.sheetDescription')">
        <AssistantPanel
          v-if="assistantOpen"
          :agents="agents"
          :project-id="projectId"
          :agent-load-error="agentLoadError"
          :chapters="chapters"
          :chapter="currentChapter"
          :bible="storyBible"
          :versions="versions"
          :proposals="proposals"
          :prompt="prompt"
          :selected-agent-id="selectedAgentId"
          :context-state="contextState"
          :selected-text="selectedText"
          :local-draft="localDraft"
          :loading-agents="loadingAgents"
          :running-agent="runningAgent"
          :loading-versions="loadingVersions"
          :diagnostics="diagnostics"
          @update:prompt="prompt = $event"
          @update:selected-agent-id="selectedAgentId = $event"
          @update:context-state="contextState = $event"
          @retry-agents="loadAgents"
          @run="runAgent"
          @insert="handleProposal($event, 'insert')"
          @replace="handleProposal($event, 'replace')"
          @append="handleProposal($event, 'append')"
          @reject="handleProposal($event, 'reject')"
          @restore-draft="restoreLocalDraft"
          @keep-backend="keepBackendDraft"
          @view-draft-diff="showDraftDiff"
          @load-version="loadVersion"
        />
      </UiSheet>

      <UiDialog v-model:open="diffOpen" size="xl" :title="t('editor.recovery.diffTitle')" :description="t('editor.recovery.diffDescription')">
        <div class="overflow-hidden border border-border bg-surface font-mono text-xs text-surface-foreground">
          <div
            v-for="(line, index) in diffLines"
            :key="`${index}-${line.kind}`"
            :class="[
              'grid grid-cols-[2.5rem_1fr] border-b border-current/10 last:border-b-0',
              line.kind === 'added' && 'bg-emerald-500/10',
              line.kind === 'removed' && 'bg-red-500/10'
            ]"
          >
            <span class="border-r border-current/10 px-2 py-1 text-center opacity-55">{{ line.kind === 'added' ? '+' : line.kind === 'removed' ? '−' : ' ' }}</span>
            <span class="whitespace-pre-wrap px-3 py-1">{{ line.text || ' ' }}</span>
          </div>
        </div>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UiButton variant="outline" @click="keepBackendDraft; diffOpen = false">{{ t('editor.recovery.keepBackend') }}</UiButton>
            <UiButton @click="restoreLocalDraft; diffOpen = false">{{ t('editor.recovery.restore') }}</UiButton>
          </div>
        </template>
      </UiDialog>
    </template>
  </div>
</template>
