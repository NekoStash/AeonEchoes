<script setup lang="ts">
import { ArrowRight, CheckCircle2, Loader2, Wand2 } from '@lucide/vue'
import type { ProjectSeed } from '~/lib/types'

const { t } = useI18n()
const api = useApi()
const workspace = useWorkspaceStore()

const seed = reactive<ProjectSeed>({
  title: t('projectNew.defaults.title'),
  one_sentence_core: t('projectNew.defaults.brief'),
  tags: [t('projectNew.defaults.tags.mystery'), t('projectNew.defaults.tags.timeline')],
  world_background: t('projectNew.defaults.world'),
  protagonist: t('projectNew.defaults.protagonist'),
  central_conflict: t('projectNew.defaults.conflict'),
  style: t('projectNew.defaults.style'),
  taboos: t('projectNew.defaults.avoid')
})

const tagInput = ref(seed.tags.join(t('common.listSeparator')))
const activeProjectNewTab = ref('core')
const optimizing = ref(false)
const initializing = ref(false)
const localError = ref('')
const createdProjectId = ref('')
const createdProjectTitle = ref('')
const successCard = ref<HTMLElement | { $el?: HTMLElement } | null>(null)

watch(tagInput, (value) => {
  seed.tags = value.split(/[，,]/).map((tag) => tag.trim()).filter(Boolean)
})

const draftSteps = computed(() => {
  const coreReady = Boolean((seed.title || '').trim() && seed.one_sentence_core.trim())
  const setupReady = Boolean(seed.protagonist.trim() && seed.central_conflict.trim() && seed.world_background.trim())
  const styleReady = Boolean(seed.style.trim() && seed.taboos.trim())

  return [
    {
      key: 'core',
      title: t('projectNew.steps.core.title'),
      description: t('projectNew.steps.core.description'),
      complete: coreReady
    },
    {
      key: 'setup',
      title: t('projectNew.steps.setup.title'),
      description: t('projectNew.steps.setup.description'),
      complete: setupReady
    },
    {
      key: 'style',
      title: t('projectNew.steps.style.title'),
      description: t('projectNew.steps.style.description'),
      complete: styleReady
    }
  ]
})

const allReady = computed(() => draftSteps.value.every((step) => step.complete))
const projectNewTabs = computed(() => [
  { label: t('projectNew.tabs.core'), value: 'core', badge: draftSteps.value[0]?.complete ? t('status.ready') : undefined },
  { label: t('projectNew.tabs.world'), value: 'setup', badge: draftSteps.value[1]?.complete ? t('status.ready') : undefined },
  { label: t('projectNew.tabs.style'), value: 'style', badge: draftSteps.value[2]?.complete ? t('status.ready') : undefined },
  { label: t('projectNew.tabs.confirm'), value: 'confirm', badge: allReady.value ? t('status.ready') : undefined }
])

const seedPreview = computed(() => {
  if (seed.optimized_prompt?.trim()) return seed.optimized_prompt.trim()
  return [seed.one_sentence_core, seed.central_conflict, seed.world_background]
    .map((item) => item.trim())
    .filter(Boolean)
    .join('\n\n')
})

const checklist = computed(() => [
  { label: t('projectNew.checklist.title'), value: (seed.title || '').trim() || t('common.emptyValue') },
  { label: t('projectNew.checklist.brief'), value: seed.one_sentence_core.trim() || t('common.emptyValue') },
  { label: t('projectNew.checklist.protagonist'), value: seed.protagonist.trim() || t('common.emptyValue') },
  { label: t('projectNew.checklist.conflict'), value: seed.central_conflict.trim() || t('common.emptyValue') },
  { label: t('projectNew.checklist.tags'), value: seed.tags.length ? seed.tags.join(t('common.listSeparator')) : t('common.emptyValue') }
])

async function optimizeSeed() {
  localError.value = ''
  optimizing.value = true
  try {
    const currentTitle = (seed.title || '').trim()
    const result = await api.optimizeProjectSeed({ ...seed, title: currentTitle, tags: [...seed.tags] })
    workspace.recordResult(t('actions.organizeBrief'), result)
    Object.assign(seed, result.data, { title: currentTitle || result.data.title || seed.title })
    tagInput.value = seed.tags.join(t('common.listSeparator'))
  } catch (error) {
    const apiError = workspace.recordError(t('actions.organizeBrief'), error)
    localError.value = apiError.message || t('projectNew.errors.organizeFailed')
  } finally {
    optimizing.value = false
  }
}

async function initializeProject() {
  localError.value = ''
  initializing.value = true
  try {
    const title = (seed.title || '').trim()
    if (!title) throw new Error(t('projectNew.errors.titleRequired'))
    const result = await api.initializeProjectFull({ ...seed, title, tags: [...seed.tags] })
    workspace.recordResult(t('actions.createProject'), result)
    workspace.activeBible = result.data.story_bible
    createdProjectId.value = result.data.project.id
    createdProjectTitle.value = result.data.project.title
    workspace.openProject({
      id: result.data.project.id,
      title: result.data.project.title,
      logline: result.data.story_bible.premise,
      tags: result.data.story_bible.themes,
      updated_at: result.data.project.updated_at,
      bible_status: result.data.story_bible.approved ? 'ready' : 'draft',
      chapter_count: result.data.story_bible.chapters.length
    })
    await nextTick()
    const target = successCard.value instanceof HTMLElement ? successCard.value : successCard.value?.$el
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    target?.focus({ preventScroll: true })
  } catch (error) {
    const apiError = workspace.recordError(t('actions.createProject'), error)
    localError.value = apiError.message || t('projectNew.errors.createFailed')
  } finally {
    initializing.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <SectionHeader :title="t('projectNew.title')" :description="t('projectNew.description')">
      <template #actions>
        <UiButton :disabled="initializing" @click="initializeProject">
          <Loader2 v-if="initializing" class="h-4 w-4 animate-spin" />
          <CheckCircle2 v-else class="h-4 w-4" />
          {{ t('actions.createProject') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="workspace.errors" />
    <div v-if="localError" class="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ localError }}
    </div>

    <div class="grid gap-4 md:grid-cols-3">
      <UiCard v-for="(step, index) in draftSteps" :key="step.key" class="p-4 sm:p-5">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectNew.stepLabel', { number: index + 1 }) }}</p>
            <h2 class="mt-2 text-base font-semibold">{{ step.title }}</h2>
          </div>
          <UiBadge :variant="step.complete ? 'success' : 'muted'">
            {{ step.complete ? t('status.ready') : t('projectNew.pending') }}
          </UiBadge>
        </div>
        <p class="mt-3 text-sm leading-6 text-muted-foreground">{{ step.description }}</p>
      </UiCard>
    </div>

    <UiTabs v-model="activeProjectNewTab" :tabs="projectNewTabs" />

    <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div class="space-y-6">
        <UiCard v-show="activeProjectNewTab === 'core'" class="p-4 sm:p-6">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectNew.stepLabel', { number: 1 }) }}</p>
              <h2 class="mt-2 text-lg font-semibold">{{ t('projectNew.steps.core.title') }}</h2>
            </div>
            <UiBadge :variant="draftSteps[0]?.complete ? 'success' : 'muted'">{{ draftSteps[0]?.complete ? t('status.ready') : t('projectNew.pending') }}</UiBadge>
          </div>
          <div class="mt-5 grid gap-5">
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.name') }}</span>
              <UiInput v-model="seed.title" />
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.brief') }}</span>
              <UiTextarea v-model="seed.one_sentence_core" :rows="4" />
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.tags') }}</span>
              <UiInput v-model="tagInput" />
            </label>
          </div>
        </UiCard>

        <UiCard v-show="activeProjectNewTab === 'setup'" class="p-4 sm:p-6">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectNew.stepLabel', { number: 2 }) }}</p>
              <h2 class="mt-2 text-lg font-semibold">{{ t('projectNew.steps.setup.title') }}</h2>
            </div>
            <UiBadge :variant="draftSteps[1]?.complete ? 'success' : 'muted'">{{ draftSteps[1]?.complete ? t('status.ready') : t('projectNew.pending') }}</UiBadge>
          </div>
          <div class="mt-5 grid gap-5 lg:grid-cols-2">
            <label class="space-y-2 lg:col-span-2">
              <span class="text-sm font-medium">{{ t('projectNew.world') }}</span>
              <UiTextarea v-model="seed.world_background" :rows="7" />
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.protagonist') }}</span>
              <UiTextarea v-model="seed.protagonist" :rows="6" />
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.conflict') }}</span>
              <UiTextarea v-model="seed.central_conflict" :rows="6" />
            </label>
          </div>
        </UiCard>

        <UiCard v-show="activeProjectNewTab === 'style'" class="p-4 sm:p-6">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectNew.stepLabel', { number: 3 }) }}</p>
              <h2 class="mt-2 text-lg font-semibold">{{ t('projectNew.steps.style.title') }}</h2>
            </div>
            <UiBadge :variant="draftSteps[2]?.complete ? 'success' : 'muted'">{{ draftSteps[2]?.complete ? t('status.ready') : t('projectNew.pending') }}</UiBadge>
          </div>
          <div class="mt-5 grid gap-5 lg:grid-cols-2">
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.style') }}</span>
              <UiTextarea v-model="seed.style" :rows="5" />
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t('projectNew.avoid') }}</span>
              <UiTextarea v-model="seed.taboos" :rows="5" />
            </label>
          </div>
        </UiCard>

        <UiCard v-show="activeProjectNewTab === 'confirm'" class="p-4 sm:p-6">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 class="text-lg font-semibold">{{ t('projectNew.finalizeTitle') }}</h2>
              <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectNew.finalizeDescription') }}</p>
            </div>
            <UiButton class="w-full lg:w-auto" :disabled="initializing" @click="initializeProject">
              <Loader2 v-if="initializing" class="h-4 w-4 animate-spin" />
              <CheckCircle2 v-else class="h-4 w-4" />
              {{ t('actions.createProject') }}
            </UiButton>
          </div>
        </UiCard>
      </div>

      <aside class="space-y-6">
        <UiCard class="p-4 sm:p-6">
          <h2 class="text-lg font-semibold">{{ t('projectNew.summaryTitle') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectNew.summaryDescription') }}</p>
          <div class="mt-5 space-y-3">
            <div v-for="item in checklist" :key="item.label" class="rounded-xl border border-border bg-muted/35 p-3">
              <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ item.label }}</p>
              <p class="mt-2 break-words text-sm leading-6">{{ item.value }}</p>
            </div>
          </div>
          <div class="mt-5 rounded-xl border border-dashed border-border px-4 py-3 text-sm text-muted-foreground">
            {{ allReady ? t('projectNew.readyHint') : t('projectNew.pendingHint') }}
          </div>
        </UiCard>

        <UiCard class="p-4 sm:p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h2 class="text-lg font-semibold">{{ t('projectNew.assistant.title') }}</h2>
              <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectNew.assistant.description') }}</p>
            </div>
            <UiButton variant="outline" class="w-full sm:w-auto" :disabled="optimizing" @click="optimizeSeed">
              <Loader2 v-if="optimizing" class="h-4 w-4 animate-spin" />
              <Wand2 v-else class="h-4 w-4" />
              {{ t('actions.organizeBrief') }}
            </UiButton>
          </div>
          <div class="mt-5 rounded-xl border border-border bg-muted/35 p-4">
            <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectNew.result') }}</p>
            <pre class="mt-3 max-h-96 min-w-0 overflow-auto whitespace-pre-wrap break-words text-sm leading-6 subtle-scrollbar">{{ seedPreview || t('common.emptyValue') }}</pre>
          </div>
        </UiCard>

        <UiCard v-if="createdProjectId" ref="successCard" tabindex="-1" class="p-6 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background">
          <UiBadge variant="success">{{ t('status.ready') }}</UiBadge>
          <h2 class="mt-3 text-lg font-semibold">{{ createdProjectTitle || t('nav.project') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectNew.createdDescription') }}</p>
          <p class="mt-3 truncate font-mono text-xs text-muted-foreground" :title="createdProjectId">{{ createdProjectId }}</p>
          <UiButton class="mt-4 w-full" :to="`/projects/${createdProjectId}`">
            {{ t('projectNew.openWorkspace') }}
            <ArrowRight class="h-4 w-4" />
          </UiButton>
        </UiCard>
      </aside>
    </div>
  </div>
</template>
