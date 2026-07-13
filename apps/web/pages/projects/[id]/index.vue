<script setup lang="ts">
import { AlertTriangle, ArrowLeft, GitFork, Plus, RotateCcw, Save } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import type { CreateChapterRequest } from '~/entities/chapter'
import { useChapterStore } from '~/entities/chapter'
import type { StoryBible } from '~/entities/story-bible'
import { useStoryBibleStore } from '~/entities/story-bible'
import { ChapterCreateDialog } from '~/features/chapter-create'
import { CharacterSyncPanel, type CharacterSyncState } from '~/features/character-sync'
import type { ChapterStatus } from '~/entities/chapter'
import {
  cloneStoryBible,
  isConflictError,
  isStoryBibleDirty,
  StoryBibleEditor,
  type StoryBibleSaveState
} from '~/features/story-bible-edit'
import { ChapterTree } from '~/widgets/chapter-tree'
import { ProjectOverview } from '~/widgets/project-overview'

const route = useRoute()
const { t } = useI18n()
const projectId = computed(() => String(route.params.id))
const storyBibleStore = useStoryBibleStore()
const chapterStore = useChapterStore()
const { byProjectId: storyBibles } = storeToRefs(storyBibleStore)
const { byProjectId: chaptersByProject } = storeToRefs(chapterStore)

const persistedBible = ref<StoryBible | null>(null)
const draftBible = ref<StoryBible | null>(null)
const workspaceReady = ref(false)
const editableBible = computed({
  get: () => {
    if (!draftBible.value) throw new Error('Story bible draft is not loaded.')
    return draftBible.value
  },
  set: (value: StoryBible) => {
    draftBible.value = value
  }
})
const saveState = ref<StoryBibleSaveState>('saved')
const characterSyncState = ref<CharacterSyncState>('idle')
const pageError = ref('')
const characterSyncError = ref('')
const chapterCreateError = ref('')
const chapterCreateOpen = ref(false)
const chapterStatusError = ref('')
const toast = useToast()
const storyBibleSection = ref<HTMLElement | null>(null)
const pendingLeaveCleanup = ref<null | (() => void)>(null)
const pendingNavigation = ref<null | { resolve: (allow: boolean) => void }>(null)
const leaveConfirmOpen = computed({
  get: () => Boolean(pendingNavigation.value),
  set: (value: boolean) => {
    if (value || !pendingNavigation.value) return
    pendingNavigation.value.resolve(false)
    pendingNavigation.value = null
  }
})

const chapters = computed(() => chaptersByProject.value[projectId.value] || [])
const isDirty = computed(() => Boolean(
  persistedBible.value
  && draftBible.value
  && isStoryBibleDirty(draftBible.value, persistedBible.value)
))
const isBusy = computed(() => saveState.value === 'saving' || characterSyncState.value === 'syncing')
const statusLabel = computed(() => t(`projectOverview.saveState.${saveState.value}`))

async function loadWorkspace() {
  pageError.value = ''
  chapterCreateError.value = ''
  workspaceReady.value = false
  persistedBible.value = null
  draftBible.value = null
  try {
    const [bibleResult] = await Promise.all([
      storyBibleStore.load(projectId.value),
      chapterStore.load(projectId.value)
    ])
    persistedBible.value = cloneStoryBible(bibleResult.data)
    draftBible.value = cloneStoryBible(bibleResult.data)
    workspaceReady.value = true
    saveState.value = 'saved'
    characterSyncState.value = 'idle'
  } catch (error) {
    console.error('[AeonEchoes Project Workspace] Failed to load workspace.', error)
    pageError.value = storyBibleStore.loadRequest.error?.message
      || chapterStore.listRequest.error?.message
      || (error instanceof Error ? error.message : t('projectOverview.errors.loadFailed'))
  }
}

watch(projectId, async () => {
  await loadWorkspace()
  await applyRouteIntent()
})
watch(draftBible, () => {
  if (!draftBible.value || !persistedBible.value || saveState.value === 'saving' || saveState.value === 'conflict') return
  saveState.value = isDirty.value ? 'dirty' : 'saved'
  if (isDirty.value && characterSyncState.value === 'synced') characterSyncState.value = 'idle'
}, { deep: true })
watch(() => storyBibles.value[projectId.value], (incoming) => {
  if (!workspaceReady.value || !incoming || saveState.value === 'saving') return
  if (!persistedBible.value) {
    persistedBible.value = cloneStoryBible(incoming)
    draftBible.value = cloneStoryBible(incoming)
    saveState.value = 'saved'
    return
  }
  if (isDirty.value && isStoryBibleDirty(incoming, persistedBible.value)) {
    saveState.value = 'conflict'
  }
})

async function applyRouteIntent() {
  if (!draftBible.value) return
  await nextTick()
  if (String(route.query.createChapter || '') === '1') {
    chapterCreateOpen.value = true
  }
  if (String(route.query.section || '') === 'story') {
    storyBibleSection.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    storyBibleSection.value?.focus({ preventScroll: true })
  }
}

onMounted(() => {
  void loadWorkspace().then(applyRouteIntent)
  if (!import.meta.client) return
  const beforeUnload = (event: BeforeUnloadEvent) => {
    if (!isDirty.value && saveState.value !== 'conflict') return
    event.preventDefault()
    event.returnValue = ''
  }
  window.addEventListener('beforeunload', beforeUnload)
  pendingLeaveCleanup.value = () => window.removeEventListener('beforeunload', beforeUnload)
})

watch(() => route.query, () => void applyRouteIntent(), { deep: true })
watch(chapterCreateOpen, (open) => {
  if (open || String(route.query.createChapter || '') !== '1') return
  const query = { ...route.query }
  delete query.createChapter
  void navigateTo({ path: route.path, query }, { replace: true })
})

onBeforeUnmount(() => pendingLeaveCleanup.value?.())
onBeforeRouteLeave(() => {
  if (!isDirty.value && saveState.value !== 'conflict') return true
  return new Promise<boolean>((resolve) => {
    pendingNavigation.value?.resolve(false)
    pendingNavigation.value = { resolve }
  })
})

function confirmLeave() {
  pendingNavigation.value?.resolve(true)
  pendingNavigation.value = null
}

async function saveStoryBible() {
  if (!draftBible.value || !isDirty.value || saveState.value === 'conflict') return
  saveState.value = 'saving'
  pageError.value = ''
  try {
    const result = await storyBibleStore.save(projectId.value, cloneStoryBible(draftBible.value))
    persistedBible.value = cloneStoryBible(result.data)
    draftBible.value = cloneStoryBible(result.data)
    saveState.value = 'saved'
  } catch (error) {
    console.error('[AeonEchoes Project Workspace] Failed to save story bible.', error)
    saveState.value = isConflictError(error) ? 'conflict' : 'failed'
    pageError.value = storyBibleStore.saveRequest.error?.message || (error instanceof Error ? error.message : t('projectOverview.errors.saveFailed'))
  }
}

function resetDraft() {
  if (!persistedBible.value) return
  draftBible.value = cloneStoryBible(persistedBible.value)
  saveState.value = 'saved'
  characterSyncState.value = 'idle'
  pageError.value = ''
}

async function reloadAfterConflict() {
  await loadWorkspace()
}

async function syncCharacters() {
  if (!draftBible.value || isDirty.value || saveState.value === 'conflict') return
  characterSyncState.value = 'syncing'
  characterSyncError.value = ''
  try {
    const syncedBible = await storyBibleStore.syncCharacters(projectId.value, cloneStoryBible(draftBible.value))
    const saveResult = await storyBibleStore.save(projectId.value, syncedBible)
    persistedBible.value = cloneStoryBible(saveResult.data)
    draftBible.value = cloneStoryBible(saveResult.data)
    saveState.value = 'saved'
    characterSyncState.value = 'synced'
  } catch (error) {
    console.error('[AeonEchoes Project Workspace] Character sync side effect failed.', error)
    characterSyncState.value = 'failed'
    characterSyncError.value = storyBibleStore.syncRequest.error?.message
      || storyBibleStore.saveRequest.error?.message
      || (error instanceof Error ? error.message : t('projectOverview.characterSync.failed'))
  }
}

async function createChapter(request: CreateChapterRequest) {
  chapterCreateError.value = ''
  try {
    await chapterStore.create(projectId.value, request)
    chapterCreateOpen.value = false
  } catch (error) {
    console.error('[AeonEchoes Project Workspace] Failed to create a real chapter.', error)
    chapterCreateError.value = chapterStore.createRequest.error?.message || (error instanceof Error ? error.message : t('projectOverview.chapterCreate.failed'))
  }
}

async function updateChapterStatus(payload: { chapterId: string; status: ChapterStatus }) {
  const chapter = chapters.value.find((item) => item.id === payload.chapterId)
  if (!chapter) {
    chapterStatusError.value = t('projectOverview.errors.chapterNotFound')
    toast.error(chapterStatusError.value)
    return
  }
  if (chapter.status === payload.status) return
  chapterStatusError.value = ''
  try {
    await chapterStore.update(projectId.value, {
      chapter_id: chapter.id,
      number: chapter.number,
      title: chapter.title,
      status: payload.status,
      summary: chapter.summary,
      metadata: chapter.metadata
    })
    toast.success(t('projectOverview.chapterStatus.updated'))
  } catch (error) {
    console.error('[AeonEchoes Project Workspace] Failed to update chapter status.', error)
    chapterStatusError.value = chapterStore.updateRequest.error?.message
      || (error instanceof Error ? error.message : t('projectOverview.chapterStatus.failed'))
    toast.error(chapterStatusError.value)
  }
}
</script>

<template>
  <div class="mx-auto max-w-[94rem] pb-24">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-3 border-b border-border pb-4">
      <UiButton variant="ghost" to="/projects">
        <ArrowLeft class="h-4 w-4" aria-hidden="true" />
        {{ t('nav.projects') }}
      </UiButton>
      <div class="flex flex-wrap items-center gap-2">
        <UiButton variant="outline" :to="`/projects/${projectId}/graph`">
          <GitFork class="h-4 w-4" aria-hidden="true" />
          {{ t('nav.graph') }}
        </UiButton>
        <UiButton variant="outline" :to="`/projects/${projectId}?createChapter=1`">
          <Plus class="h-4 w-4" aria-hidden="true" />
          {{ t('projectOverview.chapterCreate.action') }}
        </UiButton>
      </div>
    </div>

    <UiInlineNotice
      v-if="pageError && !workspaceReady"
      data-testid="project-workspace-load-error"
      tone="danger"
      :title="t('projectOverview.errors.loadFailed')"
      :description="pageError"
      class="mb-6"
    >
      <template #actions>
        <UiButton variant="outline" size="sm" :loading="storyBibleStore.loadRequest.loading || chapterStore.listRequest.loading" @click="loadWorkspace">
          {{ t('common.retry') }}
        </UiButton>
      </template>
    </UiInlineNotice>

    <div v-else-if="!workspaceReady" class="space-y-5" aria-busy="true">
      <div class="h-48 animate-pulse border-y-2 border-foreground bg-muted/40" />
      <div class="h-96 animate-pulse border-y border-border bg-muted/30" />
    </div>

    <template v-else-if="workspaceReady && draftBible">
      <div v-if="pageError" role="alert" class="mb-6 border-l-4 border-destructive bg-destructive/10 px-5 py-4 text-sm text-destructive">
        {{ pageError }}
      </div>
      <ProjectOverview :bible="editableBible" :chapters="chapters" />

      <div class="sticky top-[var(--layout-height-topbar)] z-30 -mx-4 mt-6 border-y border-border bg-background px-4 py-3 sm:mx-0 sm:px-0">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-3">
            <span
              class="inline-flex items-center gap-2 border px-3 py-1 text-xs font-bold uppercase tracking-[0.15em]"
              :class="{
                'border-state-success-border bg-state-success-surface text-state-success-foreground': saveState === 'saved',
                'border-state-warning-border bg-state-warning-surface text-state-warning-foreground': saveState === 'dirty' || saveState === 'saving',
                'border-state-danger-border bg-state-danger-surface text-state-danger-foreground': saveState === 'failed' || saveState === 'conflict'
              }"
              role="status"
            >
              <AlertTriangle v-if="saveState === 'failed' || saveState === 'conflict'" class="h-4 w-4" aria-hidden="true" />
              {{ statusLabel }}
            </span>
            <p v-if="saveState === 'conflict'" class="text-sm text-state-danger-foreground">{{ t('projectOverview.saveState.conflictDescription') }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <UiButton v-if="saveState === 'conflict'" variant="outline" @click="reloadAfterConflict">
              <RotateCcw class="h-4 w-4" aria-hidden="true" />
              {{ t('projectOverview.leaveProtection.loadServer') }}
            </UiButton>
            <UiButton v-else variant="outline" :disabled="!isDirty || isBusy" @click="resetDraft">
              <RotateCcw class="h-4 w-4" aria-hidden="true" />
              {{ t('actions.reset') }}
            </UiButton>
            <UiButton :disabled="!isDirty || saveState === 'conflict'" :loading="saveState === 'saving'" :loading-label="t('actions.saving')" @click="saveStoryBible">
              <Save class="h-4 w-4" aria-hidden="true" />
              {{ saveState === 'saving' ? t('actions.saving') : t('projectOverview.saveStoryBible') }}
            </UiButton>
          </div>
        </div>
      </div>

      <main class="mt-10 space-y-12">
        <div ref="storyBibleSection" tabindex="-1" class="scroll-mt-24 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-4 focus-visible:ring-offset-background">
          <StoryBibleEditor v-model="editableBible" :disabled="isBusy" />
        </div>
        <CharacterSyncPanel
          :bible="editableBible"
          :state="characterSyncState"
          :error="characterSyncError"
          :disabled="isBusy || isDirty || saveState === 'conflict'"
          @sync="syncCharacters"
        />

        <section class="border-y-4 border-double border-foreground py-8">
          <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.chapterCreate.eyebrow') }}</p>
              <h2 class="mt-2 font-serif text-3xl font-semibold">{{ t('projectOverview.chapterCreate.sectionTitle') }}</h2>
              <p class="mt-3 max-w-2xl text-sm leading-7 text-muted-foreground">{{ t('projectOverview.chapterCreate.sectionDescription') }}</p>
            </div>
            <UiButton size="lg" :to="`/projects/${projectId}?createChapter=1`">
              <Plus class="h-5 w-5" aria-hidden="true" />
              {{ t('projectOverview.chapterCreate.action') }}
            </UiButton>
          </div>
        </section>

        <UiInlineNotice
          v-if="chapterStatusError"
          tone="danger"
          :title="t('projectOverview.chapterStatus.failed')"
          :description="chapterStatusError"
        />

        <ChapterTree
          :project-id="projectId"
          :chapters="chapters"
          :loading="chapterStore.listRequest.loading"
          :updating="chapterStore.updateRequest.loading"
          @update-status="updateChapterStatus"
        />
      </main>

      <ChapterCreateDialog
        v-model:open="chapterCreateOpen"
        :chapters="chapters"
        :loading="chapterStore.createRequest.loading"
        :error="chapterCreateError"
        @confirm="createChapter"
      />
      <UiConfirm
        v-model:open="leaveConfirmOpen"
        :title="t('projectOverview.leaveProtection.title')"
        :description="t('projectOverview.leaveProtection.message')"
        tone="danger"
        @confirm="confirmLeave"
      />
    </template>
  </div>
</template>
