<script setup lang="ts">
import { Plus, Trash2 } from '@lucide/vue'
import type { StoryBible } from '~/entities/story-bible'
import { createStoryBibleItemId } from './model'

const { t } = useI18n()
const props = defineProps<{
  modelValue: StoryBible
  disabled?: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: StoryBible]
}>()

const foreshadowStatuses: Array<StoryBible['foreshadows'][number]['status']> = ['planted', 'active', 'paid_off']
const foreshadowStatusOptions = computed(() => foreshadowStatuses.map((status) => ({
  label: t(`status.foreshadow.${status}`),
  value: status
})))
type RemovalTarget = { kind: 'theme' | 'worldRule' | 'character' | 'foreshadow'; index: number; description: string }
const removalTarget = ref<RemovalTarget | null>(null)
const removeConfirmOpen = computed({
  get: () => Boolean(removalTarget.value),
  set: (value: boolean) => {
    if (!value) removalTarget.value = null
  }
})

function update(mutator: (draft: StoryBible) => void) {
  const next = JSON.parse(JSON.stringify(props.modelValue)) as StoryBible
  mutator(next)
  emit('update:modelValue', next)
}

function addTheme() {
  update((draft) => draft.themes.push(''))
}

function requestRemoval(target: RemovalTarget) {
  removalTarget.value = target
}

function confirmRemoval() {
  const target = removalTarget.value
  if (!target) return
  update((draft) => {
    if (target.kind === 'theme') draft.themes.splice(target.index, 1)
    else if (target.kind === 'worldRule') draft.world_rules.splice(target.index, 1)
    else if (target.kind === 'character') draft.characters.splice(target.index, 1)
    else draft.foreshadows.splice(target.index, 1)
  })
  removalTarget.value = null
}

function removeTheme(index: number) {
  requestRemoval({ kind: 'theme', index, description: t('projectOverview.confirmRemove.theme', { number: index + 1 }) })
}

function addWorldRule() {
  update((draft) => draft.world_rules.push(''))
}

function removeWorldRule(index: number) {
  requestRemoval({ kind: 'worldRule', index, description: t('projectOverview.confirmRemove.worldRule', { number: index + 1 }) })
}

function addCharacter() {
  update((draft) => draft.characters.push({
    id: createStoryBibleItemId('character', draft.characters.map((character) => character.id)),
    name: '',
    role: '',
    desire: '',
    wound: '',
    secret: '',
    summary: ''
  }))
}

function updateCharacter(index: number, patch: Partial<StoryBible['characters'][number]>) {
  update((draft) => {
    const current = draft.characters[index]
    if (!current) throw new Error(`Character at index ${index} does not exist.`)
    draft.characters[index] = { ...current, ...patch }
  })
}

function removeCharacter(index: number) {
  const character = props.modelValue.characters[index]
  if (!character) throw new Error(`Character at index ${index} does not exist.`)
  requestRemoval({ kind: 'character', index, description: t('projectOverview.confirmRemove.character', { name: character.name || index + 1 }) })
}

function addForeshadow() {
  update((draft) => draft.foreshadows.push({
    id: createStoryBibleItemId('foreshadow', draft.foreshadows.map((item) => item.id)),
    title: '',
    planted_in: '',
    payoff_hint: '',
    status: 'planted'
  }))
}

function updateForeshadow(index: number, patch: Partial<StoryBible['foreshadows'][number]>) {
  update((draft) => {
    const current = draft.foreshadows[index]
    if (!current) throw new Error(`Foreshadow at index ${index} does not exist.`)
    draft.foreshadows[index] = { ...current, ...patch }
  })
}

function removeForeshadow(index: number) {
  const item = props.modelValue.foreshadows[index]
  if (!item) throw new Error(`Foreshadow at index ${index} does not exist.`)
  requestRemoval({ kind: 'foreshadow', index, description: t('projectOverview.confirmRemove.foreshadow', { name: item.title || index + 1 }) })
}
</script>

<template>
  <section class="space-y-10" role="region" :aria-label="t('projectOverview.storyBible')">
    <div class="grid gap-8 xl:grid-cols-[minmax(0,1.15fr)_minmax(18rem,.85fr)]">
      <div class="space-y-3">
        <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.storyBibleSections.foundationEyebrow') }}</p>
        <h2 class="font-serif text-3xl font-semibold tracking-tight">{{ t('projectOverview.storyBibleSections.foundationTitle') }}</h2>
        <p class="max-w-2xl text-sm leading-7 text-muted-foreground">{{ t('projectOverview.storyBibleSections.foundationDescription') }}</p>
        <label class="block space-y-2 pt-3">
          <span class="field-label">{{ t('projectOverview.fields.premise') }}</span>
          <UiTextarea
            :model-value="modelValue.premise"
            :disabled="disabled"
            :rows="8"
            :placeholder="t('projectOverview.placeholders.premise')"
            class="rounded-none border-x-0 border-t-0 bg-transparent px-0 text-base leading-8"
            @update:model-value="update((draft) => { draft.premise = $event })"
          />
        </label>
      </div>

      <div class="border-l-4 border-foreground bg-muted/45 p-5">
        <div class="flex items-center justify-between gap-3">
          <h3 class="font-serif text-xl font-semibold">{{ t('projectOverview.fields.themes') }}</h3>
          <UiButton size="sm" variant="outline" :disabled="disabled" @click="addTheme">
            <Plus class="h-4 w-4" aria-hidden="true" />
            {{ t('actions.add') }}
          </UiButton>
        </div>
        <p v-if="modelValue.themes.length === 0" class="mt-5 text-sm leading-6 text-muted-foreground">{{ t('projectOverview.empty.themes') }}</p>
        <div v-else class="mt-5 space-y-3">
          <div v-for="(_theme, index) in modelValue.themes" :key="index" class="flex items-center gap-2">
            <UiInput
              :model-value="modelValue.themes[index]"
              :disabled="disabled"
              :aria-label="t('projectOverview.fields.themeNumber', { number: index + 1 })"
              class="rounded-none border-x-0 border-t-0 bg-transparent"
              @update:model-value="update((draft) => { draft.themes[index] = $event })"
            />
            <UiButton
              size="icon"
              variant="destructive"
              :disabled="disabled"
              :icon-label="t('projectOverview.actions.removeThemeNumber', { number: index + 1 })"
              @click="removeTheme(index)"
            >
              <Trash2 class="h-4 w-4" aria-hidden="true" />
            </UiButton>
          </div>
        </div>
      </div>
    </div>

    <div class="border-t-2 border-foreground pt-8">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.storyBibleSections.worldEyebrow') }}</p>
          <h2 class="mt-2 font-serif text-3xl font-semibold tracking-tight">{{ t('projectOverview.worldRules') }}</h2>
        </div>
        <UiButton variant="outline" :disabled="disabled" @click="addWorldRule">
          <Plus class="h-4 w-4" aria-hidden="true" />
          {{ t('projectOverview.actions.addWorldRule') }}
        </UiButton>
      </div>
      <p v-if="modelValue.world_rules.length === 0" class="mt-6 border-y border-border py-6 text-sm text-muted-foreground">{{ t('projectOverview.empty.worldRules') }}</p>
      <ol v-else class="mt-6 divide-y divide-border border-y border-border">
        <li v-for="(_rule, index) in modelValue.world_rules" :key="index" class="grid gap-4 py-5 sm:grid-cols-[3rem_minmax(0,1fr)_auto] sm:items-start">
          <span class="font-serif text-3xl text-muted-foreground">{{ String(index + 1).padStart(2, '0') }}</span>
          <UiTextarea
            :model-value="modelValue.world_rules[index]"
            :disabled="disabled"
            :rows="3"
            :aria-label="t('projectOverview.fields.worldRuleNumber', { number: index + 1 })"
            class="rounded-none border-0 bg-transparent p-0 shadow-none"
            @update:model-value="update((draft) => { draft.world_rules[index] = $event })"
          />
          <UiButton
            size="icon"
            variant="destructive"
            :disabled="disabled"
            :icon-label="t('projectOverview.actions.removeWorldRuleNumber', { number: index + 1 })"
            @click="removeWorldRule(index)"
          >
            <Trash2 class="h-4 w-4" aria-hidden="true" />
          </UiButton>
        </li>
      </ol>
    </div>

    <div class="border-t-2 border-foreground pt-8">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.storyBibleSections.charactersEyebrow') }}</p>
          <h2 class="mt-2 font-serif text-3xl font-semibold tracking-tight">{{ t('projectOverview.characters') }}</h2>
        </div>
        <UiButton variant="outline" :disabled="disabled" @click="addCharacter">
          <Plus class="h-4 w-4" aria-hidden="true" />
          {{ t('projectOverview.actions.addCharacter') }}
        </UiButton>
      </div>
      <p v-if="modelValue.characters.length === 0" class="mt-6 border-y border-border py-6 text-sm text-muted-foreground">{{ t('projectOverview.empty.characters') }}</p>
      <div v-else class="mt-6 divide-y divide-border border-y border-border">
        <article v-for="(character, index) in modelValue.characters" :key="character.id || index" class="grid gap-5 py-7 lg:grid-cols-[11rem_minmax(0,1fr)_auto]">
          <div>
            <span class="font-serif text-5xl text-muted-foreground/55">{{ String(index + 1).padStart(2, '0') }}</span>
            <p class="mt-2 text-xs font-bold uppercase tracking-[0.18em] text-muted-foreground">
              {{ character.entity_id ? t('projectOverview.characterSync.realCharacter') : t('projectOverview.characterSync.pending') }}
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <label class="space-y-2">
              <span class="field-label">{{ t('projectOverview.fields.characterName') }}</span>
              <UiInput :model-value="character.name" :disabled="disabled" @update:model-value="updateCharacter(index, { name: $event })" />
            </label>
            <label class="space-y-2">
              <span class="field-label">{{ t('projectOverview.fields.characterRole') }}</span>
              <UiInput :model-value="character.role" :disabled="disabled" @update:model-value="updateCharacter(index, { role: $event })" />
            </label>
            <label class="space-y-2">
              <span class="field-label">{{ t('projectOverview.fields.characterDesire') }}</span>
              <UiTextarea :model-value="character.desire" :disabled="disabled" :rows="3" @update:model-value="updateCharacter(index, { desire: $event })" />
            </label>
            <label class="space-y-2">
              <span class="field-label">{{ t('projectOverview.fields.characterWound') }}</span>
              <UiTextarea :model-value="character.wound" :disabled="disabled" :rows="3" @update:model-value="updateCharacter(index, { wound: $event })" />
            </label>
            <label class="space-y-2 md:col-span-2">
              <span class="field-label">{{ t('projectOverview.fields.characterSecret') }}</span>
              <UiInput :model-value="character.secret" :disabled="disabled" @update:model-value="updateCharacter(index, { secret: $event })" />
            </label>
            <label class="space-y-2 md:col-span-2">
              <span class="field-label">{{ t('projectOverview.fields.characterSummary') }}</span>
              <UiTextarea :model-value="character.summary" :disabled="disabled" :rows="3" @update:model-value="updateCharacter(index, { summary: $event })" />
            </label>
          </div>
          <UiButton
            size="icon"
            variant="destructive"
            :disabled="disabled"
            :icon-label="t('projectOverview.actions.removeCharacterNamed', { name: character.name || index + 1 })"
            @click="removeCharacter(index)"
          >
            <Trash2 class="h-4 w-4" aria-hidden="true" />
          </UiButton>
        </article>
      </div>
    </div>

    <div class="border-t-2 border-foreground pt-8">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.storyBibleSections.foreshadowEyebrow') }}</p>
          <h2 class="mt-2 font-serif text-3xl font-semibold tracking-tight">{{ t('projectOverview.foreshadowing') }}</h2>
        </div>
        <UiButton variant="outline" :disabled="disabled" @click="addForeshadow">
          <Plus class="h-4 w-4" aria-hidden="true" />
          {{ t('projectOverview.actions.addForeshadow') }}
        </UiButton>
      </div>
      <p v-if="modelValue.foreshadows.length === 0" class="mt-6 border-y border-border py-6 text-sm text-muted-foreground">{{ t('projectOverview.empty.foreshadowing') }}</p>
      <div v-else class="mt-6 divide-y divide-border border-y border-border">
        <article v-for="(item, index) in modelValue.foreshadows" :key="item.id || index" class="grid gap-4 py-6 lg:grid-cols-[minmax(0,1fr)_12rem_auto]">
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="space-y-2 sm:col-span-2">
              <span class="field-label">{{ t('projectOverview.fields.foreshadowTitle') }}</span>
              <UiInput :model-value="item.title" :disabled="disabled" @update:model-value="updateForeshadow(index, { title: $event })" />
            </label>
            <label class="space-y-2">
              <span class="field-label">{{ t('projectOverview.fields.plantedIn') }}</span>
              <UiInput :model-value="item.planted_in" :disabled="disabled" @update:model-value="updateForeshadow(index, { planted_in: $event })" />
            </label>
            <label class="space-y-2">
              <span class="field-label">{{ t('projectOverview.fields.payoffHint') }}</span>
              <UiInput :model-value="item.payoff_hint" :disabled="disabled" @update:model-value="updateForeshadow(index, { payoff_hint: $event })" />
            </label>
          </div>
          <label class="space-y-2">
            <span class="field-label">{{ t('projectOverview.fields.foreshadowStatus') }}</span>
            <UiSelect
              :model-value="item.status"
              :disabled="disabled"
              :aria-label="t('projectOverview.fields.foreshadowStatus')"
              :options="foreshadowStatusOptions"
              @update:model-value="updateForeshadow(index, { status: $event as StoryBible['foreshadows'][number]['status'] })"
            />
          </label>
          <UiButton
            size="icon"
            variant="destructive"
            :disabled="disabled"
            :icon-label="t('projectOverview.actions.removeForeshadowNamed', { name: item.title || index + 1 })"
            @click="removeForeshadow(index)"
          >
            <Trash2 class="h-4 w-4" aria-hidden="true" />
          </UiButton>
        </article>
      </div>
    </div>
    <UiConfirm
      v-model:open="removeConfirmOpen"
      :title="t('actions.delete')"
      :description="removalTarget?.description || ''"
      tone="danger"
      @confirm="confirmRemoval"
    />
  </section>
</template>
