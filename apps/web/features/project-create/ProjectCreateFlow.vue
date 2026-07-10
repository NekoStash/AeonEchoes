<script setup lang="ts">
import { ArrowLeft, ArrowRight, Check, FilePlus2, LibraryBig } from '@lucide/vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiButton from '~/components/ui/Button.vue'
import UiField from '~/components/ui/Field.vue'
import UiInlineNotice from '~/components/ui/InlineNotice.vue'
import UiInput from '~/components/ui/Input.vue'
import UiTextarea from '~/components/ui/Textarea.vue'
import type { InitializeProjectResponse, ProjectSeed } from '~/lib/types'
import { createdProjectDestinations, isProjectCreateStepComplete, projectCreateSteps, splitProjectTags, type ProjectCreateStep } from './project-create'

const props = defineProps<{
  initialSeed: ProjectSeed
  creating?: boolean
  error?: string
  created?: InitializeProjectResponse | null
}>()

const emit = defineEmits<{
  create: [seed: ProjectSeed]
  clearError: []
}>()

const { t } = useI18n()
const seed = reactive<ProjectSeed>(JSON.parse(JSON.stringify(props.initialSeed)) as ProjectSeed)
const tagInput = ref(seed.tags.join(t('common.listSeparator')))
const stepIndex = ref(0)
const fieldError = ref('')
const successTarget = ref<HTMLElement | null>(null)

const currentStep = computed(() => projectCreateSteps[stepIndex.value] as ProjectCreateStep)
const isLastStep = computed(() => stepIndex.value === projectCreateSteps.length - 1)
const destinations = computed(() => props.created ? createdProjectDestinations(props.created.project.id) : null)

watch(tagInput, (value) => {
  seed.tags = splitProjectTags(value)
})

watch(() => props.created, async (created) => {
  if (!created) return
  await nextTick()
  successTarget.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  successTarget.value?.focus({ preventScroll: true })
})

watch(() => props.initialSeed, (value) => {
  Object.assign(seed, JSON.parse(JSON.stringify(value)) as ProjectSeed)
  tagInput.value = seed.tags.join(t('common.listSeparator'))
}, { deep: true })

function stepTitle(step: ProjectCreateStep) {
  return t(`projectCreateFlow.steps.${step}.title`)
}

function stepDescription(step: ProjectCreateStep) {
  return t(`projectCreateFlow.steps.${step}.description`)
}

function validateCurrentStep() {
  fieldError.value = ''
  if (isProjectCreateStepComplete(currentStep.value, seed)) return true
  fieldError.value = t(`projectCreateFlow.steps.${currentStep.value}.error`)
  return false
}

function goNext() {
  emit('clearError')
  if (!validateCurrentStep()) return
  stepIndex.value = Math.min(stepIndex.value + 1, projectCreateSteps.length - 1)
}

function goBack() {
  emit('clearError')
  fieldError.value = ''
  stepIndex.value = Math.max(stepIndex.value - 1, 0)
}

function submit() {
  emit('clearError')
  if (!isProjectCreateStepComplete('review', seed)) {
    fieldError.value = t('projectCreateFlow.reviewIncomplete')
    return
  }
  emit('create', { ...seed, title: seed.title?.trim(), tags: [...seed.tags] })
}

</script>

<template>
  <div class="grid gap-8 xl:grid-cols-[15rem_minmax(0,1fr)]">
    <nav :aria-label="t('projectCreateFlow.progressLabel')" class="xl:sticky xl:top-6 xl:self-start">
      <ol class="grid grid-cols-4 border-y border-border xl:block xl:border-y-0 xl:border-l xl:border-border">
        <li v-for="(step, index) in projectCreateSteps" :key="step" class="min-w-0 xl:border-b xl:border-border xl:last:border-b-0">
          <button
            type="button"
            :disabled="index > stepIndex"
            :aria-current="index === stepIndex ? 'step' : undefined"
            class="focus-ring flex w-full min-w-0 flex-col gap-1 px-2 py-3 text-left disabled:cursor-not-allowed disabled:opacity-45 xl:flex-row xl:items-start xl:gap-3 xl:px-4 xl:py-4"
            :class="index === stepIndex ? 'bg-foreground text-background' : 'hover:bg-surface-muted'"
            @click="stepIndex = index"
          >
            <span class="font-mono text-xs">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="min-w-0">
              <span class="block truncate text-xs font-semibold xl:text-sm">{{ stepTitle(step) }}</span>
              <span class="mt-1 hidden text-xs leading-5 opacity-70 xl:block">{{ stepDescription(step) }}</span>
            </span>
          </button>
        </li>
      </ol>
    </nav>

    <div class="min-w-0">
      <div class="border-b border-border pb-5">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t('projectCreateFlow.stepCounter', { current: stepIndex + 1, total: projectCreateSteps.length }) }}</p>
        <h2 class="mt-2 text-3xl font-semibold tracking-[-0.03em]">{{ stepTitle(currentStep) }}</h2>
        <p class="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">{{ stepDescription(currentStep) }}</p>
      </div>

      <UiInlineNotice v-if="error" tone="danger" class="mt-5" :title="t('projectCreateFlow.createErrorTitle')" :description="error" />
      <UiInlineNotice v-if="fieldError" tone="warning" class="mt-5" :title="t('projectCreateFlow.incompleteTitle')" :description="fieldError" />

      <form class="mt-7" @submit.prevent="isLastStep ? submit() : goNext()">
        <div v-if="currentStep === 'core'" class="grid gap-6">
          <UiField :label="t('projectNew.name')" :description="t('projectCreateFlow.fields.titleDescription')" required>
            <template #default="slotProps"><UiInput v-model="seed.title" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
          <UiField :label="t('projectNew.brief')" :description="t('projectCreateFlow.fields.briefDescription')" required>
            <template #default="slotProps"><UiTextarea v-model="seed.one_sentence_core" :rows="5" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
          <UiField :label="t('projectNew.tags')" :description="t('projectCreateFlow.fields.tagsDescription')">
            <template #default="slotProps"><UiInput v-model="tagInput" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
        </div>

        <div v-else-if="currentStep === 'world'" class="grid gap-6 lg:grid-cols-2">
          <UiField class="lg:col-span-2" :label="t('projectNew.world')" :description="t('projectCreateFlow.fields.worldDescription')" required>
            <template #default="slotProps"><UiTextarea v-model="seed.world_background" :rows="7" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
          <UiField :label="t('projectNew.protagonist')" :description="t('projectCreateFlow.fields.protagonistDescription')" required>
            <template #default="slotProps"><UiTextarea v-model="seed.protagonist" :rows="6" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
          <UiField :label="t('projectNew.conflict')" :description="t('projectCreateFlow.fields.conflictDescription')" required>
            <template #default="slotProps"><UiTextarea v-model="seed.central_conflict" :rows="6" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
        </div>

        <div v-else-if="currentStep === 'voice'" class="grid gap-6 lg:grid-cols-2">
          <UiField :label="t('projectNew.style')" :description="t('projectCreateFlow.fields.styleDescription')" required>
            <template #default="slotProps"><UiTextarea v-model="seed.style" :rows="7" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
          <UiField :label="t('projectNew.avoid')" :description="t('projectCreateFlow.fields.avoidDescription')" required>
            <template #default="slotProps"><UiTextarea v-model="seed.taboos" :rows="7" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" /></template>
          </UiField>
        </div>

        <div v-else class="grid gap-6">
          <div class="border-y border-border">
            <dl class="divide-y divide-border">
              <div class="grid gap-2 py-4 sm:grid-cols-[10rem_minmax(0,1fr)]"><dt class="text-sm text-muted-foreground">{{ t('projectNew.name') }}</dt><dd class="break-words font-semibold">{{ seed.title }}</dd></div>
              <div class="grid gap-2 py-4 sm:grid-cols-[10rem_minmax(0,1fr)]"><dt class="text-sm text-muted-foreground">{{ t('projectNew.brief') }}</dt><dd class="break-words text-sm leading-6">{{ seed.one_sentence_core }}</dd></div>
              <div class="grid gap-2 py-4 sm:grid-cols-[10rem_minmax(0,1fr)]"><dt class="text-sm text-muted-foreground">{{ t('projectNew.protagonist') }}</dt><dd class="break-words text-sm leading-6">{{ seed.protagonist }}</dd></div>
              <div class="grid gap-2 py-4 sm:grid-cols-[10rem_minmax(0,1fr)]"><dt class="text-sm text-muted-foreground">{{ t('projectNew.tags') }}</dt><dd class="flex flex-wrap gap-2"><UiBadge v-for="tag in seed.tags" :key="tag" tone="muted">{{ tag }}</UiBadge><span v-if="seed.tags.length === 0">{{ t('common.emptyValue') }}</span></dd></div>
            </dl>
          </div>
          <UiInlineNotice tone="neutral" :title="t('projectCreateFlow.realChapterTitle')" :description="t('projectCreateFlow.realChapterDescription')" />
        </div>

        <div class="mt-8 flex flex-col-reverse gap-3 border-t border-border pt-5 sm:flex-row sm:items-center sm:justify-between">
          <UiButton v-if="stepIndex > 0" variant="ghost" class="w-full sm:w-auto" @click="goBack"><ArrowLeft class="h-4 w-4" />{{ t('actions.back') }}</UiButton>
          <span v-else />
          <UiButton v-if="!isLastStep" type="submit" class="w-full sm:w-auto">{{ t('projectCreateFlow.next') }}<ArrowRight class="h-4 w-4" /></UiButton>
          <UiButton v-else type="submit" :loading="creating" :loading-label="t('actions.createProject')" class="w-full sm:w-auto"><Check class="h-4 w-4" />{{ t('actions.createProject') }}</UiButton>
        </div>
      </form>

      <section v-if="created && destinations" ref="successTarget" tabindex="-1" class="mt-10 border-t-4 border-state-success-border bg-state-success-surface p-6 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background" aria-live="polite">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-state-success-foreground">{{ t('projectCreateFlow.createdEyebrow') }}</p>
        <h2 class="mt-2 text-2xl font-semibold text-state-success-foreground">{{ created.project.title }}</h2>
        <p class="mt-3 max-w-2xl text-sm leading-6 text-state-success-foreground">{{ t('projectCreateFlow.createdDescription') }}</p>
        <div class="mt-6 grid gap-4 md:grid-cols-2">
          <UiButton :to="destinations.storyBible" class="min-h-20 h-auto justify-start whitespace-normal px-5 py-4 text-left">
            <LibraryBig class="h-5 w-5 shrink-0" />
            <span><span class="block">{{ t('projectCreateFlow.completeStoryBible') }}</span><span class="mt-1 block text-xs font-normal opacity-75">{{ t('projectCreateFlow.completeStoryBibleDescription') }}</span></span>
          </UiButton>
          <UiButton :to="destinations.newChapter" variant="outline" class="min-h-20 h-auto justify-start whitespace-normal px-5 py-4 text-left">
            <FilePlus2 class="h-5 w-5 shrink-0" />
            <span><span class="block">{{ t('projectCreateFlow.createChapter') }}</span><span class="mt-1 block text-xs font-normal opacity-75">{{ t('projectCreateFlow.createChapterDescription') }}</span></span>
          </UiButton>
        </div>
      </section>
    </div>
  </div>
</template>
