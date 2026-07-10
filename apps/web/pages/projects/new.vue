<script setup lang="ts">
import type { InitializeProjectResponse, ProjectSeed } from '~/lib/types'
import { projectSummaryFromInitialization } from '~/features/project-create/project-create'
import ProjectCreateFlow from '~/features/project-create/ProjectCreateFlow.vue'
import { useChapterStore } from '~/entities/chapter'
import { useProjectStore } from '~/entities/project'
import { useStoryBibleStore } from '~/entities/story-bible'

const { t } = useI18n()
const workspace = useWorkspaceStore()
const projectStore = useProjectStore()
const storyBibleStore = useStoryBibleStore()
const chapterStore = useChapterStore()

const seed = ref<ProjectSeed>({
  title: t('projectNew.defaults.title'),
  one_sentence_core: t('projectNew.defaults.brief'),
  tags: [t('projectNew.defaults.tags.mystery'), t('projectNew.defaults.tags.timeline')],
  world_background: t('projectNew.defaults.world'),
  protagonist: t('projectNew.defaults.protagonist'),
  central_conflict: t('projectNew.defaults.conflict'),
  style: t('projectNew.defaults.style'),
  taboos: t('projectNew.defaults.avoid')
})
const creating = ref(false)
const localError = ref('')
const created = ref<InitializeProjectResponse | null>(null)

async function createProject(nextSeed: ProjectSeed) {
  localError.value = ''
  creating.value = true
  try {
    const title = nextSeed.title?.trim()
    if (!title) throw new Error(t('projectNew.errors.titleRequired'))
    const result = await projectStore.initialize({ ...nextSeed, title, tags: [...nextSeed.tags] })
    const project = projectSummaryFromInitialization(result.data)
    projectStore.upsert(project)
    storyBibleStore.set(result.data.project.id, result.data.story_bible)
    chapterStore.setProjectChapters(result.data.project.id, [])
    workspace.openProject(project)
    created.value = result.data
  } catch (error) {
    localError.value = projectStore.createRequest.error?.message || (error instanceof Error ? error.message : t('projectNew.errors.createFailed'))
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="mx-auto w-full max-w-[var(--layout-width-page)] px-[var(--layout-gutter)] py-6 sm:py-10">
    <header class="max-w-4xl border-b border-border pb-7">
      <p class="text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">{{ t('projectCreateFlow.pageEyebrow') }}</p>
      <h1 class="mt-3 text-4xl font-semibold tracking-[-0.045em] sm:text-5xl">{{ t('projectCreateFlow.pageTitle') }}</h1>
      <p class="mt-4 max-w-2xl text-base leading-7 text-muted-foreground">{{ t('projectCreateFlow.pageDescription') }}</p>
    </header>

    <ProjectCreateFlow
      class="mt-8"
      :initial-seed="seed"
      :creating="creating"
      :error="localError"
      :created="created"
      @create="createProject"
      @clear-error="localError = ''"
    />
  </div>
</template>
