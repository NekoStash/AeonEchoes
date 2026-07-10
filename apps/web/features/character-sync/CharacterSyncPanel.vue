<script setup lang="ts">
import { CheckCircle2, RefreshCw, TriangleAlert } from '@lucide/vue'
import type { StoryBible } from '~/entities/story-bible'
import { countSyncableCharacters, type CharacterSyncState } from './model'

const { t } = useI18n()
const props = defineProps<{
  bible: StoryBible
  state: CharacterSyncState
  disabled?: boolean
  error?: string
}>()
const emit = defineEmits<{
  sync: []
}>()

const syncableCount = computed(() => countSyncableCharacters(props.bible))
const pendingCount = computed(() => props.bible.characters.filter((character) => !character.entity_id).length)
</script>

<template>
  <section class="border-y-2 border-foreground bg-foreground px-5 py-6 text-background" :aria-label="t('projectOverview.characterSync.title')">
    <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <p class="text-xs font-bold uppercase tracking-[0.22em] text-background/65">{{ t('projectOverview.characterSync.eyebrow') }}</p>
        <h2 class="mt-2 font-serif text-2xl font-semibold">{{ t('projectOverview.characterSync.title') }}</h2>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-background/75">
          {{ t('projectOverview.characterSync.description', { syncable: syncableCount, pending: pendingCount }) }}
        </p>
      </div>
      <div class="flex flex-col items-stretch gap-3 sm:flex-row sm:items-center">
        <span v-if="state === 'synced'" class="inline-flex items-center gap-2 text-sm font-semibold">
          <CheckCircle2 class="h-4 w-4" aria-hidden="true" />
          {{ t('projectOverview.characterSync.synced') }}
        </span>
        <span v-else-if="state === 'failed'" class="inline-flex items-center gap-2 text-sm font-semibold text-red-200">
          <TriangleAlert class="h-4 w-4" aria-hidden="true" />
          {{ error || t('projectOverview.characterSync.failed') }}
        </span>
        <UiButton
          variant="secondary"
          :disabled="disabled || syncableCount === 0"
          :loading="state === 'syncing'"
          :loading-label="t('projectOverview.characterSync.syncing')"
          @click="emit('sync')"
        >
          <RefreshCw class="h-4 w-4" aria-hidden="true" />
          {{ state === 'syncing' ? t('projectOverview.characterSync.syncing') : t('projectOverview.characterSync.action') }}
        </UiButton>
      </div>
    </div>
  </section>
</template>
